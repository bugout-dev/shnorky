package components

import (
	"testing"
)

// TestGenerateComponentMetadata tests that ComponentMetadata validation and defaults behave as
// expected. Does not test CreatedAt.
func TestGenerateComponentMetadata(t *testing.T) {
	type GenerateComponentMetadataTest struct {
		id                string
		componentType     string
		componentPath     string
		specificationPath string
		expectedMetadata  ComponentMetadata
		expectedError     error
	}

	tests := []GenerateComponentMetadataTest{
		{
			id:                "explicit-valid",
			componentType:     Task,
			componentPath:     "/tmp/component",
			specificationPath: "/tmp/specification.json",
			expectedMetadata: ComponentMetadata{
				ID:                "explicit-valid",
				ComponentType:     Task,
				ComponentPath:     "/tmp/component",
				SpecificationPath: "/tmp/specification.json",
			},
			expectedError: nil,
		},
		{
			id:            "implicit-valid",
			componentType: Task,
			componentPath: "/tmp/component",
			expectedMetadata: ComponentMetadata{
				ID:                "implicit-valid",
				ComponentType:     Task,
				ComponentPath:     "/tmp/component",
				SpecificationPath: "/tmp/component/component.json",
			},
			expectedError: nil,
		},
		{
			componentType:    Task,
			componentPath:    "/tmp/component",
			expectedMetadata: ComponentMetadata{},
			expectedError:    ErrEmptyID,
		},
		{
			componentPath:    "/tmp/component",
			expectedMetadata: ComponentMetadata{},
			expectedError:    ErrEmptyID,
		},
		{
			id:               "invalid-component-type",
			componentType:    "erroneous-component-type",
			componentPath:    "/tmp/component",
			expectedMetadata: ComponentMetadata{},
			expectedError:    ErrInvalidComponentType,
		},
		{
			id:               "empty-component-type",
			componentPath:    "/tmp/component",
			expectedMetadata: ComponentMetadata{},
			expectedError:    ErrInvalidComponentType,
		},
		{
			id:               "empty-component-path",
			componentType:    Service,
			expectedMetadata: ComponentMetadata{},
			expectedError:    ErrEmptyComponentPath,
		},
	}

	for i, test := range tests {
		metadata, err := GenerateComponentMetadata(test.id, test.componentType, test.componentPath, test.specificationPath)
		if test.expectedError == nil && err != nil {
			t.Errorf("[Test %d] Unexpected error: %s", i, err.Error())
		} else if test.expectedError != nil {
			if err == nil {
				t.Errorf("[Test %d] GenerateComponentMetadata returned no error but should have returned: %s", i, test.expectedError.Error())
			} else if err != test.expectedError {
				t.Errorf("[Test %d] GenerateComponentMetadata returned an unexpected error: expected=%s, actual=%s", i, test.expectedError.Error(), err.Error())
			}
		}

		if metadata.ID != test.expectedMetadata.ID {
			t.Errorf("[Test %d] ComponentMetadata ID mismatch: expected=%s, actual=%s", i, test.expectedMetadata.ID, metadata.ID)
		}
		if metadata.ComponentType != test.expectedMetadata.ComponentType {
			t.Errorf("[Test %d] ComponentMetadata ComponentType mismatch: expected=%s, actual=%s", i, test.expectedMetadata.ComponentType, metadata.ComponentType)
		}
		if metadata.ComponentPath != test.expectedMetadata.ComponentPath {
			t.Errorf("[Test %d] ComponentMetadata ComponentPath mismatch: expected=%s, actual=%s", i, test.expectedMetadata.ComponentPath, metadata.ComponentPath)
		}
		if metadata.SpecificationPath != test.expectedMetadata.SpecificationPath {
			t.Errorf("[Test %d] ComponentMetadata SpecificationPath mismatch: expected=%s, actual=%s", i, test.expectedMetadata.SpecificationPath, metadata.SpecificationPath)
		}
	}
}
