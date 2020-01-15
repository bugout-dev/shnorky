package executions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/user"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerMount "github.com/docker/docker/api/types/mount"
	docker "github.com/docker/docker/client"
	"github.com/google/uuid"

	"github.com/simiotics/simplex/builds"
	"github.com/simiotics/simplex/components"
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
func GenerateExecutionMetadata(build builds.BuildMetadata, flowID string) (ExecutionMetadata, error) {
	if build.ID == "" {
		return ExecutionMetadata{}, ErrEmptyBuildID
	}
	if build.ComponentID == "" {
		return ExecutionMetadata{}, builds.ErrEmptyComponentID
	}

	createdAt := time.Now()

	executionID, err := uuid.NewRandom()
	if err != nil {
		return ExecutionMetadata{}, err
	}

	return ExecutionMetadata{ID: executionID.String(), BuildID: build.ID, ComponentID: build.ComponentID, CreatedAt: createdAt, FlowID: flowID}, nil
}

// Execute runs a container corresponding to the given build of the given component.
func Execute(
	ctx context.Context,
	db *sql.DB,
	dockerClient *docker.Client,
	buildID string,
	flowID string,
	mounts map[string]string,
) (ExecutionMetadata, error) {
	var inverseMounts map[string]string
	for source, target := range mounts {
		inverseMounts[target] = source
	}

	buildMetadata, err := builds.SelectBuildByID(db, buildID)
	if err != nil {
		return ExecutionMetadata{}, fmt.Errorf("Error retrieving build metadata for build ID (%s) from state database: %s", buildID, err.Error())
	}

	executionMetadata, err := GenerateExecutionMetadata(buildMetadata, flowID)
	if err != nil {
		return ExecutionMetadata{}, fmt.Errorf("Error generating execution metadata for build (%s): %s", buildMetadata.ID, err.Error())
	}

	componentMetadata, err := components.SelectComponentByID(db, buildMetadata.ComponentID)
	if err != nil {
		return executionMetadata, fmt.Errorf("Error retrieving component metadata for component ID (%s) from state database: %s", buildMetadata.ComponentID, err.Error())
	}

	specFile, err := os.Open(componentMetadata.SpecificationPath)
	defer specFile.Close()
	specification, err := components.ReadSingleSpecification(specFile)
	if err != nil {
		return executionMetadata, fmt.Errorf("Could not open specification file (%s): %s", componentMetadata.SpecificationPath, err.Error())
	}

	containerConfig := &dockerContainer.Config{
		Cmd:   specification.Run.Cmd,
		Image: buildMetadata.ID,
	}

	if specification.Run.User == "${CURRENT_USER}" {
		targetUser, err := user.Current()
		if err != nil {
			return executionMetadata, fmt.Errorf("Error retrieving information about current user (as per $CURRENT value set in component specification: %s", err.Error())
		}
		containerConfig.User = targetUser.Uid
	} else if len(specification.Run.User) >= 5 && specification.Run.User[:5] == "name:" {
		targetUser, err := user.Lookup(specification.Run.User[5:])
		if err != nil {
			return executionMetadata, fmt.Errorf("Error looking up user with given username (%s): %s", specification.Run.User[5:], err)
		}
		containerConfig.User = targetUser.Uid
	} else {
		containerConfig.User = specification.Run.User
	}

	hostConfig := &dockerContainer.HostConfig{
		Mounts: make([]dockerMount.Mount, len(inverseMounts)),
	}

	currentMount := 0
	for _, mountpoint := range specification.Run.Mountpoints {
		source, ok := inverseMounts[mountpoint.Mountpoint]
		if mountpoint.Required && !ok {
			return executionMetadata, fmt.Errorf("No mount provided for required mountpoint: %s", mountpoint.Mountpoint)
		}

		if ok {
			if currentMount > len(inverseMounts) {
				return executionMetadata, errors.New("Too many mounts in host configuration")
			}
			hostConfig.Mounts[currentMount] = dockerMount.Mount{
				Type:   components.ValidMountTypes[mountpoint.MountType],
				Source: source,
				Target: mountpoint.Mountpoint,
			}

			currentMount++
		}
	}

	response, err := dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, executionMetadata.ID)
	if err != nil {
		return executionMetadata, fmt.Errorf("Error creating container for build (%s): %s", buildMetadata.ID, err.Error())
	}
	if response.ID != executionMetadata.ID {
		return executionMetadata, fmt.Errorf("Container ID assigned by docker daemon (%s) did not match execution metadata ID (%s): %s", response.ID, executionMetadata.ID, err.Error())
	}

	err = dockerClient.ContainerStart(ctx, response.ID, dockerTypes.ContainerStartOptions{})
	if err != nil {
		return executionMetadata, fmt.Errorf("Error starting container (ID=%s): %s", response.ID, err.Error())
	}

	return executionMetadata, nil
}
