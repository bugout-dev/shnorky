package components

import (
	"encoding/json"
	"errors"
	"io"

	dockerMount "github.com/docker/docker/api/types/mount"
)

// ErrInvalidMountMethod signifies that there was an error parsing a mount in a component mount
// configuration. It indicates that the value for the Method member was invalid.
var ErrInvalidMountMethod = errors.New("Invalid mount method in component mount configuration: must be one of \"bind\", \"volume\", \"tmpfs\"")

// MountConfiguration - describes the run-time mount configuration for a shnorky component
type MountConfiguration struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Method string `json:"method"`
}

// ValidMountMethods defines the values for the MountConfiguration Method member
var ValidMountMethods = map[string]dockerMount.Type{
	"bind":   dockerMount.TypeBind,
	"volume": dockerMount.TypeVolume,
	"tmpfs":  dockerMount.TypeTmpfs,
}

// ReadMountConfiguration reads a single MountConfiguration JSON document from the given reader,
// validates it, and returns it as a MountConfiguration struct. Returns error (in the error
// position) if the MountConfiguration document is invalid or if there is an error reading it from
// the reader.
func ReadMountConfiguration(reader io.Reader) ([]MountConfiguration, error) {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	var mountConfigurations []MountConfiguration
	err := dec.Decode(&mountConfigurations)
	if err != nil {
		return []MountConfiguration{}, err
	}

	// TODO(nkashy1): Factor this validation out into a separate function so that it can be reused
	// in the flows package equivalent to this function.
	for _, mountConfiguration := range mountConfigurations {
		if _, ok := ValidMountMethods[mountConfiguration.Method]; !ok {
			return mountConfigurations, ErrInvalidMountMethod
		}
	}

	return mountConfigurations, nil
}
