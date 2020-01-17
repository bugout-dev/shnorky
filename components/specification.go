package components

import (
	"encoding/json"
	"errors"
	"io"

	dockerMount "github.com/docker/docker/api/types/mount"
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
	// Special keys:
	//
	// Special values:
	// "env:<VARIABLE_NAME>" - specifies that the value of the environment variable denoted by
	// VARIABLE_NAME in the simplex process should be interpolated into the specification; if the
	// environment variable is not set in the simplex process, it will use the empty string "" as
	// the value
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
// TODO(nkashy1): It does not make sense to specify this kind of mount type in the
// MountSpecification - the mount type (e.g. whether it is a bind mount or a docker volume mount) is
// the responsibility of the caller. What does make sense is for MountType to specify the type of
// filesystem object that the mountpoint expects (e.g. file vs. directory)
type MountSpecification struct {
	// See documentation of mount type here: https://godoc.org/github.com/docker/docker/api/types/mount#Type
	// Can be one of the keys of the ValidMountTypes map.
	MountType  string `json:"mount_type"`
	Mountpoint string `json:"mountpoint"`
	ReadOnly   bool   `json:"read_only"`
	Required   bool   `json:"required"`
}

// ValidMountTypes defines the values for the "run.mountpoints[].mount_type" fields which are
// understood by the component specification parser
var ValidMountTypes = map[string]dockerMount.Type{
	"bind":   dockerMount.TypeBind,
	"volume": dockerMount.TypeVolume,
	"tmpfs":  dockerMount.TypeTmpfs,
}

// ErrInvalidMountType signifies that there was an error parsing a mount in a component run
// specification. It indicates that the mount type specified for the mount is not a known value.
var ErrInvalidMountType = errors.New("Invalid mount type in component specification: must be one of \"bind\", \"volume\", \"tmpfs\"")

// ReadSingleSpecification reads a single ComponentSpecification JSON document and returns the
// corresponding ComponentSpecification struct. It returns an error if there was an issue parsing
// the specification into the struct.
func ReadSingleSpecification(reader io.Reader) (ComponentSpecification, error) {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	var specification ComponentSpecification
	err := dec.Decode(&specification)

	// Check that mountpoints have valid mount_type fields
	for _, mountpoint := range specification.Run.Mountpoints {
		if _, ok := ValidMountTypes[mountpoint.MountType]; !ok {
			return specification, ErrInvalidMountType
		}
	}

	return specification, err
}
