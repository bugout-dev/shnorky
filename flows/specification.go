package components

import (
	"encoding/json"
	"io"
)

// FlowSpecification - struct specifying a simplex data processing flow
type FlowSpecification struct {
	// Steps indexes each step in the flow and maps step names to component IDs
	Steps map[string]string
	// Dependencies has step names as its keys and the corresponding value are the names of steps
	// that the key step depends on
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
