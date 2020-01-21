package flows

import (
	"encoding/json"
	"io"

	"github.com/simiotics/simplex/components"
)

// ReadMountConfiguration reads mount configurations for each step of a simplex flow. The flow
// mount configuration is expected to be a JSON object, the keys of which are steps in the flow, and
// the values of which are mount configuration arrays for the corresponding components.
func ReadMountConfiguration(reader io.Reader) (map[string][]components.MountConfiguration, error) {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	var mountConfigurations map[string][]components.MountConfiguration
	err := dec.Decode(&mountConfigurations)
	if err != nil {
		return map[string][]components.MountConfiguration{}, err
	}

	return mountConfigurations, nil
}
