package flows

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	docker "github.com/docker/docker/client"

	"github.com/simiotics/simplex/builds"
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

// AddFlow registers a flow (by metadata) against a simplex state database. It validates the
// specification at the given path first.
// This is the handler for `simplex flows add`
func AddFlow(db *sql.DB, id, specificationPath string) (FlowMetadata, error) {
	specFile, err := os.Open(specificationPath)
	if err != nil {
		return FlowMetadata{}, fmt.Errorf("Error opening specification file (%s): %s", specificationPath, err.Error())
	}
	_, err = ReadSingleSpecification(specFile)
	if err != nil {
		return FlowMetadata{}, fmt.Errorf("Error reading specification (%s): %s", specificationPath, err.Error())
	}

	metadata, err := GenerateFlowMetadata(id, specificationPath)
	if err != nil {
		return metadata, err
	}

	err = InsertFlow(db, metadata)

	return metadata, err
}

// Build - Builds images for each component of a given flow
func Build(ctx context.Context, db *sql.DB, dockerClient *docker.Client, outstream io.Writer, flowID string) ([]builds.BuildMetadata, error) {
	flow, err := SelectFlowByID(db, flowID)
	if err != nil {
		return []builds.BuildMetadata{}, err
	}

	specFile, err := os.Open(flow.SpecificationPath)
	if err != nil {
		return []builds.BuildMetadata{}, err
	}

	specification, err := ReadSingleSpecification(specFile)
	if err != nil {
		return []builds.BuildMetadata{}, err
	}

	buildInfo := make([]builds.BuildMetadata, len(specification.Steps))
	componentBuilds := map[string]builds.BuildMetadata{}

	currentComponent := 0
	for _, component := range specification.Steps {
		_, ok := componentBuilds[component]
		if ok {
			continue
		}

		buildMetadata, err := builds.CreateBuild(ctx, db, dockerClient, outstream, component)
		if err != nil {
			return buildInfo, err
		}

		buildInfo[currentComponent] = buildMetadata
		currentComponent++
	}

	buildInfo = buildInfo[:currentComponent]
	return buildInfo, nil
}
