package components

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerMount "github.com/docker/docker/api/types/mount"
	docker "github.com/docker/docker/client"
	"github.com/google/uuid"
)

// ErrEmptyBuildID signifies that a caller attempted to create execution metadata in which the
// BuildID string was the empty string
var ErrEmptyBuildID = errors.New("BuildID must be a non-empty string")

// ExecutionMetadata - the metadata about a component build execution that gets stored in the state database
type ExecutionMetadata struct {
	ID          string    `json:"id"`
	BuildID     string    `json:"build_id"`
	ComponentID string    `json:"component_id"`
	CreatedAt   time.Time `json:"created_at"`
	FlowID      string    `json:"flow_id"`
}

// GenerateExecutionMetadata creates an ExecutionMetadata instance representing a potential
// execution of the build specified by the given build metadata.
func GenerateExecutionMetadata(build BuildMetadata, flowID string) (ExecutionMetadata, error) {
	if build.ID == "" {
		return ExecutionMetadata{}, ErrEmptyBuildID
	}
	if build.ComponentID == "" {
		return ExecutionMetadata{}, ErrEmptyComponentID
	}

	createdAt := time.Now()

	executionID, err := uuid.NewRandom()
	if err != nil {
		return ExecutionMetadata{}, err
	}

	return ExecutionMetadata{ID: executionID.String(), BuildID: build.ID, ComponentID: build.ComponentID, CreatedAt: createdAt, FlowID: flowID}, nil
}

// Execute runs a container corresponding to the given build of the given component.
// TODO(nkashy1): Maybe take build metadata instead of build ID? This will reduce the number of
// database lookups that happen in flow execution.
func Execute(
	ctx context.Context,
	db *sql.DB,
	dockerClient *docker.Client,
	buildID string,
	flowID string,
	mounts []MountConfiguration,
	env map[string]string,
) (ExecutionMetadata, error) {
	inverseMounts := map[string]int{}
	for i, mountConfig := range mounts {
		inverseMounts[mountConfig.Target] = i
	}

	buildMetadata, err := SelectBuildByID(db, buildID)
	if err != nil {
		return ExecutionMetadata{}, fmt.Errorf("Error retrieving build metadata for build ID (%s) from state database: %s", buildID, err.Error())
	}

	executionMetadata, err := GenerateExecutionMetadata(buildMetadata, flowID)
	if err != nil {
		return ExecutionMetadata{}, fmt.Errorf("Error generating execution metadata for build (%s): %s", buildMetadata.ID, err.Error())
	}

	componentMetadata, err := SelectComponentByID(db, buildMetadata.ComponentID)
	if err != nil {
		return executionMetadata, fmt.Errorf("Error retrieving component metadata for component ID (%s) from state database: %s", buildMetadata.ComponentID, err.Error())
	}

	specFile, err := os.Open(componentMetadata.SpecificationPath)
	defer specFile.Close()
	rawSpecification, err := ReadSingleSpecification(specFile)
	if err != nil {
		return executionMetadata, fmt.Errorf("Could not open specification file (%s): %s", componentMetadata.SpecificationPath, err.Error())
	}

	specification, err := MaterializeComponentSpecification(rawSpecification)
	if err != nil {
		return executionMetadata, fmt.Errorf("Could not materialize component specification: %s", err.Error())
	}

	containerConfig := &dockerContainer.Config{
		Cmd:   specification.Run.Cmd,
		Image: buildMetadata.ID,
	}

	containerConfig.Env = make([]string, len(specification.Run.Env))
	i := 0
	// finalEnv is formed by merging the env argument to this function over the env specified
	// in the component specification. This determines the environment variables that get set
	// for the execution container.
	finalEnv := map[string]string{}
	for key, value := range specification.Run.Env {
		finalEnv[key] = value
	}
	for key, value := range env {
		finalEnv[key] = value
	}
	for key, value := range finalEnv {
		containerConfig.Env[i] = fmt.Sprintf("%s=%s", key, value)
		i++
	}

	containerConfig.User = specification.Run.User

	hostConfig := &dockerContainer.HostConfig{
		Mounts: make([]dockerMount.Mount, len(inverseMounts)),
	}

	currentMount := 0
	for _, mountpoint := range specification.Run.Mountpoints {
		mountsIndex, ok := inverseMounts[mountpoint.Mountpoint]
		if mountpoint.Required && !ok {
			return executionMetadata, fmt.Errorf("No mount provided for required mountpoint: %s", mountpoint.Mountpoint)
		}

		if ok {
			if currentMount > len(inverseMounts) {
				return executionMetadata, errors.New("Too many mounts in host configuration")
			}
			mountMethod := ValidMountMethods[mounts[mountsIndex].Method]
			mountSource := mounts[mountsIndex].Source
			hostConfig.Mounts[currentMount] = dockerMount.Mount{
				Type:   mountMethod,
				Source: mountSource,
				Target: mountpoint.Mountpoint,
			}

			currentMount++
		}
	}

	response, err := dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, executionMetadata.ID)
	if err != nil {
		return executionMetadata, fmt.Errorf("Error creating container for build (%s): %s", buildMetadata.ID, err.Error())
	}

	err = InsertExecution(db, executionMetadata)
	if err != nil {
		return executionMetadata, fmt.Errorf("Error inserting execution into state database: %s", err.Error())
	}

	err = dockerClient.ContainerStart(ctx, response.ID, dockerTypes.ContainerStartOptions{})
	if err != nil {
		return executionMetadata, fmt.Errorf("Error starting container (ID=%s): %s", response.ID, err.Error())
	}

	return executionMetadata, nil
}
