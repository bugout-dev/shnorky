package components

import (
	"strings"
	"testing"
)

func TestReadSingleSpecification(t *testing.T) {
	type ReadSingleSpecificationTestCase struct {
		specificationRaw string
		returnsError     bool
	}

	testCases := []ReadSingleSpecificationTestCase{
		// Ideal case
		{
			specificationRaw: `
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
}`,
			returnsError: false,
		},
		// No extra keys allowed in any object
		{
			specificationRaw: `
{
	"build": {
		"extra_key": "extra_value",
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
}`,
			returnsError: true,
		},
		// run.env must be parseable into a map[string]string
		{
			specificationRaw: `
{
	"build": {
		"Dockerfile": "Dockerfile",
		"context": "component-dir"
	},
	"run": {
		"env": ["ENV_KEY_1=ENV_VALUE_1", "ENV_KEY_2=ENV_VALUE_2"],
		"cmd": ["echo", "hello", "world"],
		"mountpoints": [
			{
				"mountpoint": "/opt/mounthere",
				"read_only": false,
				"required": true
			}
		]
	}
}`,
			returnsError: true,
		},
		// run.cmd must be parseable into a []string
		{
			specificationRaw: `
{
	"build": {
		"Dockerfile": "Dockerfile",
		"context": "component-dir"
	},
	"run": {
		"env": {"ENV_KEY_1": "ENV_VALUE_1", "ENV_KEY_2": "ENV_VALUE_2"},
		"cmd": "bash",
		"mountpoints": [
			{
				"mountpoint": "/opt/mounthere",
				"read_only": false,
				"required": true
			}
		]
	}
}`,
			returnsError: true,
		},
		// Mountpoints can be an empty array
		{
			specificationRaw: `
{
	"build": {
		"Dockerfile": "Dockerfile",
		"context": "component-dir"
	},
	"run": {
		"env": {"ENV_KEY_1": "ENV_VALUE_1", "ENV_KEY_2": "ENV_VALUE_2"},
		"cmd": ["echo", "hello", "world"],
		"mountpoints": []
	}
}`,
			returnsError: false,
		},
	}

	for i, testCase := range testCases {
		reader := strings.NewReader(testCase.specificationRaw)
		_, err := ReadSingleSpecification(reader)
		if err != nil && !testCase.returnsError {
			t.Errorf("[Test %d] Did not expect error: %s", i, err.Error())
		} else if err == nil && testCase.returnsError {
			t.Errorf("[Test %d] Expected error but received none", i)
		}
	}
}
