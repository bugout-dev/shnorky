package components

import (
	"archive/tar"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
)

// DockerImagePrefix is the prefix that simplex attaches to each docker image name
var DockerImagePrefix = "simplex/"

// ErrEmptyComponentID signifies that a caller attempted to create build or execution metadata in
// which the ComponentID string was the empty string
var ErrEmptyComponentID = errors.New("ComponentID must be a non-empty string")

// BuildMetadata - the metadata about a component build that gets stored in the state database
type BuildMetadata struct {
	ID          string    `json:"id"`
	ComponentID string    `json:"component_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// GenerateBuildMetadata creates a BuildMetadata instance representing a fresh (as yet unbuilt)
// build of the component specified by the given componentID.
func GenerateBuildMetadata(componentID string) (BuildMetadata, error) {
	if componentID == "" {
		return BuildMetadata{}, ErrEmptyComponentID
	}
	createdAt := time.Now()
	buildID := fmt.Sprintf("%s%s:%d", DockerImagePrefix, componentID, createdAt.Unix())
	return BuildMetadata{ID: buildID, ComponentID: componentID, CreatedAt: createdAt}, nil
}

// CreateBuild creates a new build for the component with the given componentID
func CreateBuild(ctx context.Context, db *sql.DB, dockerClient *docker.Client, outstream io.Writer, componentID string) (BuildMetadata, error) {
	componentMetadata, err := SelectComponentByID(db, componentID)
	if err != nil {
		return BuildMetadata{}, err
	}

	buildMetadata, err := GenerateBuildMetadata(componentMetadata.ID)
	if err != nil {
		return BuildMetadata{}, err
	}

	specFile, err := os.Open(componentMetadata.SpecificationPath)
	if err != nil {
		return buildMetadata, fmt.Errorf("Could not open specification file (%s): %s", componentMetadata.SpecificationPath, err.Error())
	}
	defer specFile.Close()

	specification, err := ReadSingleSpecification(specFile)
	if err != nil {
		return buildMetadata, fmt.Errorf("Could not parse specification from specification file (%s): %s", componentMetadata.SpecificationPath, err.Error())
	}

	context := filepath.Join(componentMetadata.ComponentPath, specification.Build.Context)

	tarfile, err := ioutil.TempFile("", "*.tar")
	if err != nil {
		return buildMetadata, fmt.Errorf("Error creating context tarfile: %s", err)
	}
	defer os.Remove(tarfile.Name())

	tarWriter := tar.NewWriter(tarfile)

	err = filepath.Walk(context, func(targetPath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if info.IsDir() {
			return nil
		}

		target, tarErr := os.Open(targetPath)
		if tarErr != nil {
			return tarErr
		}
		defer target.Close()

		contextPath, tarErr := filepath.Rel(context, targetPath)
		if tarErr != nil {
			return tarErr
		}

		header := &tar.Header{
			Name:    contextPath,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}

		tarErr = tarWriter.WriteHeader(header)
		if tarErr != nil {
			return tarErr
		}

		_, tarErr = io.Copy(tarWriter, target)

		return tarErr
	})
	if err != nil {
		return buildMetadata, fmt.Errorf("Error building context tarfile: %s", err.Error())
	}

	err = tarWriter.Close()
	if err != nil {
		return buildMetadata, fmt.Errorf("Error closing context tarfile: %s", err.Error())
	}

	buildContext, err := os.Open(tarfile.Name())
	if err != nil {
		return buildMetadata, fmt.Errorf("Could not reopen context tarfile: %s", err.Error())
	}
	defer buildContext.Close()

	tags := []string{buildMetadata.ID}
	imageIDComponents := strings.Split(buildMetadata.ID, ":")
	if len(imageIDComponents) > 1 {
		imageIDComponents[len(imageIDComponents)-1] = "latest"
		tags = append(tags, strings.Join(imageIDComponents, ":"))
	}
	buildOptions := dockerTypes.ImageBuildOptions{
		Tags:       tags,
		Dockerfile: specification.Build.Dockerfile,
		// Setting Remove to true means that intermediate containers for the build will be removed
		// on a successful build.
		Remove: true,
	}

	response, err := dockerClient.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		return buildMetadata, fmt.Errorf("Error building image: %s", err.Error())
	}
	defer response.Body.Close()
	io.Copy(outstream, response.Body)

	err = InsertBuild(db, buildMetadata)
	if err != nil {
		return buildMetadata, fmt.Errorf("Error inserting build metadata into state database: %s", err.Error())
	}

	return buildMetadata, nil
}

// ListBuilds streams builds one by one from the given state database into the given builds channel.
// This function closes the builds channel when it is finished.
func ListBuilds(db *sql.DB, builds chan<- BuildMetadata, componentID string) error {
	defer close(builds)

	var rows *sql.Rows
	var err error
	if componentID != "" {
		rows, err = db.Query(selectBuildsByComponentID, componentID)
	} else {
		rows, err = db.Query(selectBuilds)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	var id, rowComponentID string
	var createdAt int64

	for rows.Next() {
		err = rows.Scan(&id, &rowComponentID, &createdAt)
		if err != nil {
			return err
		}

		builds <- BuildMetadata{
			ID:          id,
			ComponentID: rowComponentID,
			CreatedAt:   time.Unix(createdAt, 0),
		}
	}

	return nil
}
