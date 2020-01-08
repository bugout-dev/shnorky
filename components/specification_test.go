package components

import (
	"strings"
	"testing"
)

func TestReadSingleSpecification(t *testing.T) {
	specificationRaw := `
{
	"build": {
		"Dockerfile": "Dockerfile",
		"context": "component-dir"
	},
	"run": {
		"env": {"ENV_KEY_1": "ENV_VALUE_1", "ENV_KEY_2": "ENV_VALUE_2"},
		"cmd": ["echo", "hello", "world"],
		"mountpoints": [
			{
				"mountpoint": "/opt/mounthere",
				"read_only": false,
				"required": true
			}
		]
	}
}
`

	reader := strings.NewReader(specificationRaw)
	_, err := ReadSingleSpecification(reader)
	if err != nil {
		t.Fatalf("Unexpected error reading specification: %s", err.Error())
	}
}
