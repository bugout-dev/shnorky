package components

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
)

// ErrInvalidMountType signifies that there was an error parsing a component mount specification.
// Specifically, that the MountType member did not have a valid value.
var ErrInvalidMountType = errors.New("Invalid mount type in component mount specification: must be one of \"file\", \"dir\"")

// ComponentSpecification - struct specifying how a component of a shnorky data processing flow
// should be built and executed
type ComponentSpecification struct {
	Build BuildSpecification `json:"build"`
	Run   RunSpecification   `json:"run"`
}

// BuildSpecification - struct specifying how a component of a shnorky data processing flow should
// be built; all paths are assumed to be paths relative to the component path (i.e. the directory
// containing the implementation of the component)
type BuildSpecification struct {
	// Path to context directory (used to build docker image)
	Context string `json:"context"`

	// Path to Dockerfile to be used to build the component - should be relative to the context
	// path
	Dockerfile string `json:"Dockerfile"`
}

// RunSpecification - struct specifying how a component of a shnorky data processing flow should be
// executed
type RunSpecification struct {
	// Mapping of environment variable names to values to be set in component container at runtime
	// Special keys:
	//
	// Special values:
	// "env:<VARIABLE_NAME>" - specifies that the value of the environment variable denoted by
	// VARIABLE_NAME in the shnorky process should be interpolated into the specification; if the
	// environment variable is not set in the shnorky process, it will use the empty string "" as
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
	// "env:<VARIABLE_NAME>" - container runs as user specified by environment variable; use
	// "env:USER" to use the user running the current shnorky process, for example
	// "user:<username>" - container runs as the user with the given username
	User string `json:"user"`
}

// MountType is an enum representing the valid mount types for mount specifications
type MountType int

const (
	// MountTypeFile - mount point is a file
	MountTypeFile MountType = iota + 1
	// MountTypeDir - mount point is a directory
	MountTypeDir
)

// MountSpecification - specifies a mount point within a shnorky component, how it should be mounted
// on the container side, and whether or not it is required to be mounted at runtime
// TODO(nkashy1): It does not make sense to specify this kind of mount type in the
// MountSpecification - the mount type (e.g. whether it is a bind mount or a docker volume mount) is
// the responsibility of the caller. What does make sense is for MountType to specify the type of
// filesystem object that the mountpoint expects (e.g. file vs. directory)
type MountSpecification struct {
	// Can be one of the keys of the ValidMountTypes map.
	MountType  string `json:"mount_type"`
	Mountpoint string `json:"mountpoint"`
	ReadOnly   bool   `json:"read_only"`
	Required   bool   `json:"required"`
}

// ValidMountTypes is a map whose keys are the valid values for the Type member in a
// MountSpecification. This is here to make it easier to create a MountSpecification JSON document.
var ValidMountTypes = map[string]MountType{
	"file": MountTypeFile,
	"dir":  MountTypeDir,
}

// ReadSingleSpecification reads a single ComponentSpecification JSON document and returns the
// corresponding ComponentSpecification struct. It returns an error if there was an issue parsing
// the specification into the struct.
func ReadSingleSpecification(reader io.Reader) (ComponentSpecification, error) {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	var specification ComponentSpecification
	err := dec.Decode(&specification)
	if err != nil {
		return ComponentSpecification{}, err
	}

	for _, mountSpec := range specification.Run.Mountpoints {
		if _, ok := ValidMountTypes[mountSpec.MountType]; !ok {
			return specification, ErrInvalidMountType
		}
	}

	return specification, nil
}

// MaterializeComponentSpecification applies all run-time substitutions to the given
// ComponentSpecification
// For example, it replaces all "env:..." values with values of the corresponding environment
// variables in the invoking process.
func MaterializeComponentSpecification(rawSpecification ComponentSpecification) (ComponentSpecification, error) {
	materializedRunSpecification, err := MaterializeRunSpecification(rawSpecification.Run)
	if err != nil {
		return rawSpecification, fmt.Errorf("Could not materialize run specification: %s", err.Error())
	}

	materializedSpecification := ComponentSpecification{
		Build: rawSpecification.Build,
		Run:   materializedRunSpecification,
	}
	return materializedSpecification, nil
}

// MaterializeRunSpecification applies all run-time substitutions to the given RunSpecification
func MaterializeRunSpecification(rawSpecification RunSpecification) (RunSpecification, error) {
	materializedUser, err := MaterializeUsername(rawSpecification.User)
	if err != nil {
		return rawSpecification, fmt.Errorf("Could not materialize user: %s", err.Error())
	}

	materializedEnv := map[string]string{}
	for key, value := range rawSpecification.Env {
		materializedEnv[key] = MaterializeEnv(value)
	}

	materializedSpecification := RunSpecification{
		Env:         materializedEnv,
		Entrypoint:  rawSpecification.Entrypoint,
		Cmd:         rawSpecification.Cmd,
		Mountpoints: rawSpecification.Mountpoints,
		User:        materializedUser,
	}
	return materializedSpecification, nil
}

// SpecialPrefixEnv denotes that a value in a specification refers to the environment variable whose
// name is its suffix.
var SpecialPrefixEnv = "env:"

// SpecialPrefixUsername denotes that a value in a specification refers to a username, its suffix.
var SpecialPrefixUsername = "user:"

// MaterializeEnv checks if a string is prefixed with "env:". If it is, it returns the value of the
// environment variable whose name is the remainder of the string. If not, it returns the input
// value.
func MaterializeEnv(rawValue string) string {
	if len(rawValue) >= len(SpecialPrefixEnv) && rawValue[:len(SpecialPrefixEnv)] == SpecialPrefixEnv {
		return os.Getenv(rawValue[len(SpecialPrefixEnv):])
	}
	return rawValue
}

// MaterializeUsername returns a "uid:gid" string for the user with the given name if the user
// exists, otherwise it returns an error
func MaterializeUsername(rawValue string) (string, error) {
	if len(rawValue) >= len(SpecialPrefixUsername) && rawValue[:len(SpecialPrefixUsername)] == SpecialPrefixUsername {
		targetUser, err := user.Lookup(rawValue[len(SpecialPrefixUsername):])
		if err != nil {
			return rawValue, err
		}
		return fmt.Sprintf("%s:%s", targetUser.Uid, targetUser.Gid), nil
	}
	return rawValue, nil
}
