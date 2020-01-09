package builds

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	dockerContainer "github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	"github.com/google/uuid"

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
	FlowID      string    `json:"flow_id"`
	CreatedAt   time.Time `json:"created_at"`
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

// ExecuteBuild runs a container corresponding to the given build of the given component.
func ExecuteBuild(
	ctx context.Context,
	db *sql.DB,
	dockerClient *docker.Client,
	buildID string,
	flowID string,
) (ExecutionMetadata, error) {
	buildMetadata, err := SelectBuildByID(db, buildID)
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

	_, err = dockerClient.ContainerCreate(ctx, containerConfig, nil, nil, executionMetadata.ID)
	if err != nil {
		return executionMetadata, fmt.Errorf("Error creating container for build (%s): %s", buildMetadata.ID, err.Error())
	}

	return executionMetadata, nil
}
