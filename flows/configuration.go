package flows

import (
	"encoding/json"
	"io"

	"github.com/simiotics/shnorky/components"
)

// ReadMountConfiguration reads mount configurations for each step of a shnorky flow. The flow
// mount configuration is expected to be a JSON object, the keys of which are steps in the flow, and
// the values of which are mount configuration arrays for the corresponding components.
func ReadMountConfiguration(reader io.Reader) (map[string][]components.MountConfiguration, error) {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	var rawMountConfigurations map[string][]components.MountConfiguration
	err := dec.Decode(&rawMountConfigurations)
	if err != nil {
		return map[string][]components.MountConfiguration{}, err
	}

	var mountConfigurations map[string][]components.MountConfiguration
	for step, rawConfigs := range rawMountConfigurations {
		materializedConfigs := make([]components.MountConfiguration, len(rawConfigs))
		for i, rawConfig := range rawConfigs {
			materializedConfig, err := components.MaterializeMountConfiguration(rawConfig)
			if err != nil {
				return map[string][]components.MountConfiguration{step: {materializedConfig}}, err
			}
			materializedConfigs[i] = materializedConfig
		}
		mountConfigurations[step] = materializedConfigs
	}

	return mountConfigurations, nil
}
