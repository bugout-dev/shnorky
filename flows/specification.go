package flows

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/simiotics/shnorky/components"
	"io"
)

// FlowSpecification - struct specifying a shnorky data processing flow
type FlowSpecification struct {
	// Steps indexes each step in the flow and maps step names to component IDs
	Steps map[string]string `json:"steps"`
	// Dependencies has step names as its keys and the corresponding value are the names of steps
	// that the key step depends on. Steps which have no dependencies need not be included in this
	// map
	Dependencies map[string][]string `json:"dependencies"`
	// Stages denotes the sequence in which steps will execute. Steps appearing in the same stage
	// can be run in parallel.
	Stages [][]string `json:"stages,omitempty"`
	// Mounts maps each step (by name) to mount configurations for its corresponding component
	Mounts map[string][]components.MountConfiguration `json:"mounts"`
	// Env maps each step (by name) to environment variable mappings (key-value mappings of variable
	// name to variable value) for that step. The environment variable values get materialized
	// following the same rules as values in a component runtime specification.
	Env map[string]map[string]string `json:"env,omitempty"`
}

// MaterializeFlowSpecification takes a raw FlowSpecification struct and returns a materialized one
// in which the members of the raw specification have been validated and special values have been
// rendered.
func MaterializeFlowSpecification(rawSpecification FlowSpecification) (FlowSpecification, error) {
	for step, component := range rawSpecification.Steps {
		if component == "" {
			return rawSpecification, fmt.Errorf("Invalid component for step %s", step)
		}
	}

	for step, deps := range rawSpecification.Dependencies {
		_, ok := rawSpecification.Steps[step]
		if !ok {
			return rawSpecification, fmt.Errorf("Unknown step in dependencies: %s", step)
		}

		for _, dependency := range deps {
			_, ok = rawSpecification.Steps[dependency]
			if !ok {
				return rawSpecification, fmt.Errorf("Unknown dependency (%s) for step (%s)", dependency, step)
			}
		}
	}

	materializedSpecification := FlowSpecification{
		Steps:        rawSpecification.Steps,
		Dependencies: rawSpecification.Dependencies,
	}

	// Stages will always get recalculated, even if it is already populated in the rawSpecification
	stages, err := CalculateStages(rawSpecification)
	materializedSpecification.Stages = stages
	if err != nil {
		return materializedSpecification, err
	}

	materializedMounts := map[string][]components.MountConfiguration{}
	for step, rawConfigs := range rawSpecification.Mounts {
		materializedConfigs := make([]components.MountConfiguration, len(rawConfigs))
		for i, rawConfig := range rawConfigs {
			materializedConfig, err := components.MaterializeMountConfiguration(rawConfig)
			if err != nil {
				materializedSpecification.Mounts = map[string][]components.MountConfiguration{
					step: {materializedConfig},
				}
				return materializedSpecification, err
			}
			materializedConfigs[i] = materializedConfig
		}
		materializedMounts[step] = materializedConfigs
	}
	materializedSpecification.Mounts = materializedMounts

	materializedEnv := map[string]map[string]string{}
	for step, envMap := range rawSpecification.Env {
		materializedEnvMap := map[string]string{}
		for key, value := range envMap {
			materializedEnvMap[key] = components.MaterializeEnv(value)
		}
		materializedEnv[step] = materializedEnvMap
	}
	materializedSpecification.Env = materializedEnv

	return materializedSpecification, nil
}

// ReadSingleSpecification reads a single ComponentSpecification JSON document and returns the
// corresponding ComponentSpecification struct. It returns an error if there was an issue parsing
// the specification into the struct.
func ReadSingleSpecification(reader io.Reader) (FlowSpecification, error) {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	var rawSpecification FlowSpecification
	err := dec.Decode(&rawSpecification)
	if err != nil {
		return rawSpecification, fmt.Errorf("Error decoding flow specification: %s", err.Error())
	}

	// Performs full verification (including dependency resolution)
	specification, err := MaterializeFlowSpecification(rawSpecification)
	if err != nil {
		return specification, fmt.Errorf("Error validating flow specification: %s", err.Error())
	}

	return specification, nil
}

// ErrCyclicDependency is returned when flow dependency resolution fails because there was a cycle
// in the dependency graph.
var ErrCyclicDependency = errors.New("Cyclic dependency detected in given flow")

// CalculateStages calculates stages for the execution of the flow with the given specification.
// Each stage is an array of flow steps which can be executed concurrently (although they do not
// have to be)
func CalculateStages(specification FlowSpecification) ([][]string, error) {
	// Base case of the recursion
	if len(specification.Steps) == 0 {
		return [][]string{}, nil
	}

	initialSteps := map[string]bool{}
	for step := range specification.Steps {
		dependencies, ok := specification.Dependencies[step]
		if !ok || (len(dependencies) == 0) {
			initialSteps[step] = true
		}
	}

	if len(initialSteps) == 0 {
		return [][]string{}, ErrCyclicDependency
	}

	currentStage := make([]string, len(initialSteps))
	i := 0
	for step := range initialSteps {
		currentStage[i] = step
		i++
	}

	nextSteps := map[string]string{}
	for step, component := range specification.Steps {
		_, ok := initialSteps[step]
		if !ok {
			nextSteps[step] = component
		}
	}

	nextDependencies := map[string][]string{}
	for step := range nextSteps {
		survivingDeps := make([]string, len(specification.Dependencies[step]))
		i = 0
		for _, dep := range specification.Dependencies[step] {
			_, ok := initialSteps[dep]
			if !ok {
				survivingDeps[i] = dep
				i++
			}
		}
		survivingDeps = survivingDeps[:i]
		nextDependencies[step] = survivingDeps
	}

	nextSpecification := FlowSpecification{Steps: nextSteps, Dependencies: nextDependencies}

	downstreamStages, err := CalculateStages(nextSpecification)
	if err != nil {
		return [][]string{}, err
	}

	stages := [][]string{currentStage}
	stages = append(stages, downstreamStages...)
	return stages, nil
}
