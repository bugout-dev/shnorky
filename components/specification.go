package components

import (
	"encoding/json"
	"io"
)

// ComponentSpecification - struct specifying how a component of a simplex data processing flow
// should be built and executed
type ComponentSpecification struct {
	Build BuildSpecification `json:"build"`
	Run   RunSpecification   `json:"run"`
}

// BuildSpecification - struct specifying how a component of a simplex data processing flow should
// be built; all paths are assumed to be paths relative to the component path (i.e. the directory
// containing the implementation of the component)
type BuildSpecification struct {
	// Path to context directory (used to build docker image)
	Context string `json:"context"`

	// Path to Dockerfile to be used to build the component - should be relative to the context
	// path
	Dockerfile string `json:"Dockerfile"`
}

// RunSpecification - struct specifying how a component of a simplex data processing flow should be
// executed
type RunSpecification struct {
	// Mapping of environment variable names to values to be set in component container at runtime
	Env map[string]string `json:"env"`

	// Entrypoint override for containers representing this component
	Entrypoint []string `json:"entrypoint"`

	// Command to be invoked when starting component container at runtime
	Cmd []string `json:"cmd"`

	// Mountpoint specify paths inside each container (for this component) that can accept data
	Mountpoints []MountSpecification `json:"mountpoints"`

	// User specifies the uid (and optionally guid that the container should run as) - format the
	// string as "<uid>:<guid>".
	// Special values:
	// "" - container runs as root
	// "${CURRENT_USER}" - container runs as the user executing the simplex process
	// "name:<username>" - container runs as the user with the given username
	User string `json:"user"`
}

// MountSpecification - specifies a mount point within a simplex component, how it should be mounted
// on the container side, and whether or not it is required to be mounted at runtime
type MountSpecification struct {
	// See documentation of mount type here: https://godoc.org/github.com/docker/docker/api/types/mount#Type
	// Can be one of "bind", "volume", "tmpfs", "npipe"
	// TODO(nkashy1): Check the value of MountType when parsing specification. This is where JSONSchema
	// may be a good idea. However, rather than integrating JSONSchema right away, the best fix is
	// probably just to make explicit checks for this kind of thing in the `ReadSingleSpecification`
	// function.
	MountType  string `json:"mount_type"`
	Mountpoint string `json:"mountpoint"`
	ReadOnly   bool   `json:"read_only"`
	Required   bool   `json:"required"`
}

// ReadSingleSpecification reads a single ComponentSpecification JSON document and returns the
// corresponding ComponentSpecification struct. It returns an error if there was an issue parsing
// the specification into the struct.
func ReadSingleSpecification(reader io.Reader) (ComponentSpecification, error) {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	var specification ComponentSpecification
	err := dec.Decode(&specification)
	return specification, err
}
