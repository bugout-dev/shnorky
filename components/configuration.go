package components

import (
	"encoding/json"
	"errors"
	"io"
	"path/filepath"

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

// MaterializeMountConfiguration validates the members of its input mount configuration, applies
// the required substitutions, and returns the resulting values in a new MountConfiguration struct.
func MaterializeMountConfiguration(rawConfig MountConfiguration) (MountConfiguration, error) {
	materializedSource := MaterializeEnv(rawConfig.Source)
	absoluteSource, err := filepath.Abs(materializedSource)
	if err != nil {
		return MountConfiguration{}, err
	}

	materializedConfig := MountConfiguration{
		Source: absoluteSource,
		Target: rawConfig.Target,
		Method: rawConfig.Method,
	}
	if _, ok := ValidMountMethods[materializedConfig.Method]; !ok {
		return materializedConfig, ErrInvalidMountMethod
	}
	return materializedConfig, nil
}

// ReadMountConfiguration reads a single MountConfiguration JSON document from the given reader,
// validates it, and returns it as a MountConfiguration struct. Returns error (in the error
// position) if the MountConfiguration document is invalid or if there is an error reading it from
// the reader. If an error is returned, the offending mount configuration object is returned in a
// singleton array.
func ReadMountConfiguration(reader io.Reader) ([]MountConfiguration, error) {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	var rawMountConfigurations []MountConfiguration
	err := dec.Decode(&rawMountConfigurations)
	if err != nil {
		return []MountConfiguration{}, err
	}

	mountConfigurations := make([]MountConfiguration, len(rawMountConfigurations))
	for i, rawConfig := range rawMountConfigurations {
		materializedConfig, err := MaterializeMountConfiguration(rawConfig)
		mountConfigurations[i] = materializedConfig
		if err != nil {
			return []MountConfiguration{materializedConfig}, err
		}
	}

	return mountConfigurations, nil
}
