package flows

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	docker "github.com/docker/docker/client"

	"github.com/simiotics/shnorky/components"
)

// ErrEmptyID signifies that a caller attempted to create component metadata in which the ID string
// was the empty string
var ErrEmptyID = errors.New("ID must be a non-empty string")

// ErrEmptySpecificationPath signifies that a caller attempted to create component metadata in which
// the SpecificationPath string was the empty string
var ErrEmptySpecificationPath = errors.New("SpecificationPath must be a non-empty string")

// FlowMetadata - the metadata about a flow that gets stored in the state database
type FlowMetadata struct {
	ID                string    `json:"id"`
	SpecificationPath string    `json:"specification_path"`
	CreatedAt         time.Time `json:"created_at"`
}

// GenerateFlowMetadata creates a FlowMetadata instance from the specified parameters, applying
// defaults as required and reasonable.
func GenerateFlowMetadata(id, specificationPath string) (FlowMetadata, error) {
	if id == "" {
		return FlowMetadata{}, ErrEmptyID
	}

	if specificationPath == "" {
		return FlowMetadata{}, ErrEmptySpecificationPath
	}

	createdAt := time.Now()

	metadata := FlowMetadata{ID: id, SpecificationPath: specificationPath, CreatedAt: createdAt}

	return metadata, nil
}

// AddFlow registers a flow (by metadata) against a shnorky state database. It validates the
// specification at the given path first.
// This is the handler for `shnorky flows add`
func AddFlow(db *sql.DB, id, specificationPath string) (FlowMetadata, error) {
	absoluteSpecificationPath, err := filepath.Abs(specificationPath)
	if err != nil {
		return FlowMetadata{}, err
	}

	specFile, err := os.Open(absoluteSpecificationPath)
	if err != nil {
		return FlowMetadata{}, fmt.Errorf("Error opening specification file (%s): %s", absoluteSpecificationPath, err.Error())
	}
	_, err = ReadSingleSpecification(specFile)
	if err != nil {
		return FlowMetadata{}, fmt.Errorf("Error reading specification (%s): %s", absoluteSpecificationPath, err.Error())
	}

	metadata, err := GenerateFlowMetadata(id, absoluteSpecificationPath)
	if err != nil {
		return metadata, err
	}

	err = InsertFlow(db, metadata)

	return metadata, err
}

// Build - Builds images for each component of a given flow
func Build(ctx context.Context, db *sql.DB, dockerClient *docker.Client, outstream io.Writer, flowID string) (map[string]components.BuildMetadata, error) {
	flow, err := SelectFlowByID(db, flowID)
	if err != nil {
		return map[string]components.BuildMetadata{}, err
	}

	specFile, err := os.Open(flow.SpecificationPath)
	if err != nil {
		return map[string]components.BuildMetadata{}, err
	}

	specification, err := ReadSingleSpecification(specFile)
	if err != nil {
		return map[string]components.BuildMetadata{}, err
	}

	componentBuilds := map[string]components.BuildMetadata{}

	for _, component := range specification.Steps {
		_, ok := componentBuilds[component]
		if ok {
			continue
		}

		buildMetadata, err := components.CreateBuild(ctx, db, dockerClient, outstream, component)
		if err != nil {
			return componentBuilds, err
		}

		componentBuilds[component] = buildMetadata
	}

	return componentBuilds, nil
}

// Execute - Executes the given builds of each step in a workflow in an order which respects the
// dependencies between steps
func Execute(
	ctx context.Context,
	db *sql.DB,
	dockerClient *docker.Client,
	flowID string,
) (map[string]components.ExecutionMetadata, error) {
	flow, err := SelectFlowByID(db, flowID)
	if err != nil {
		return map[string]components.ExecutionMetadata{}, err
	}

	specFile, err := os.Open(flow.SpecificationPath)
	if err != nil {
		return map[string]components.ExecutionMetadata{}, err
	}

	specification, err := ReadSingleSpecification(specFile)
	if err != nil {
		return map[string]components.ExecutionMetadata{}, err
	}

	// buildIDs maps steps to build IDs
	buildIDs := map[string]string{}
	for step, componentID := range specification.Steps {
		buildID, err := components.SelectMostRecentBuildForComponent(db, componentID)
		if err != nil {
			return map[string]components.ExecutionMetadata{}, err
		}
		buildIDs[step] = buildID.ID
	}

	stages, err := CalculateStages(specification)
	if err != nil {
		return map[string]components.ExecutionMetadata{}, err
	}

	componentExecutions := map[string]components.ExecutionMetadata{}
	for _, stage := range stages {
		stepExecutions := map[string]components.ExecutionMetadata{}
		for _, step := range stage {
			executionMetadata, err := components.Execute(ctx, db, dockerClient, buildIDs[step], flowID, specification.Mounts[step], specification.Env[step])
			if err != nil {
				return componentExecutions, err
			}
			componentExecutions[step] = executionMetadata
			stepExecutions[step] = executionMetadata
		}

		for step, executionMetadata := range stepExecutions {
			for {
				info, err := dockerClient.ContainerInspect(ctx, executionMetadata.ID)
				if err != nil {
					return componentExecutions, fmt.Errorf("Error executing step (%s): %s", step, err.Error())
				}
				if info.State.Running {
					continue
				} else if info.State.ExitCode == 0 {
					break
				} else {
					return componentExecutions, fmt.Errorf("Container (%s) for step (%s) exited with non-zero code: %d", info.ID, step, info.State.ExitCode)
				}
			}
		}
	}

	return componentExecutions, nil
}
