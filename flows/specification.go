package flows

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// FlowSpecification - struct specifying a simplex data processing flow
type FlowSpecification struct {
	// Steps indexes each step in the flow and maps step names to component IDs
	Steps map[string]string
	// Dependencies has step names as its keys and the corresponding value are the names of steps
	// that the key step depends on. Steps which have no dependencies need not be included in this
	// map
	Dependencies map[string][]string
}

// ReadSingleSpecification reads a single ComponentSpecification JSON document and returns the
// corresponding ComponentSpecification struct. It returns an error if there was an issue parsing
// the specification into the struct.
func ReadSingleSpecification(reader io.Reader) (FlowSpecification, error) {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	var specification FlowSpecification
	err := dec.Decode(&specification)

	return specification, err
}

// ValidateSpecification accepts a FlowSpecification and returns nil if the specification is valid
// and returns an appropriate error if that is not the case.
func ValidateSpecification(specification FlowSpecification) error {
	for step, component := range specification.Steps {
		if component == "" {
			return fmt.Errorf("Invalid component for step %s", step)
		}
	}

	for step, deps := range specification.Dependencies {
		_, ok := specification.Steps[step]
		if !ok {
			return fmt.Errorf("Unknown step in dependencies: %s", step)
		}

		for _, dependency := range deps {
			_, ok = specification.Steps[dependency]
			if !ok {
				return fmt.Errorf("Unknown dependency (%s) for step (%s)", dependency, step)
			}
		}
	}

	return nil
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
