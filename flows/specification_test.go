package flows

import (
	"testing"
)

func TestCalculateStages(t *testing.T) {
	type CalculateStagesTest struct {
		specification  FlowSpecification
		expectedStages [][]string
		expectedError  error
	}

	testCases := []CalculateStagesTest{
		{
			specification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
				},
				Dependencies: map[string][]string{
					"b": {"a"},
				},
			},
			expectedStages: [][]string{
				{"a"},
				{"b"},
			},
			expectedError: nil,
		},
		{
			specification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
				},
				Dependencies: map[string][]string{
					"a": {"b"},
					"b": {"a"},
				},
			},
			expectedStages: [][]string{},
			expectedError:  ErrCyclicDependency,
		},
		{
			specification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
					"c": "component-c",
				},
				Dependencies: map[string][]string{
					"b": {"a"},
				},
			},
			expectedStages: [][]string{
				{"a", "c"},
				{"b"},
			},
			expectedError: nil,
		},
		{
			specification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
					"c": "component-c",
				},
				Dependencies: map[string][]string{
					"b": {"a"},
					"c": {"a"},
				},
			},
			expectedStages: [][]string{
				{"a"},
				{"b", "c"},
			},
			expectedError: nil,
		},
		{
			specification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
					"c": "component-c",
				},
				Dependencies: map[string][]string{
					"b": {"a"},
					"c": {"b"},
				},
			},
			expectedStages: [][]string{
				{"a"},
				{"b"},
				{"c"},
			},
			expectedError: nil,
		},
		{
			specification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
					"c": "component-c",
					"d": "component-d",
					"e": "component-e",
					"f": "component-f",
					"g": "component-g",
				},
				Dependencies: map[string][]string{
					"f": {"a", "b", "c"},
					"g": {"a", "b", "c", "d", "e"},
				},
			},
			expectedStages: [][]string{
				{"a", "b", "c", "d", "e"},
				{"f", "g"},
			},
			expectedError: nil,
		},
		{
			specification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
					"c": "component-c",
					"d": "component-d",
					"e": "component-e",
					"f": "component-f",
					"g": "component-g",
					"h": "component-h",
					"i": "component-i",
				},
				Dependencies: map[string][]string{
					"f": {"a", "b", "c"},
					"g": {"a", "b", "c", "d", "e"},
					"h": {"f", "g"},
				},
			},
			expectedStages: [][]string{
				{"a", "b", "c", "d", "e", "i"},
				{"f", "g"},
				{"h"},
			},
			expectedError: nil,
		},
		{
			specification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
					"c": "component-c",
					"d": "component-d",
				},
				Dependencies: map[string][]string{
					"b": {"a"},
					"c": {"a"},
					"d": {"b", "c"},
				},
			},
			expectedStages: [][]string{
				{"a"},
				{"b", "c"},
				{"d"},
			},
			expectedError: nil,
		},
		{
			specification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
					"c": "component-c",
					"d": "component-d",
				},
				Dependencies: map[string][]string{
					"b": {"a"},
					"c": {"b"},
					"d": {"c"},
					"a": {"d"},
				},
			},
			expectedStages: [][]string{},
			expectedError:  ErrCyclicDependency,
		},
	}

	for i, testCase := range testCases {
		stages, err := CalculateStages(testCase.specification)
		if err != testCase.expectedError {
			t.Errorf("[Test %d] Did not get expected error: expected=%v, actual=%v", i, testCase.expectedError, err)
		}
		if len(stages) != len(testCase.expectedStages) {
			t.Fatalf("[Test %d] Calculated stages did not have expected length: expected=%d, actual=%d", i, len(testCase.expectedStages), len(stages))
		}
		for j, stage := range stages {
			if len(stage) != len(testCase.expectedStages[j]) {
				t.Fatalf("[Test %d] [Stage %d] Stage did not have expected length: expected=%d, actual=%d", i, j, len(testCase.expectedStages[j]), len(stage))
			}
			expectedStepMap := map[string]bool{}
			for _, expectedStep := range testCase.expectedStages[j] {
				expectedStepMap[expectedStep] = true
			}

			for _, step := range stage {
				_, ok := expectedStepMap[step]
				if !ok {
					t.Fatalf("[Test %d] [Stage %d] Did not find expected step at this stage: %s", i, j, step)
				}
			}
		}
	}
}

func TestMaterializeSpecification(t *testing.T) {
	type MaterializeSpecificationTest struct {
		rawSpecification      FlowSpecification
		expectedSpecification FlowSpecification
		returnsError          bool
	}

	testCases := []MaterializeSpecificationTest{
		{
			rawSpecification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
				},
				Dependencies: map[string][]string{
					"b": {"a"},
				},
			},
			expectedSpecification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
				},
				Dependencies: map[string][]string{
					"b": {"a"},
				},
				Stages: [][]string{{"a"}, {"b"}},
			},
			returnsError: false,
		},
		{
			rawSpecification: FlowSpecification{
				Steps: map[string]string{
					"a": "",
					"b": "component-b",
				},
				Dependencies: map[string][]string{
					"b": {"a"},
				},
			},
			expectedSpecification: FlowSpecification{
				Steps: map[string]string{
					"a": "",
					"b": "component-b",
				},
				Dependencies: map[string][]string{
					"b": {"a"},
				},
			},
			returnsError: true,
		},
		{
			rawSpecification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
				},
				Dependencies: map[string][]string{
					"c": {"a"},
				},
			},
			expectedSpecification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
				},
				Dependencies: map[string][]string{
					"c": {"a"},
				},
			},
			returnsError: true,
		},
		{
			rawSpecification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
				},
				Dependencies: map[string][]string{
					"b": {"a", "c"},
				},
			},
			expectedSpecification: FlowSpecification{
				Steps: map[string]string{
					"a": "component-a",
					"b": "component-b",
				},
				Dependencies: map[string][]string{
					"b": {"a", "c"},
				},
			},
			returnsError: true,
		},
	}

	for i, testCase := range testCases {
		specification, err := MaterializeFlowSpecification(testCase.rawSpecification)
		if err != nil && !testCase.returnsError {
			t.Errorf("[Test %d] Received error when none was expected: %s", i, err.Error())
			break
		} else if err == nil && testCase.returnsError {
			t.Errorf("[Test %d] No error was thrown but one was expected", i)
			break
		}
		if len(specification.Steps) != len(testCase.expectedSpecification.Steps) {
			t.Errorf("[Test %d] Materialized specification steps did not have expected length: expected=%d, actual=%d", i, len(testCase.expectedSpecification.Steps), len(specification.Steps))
			break
		}
		for step, component := range specification.Steps {
			expectedcomponent, ok := testCase.expectedSpecification.Steps[step]
			if !ok {
				t.Errorf("[Test %d] Unexpected key in materialized steps: %s", i, step)
			} else if component != expectedcomponent {
				t.Errorf("[Test %d] Mismatch in components for step (%s): expected=%s, actual=%s", i, step, expectedcomponent, component)
			}
		}
		if len(specification.Dependencies) != len(testCase.expectedSpecification.Dependencies) {
			t.Errorf("[Test %d] Materialized specification dependencies did not have expected length: expected=%d, actual=%d", i, len(testCase.expectedSpecification.Dependencies), len(specification.Dependencies))
			break
		}
		for step, deps := range specification.Dependencies {
			expectedDeps, ok := testCase.expectedSpecification.Dependencies[step]
			if !ok {
				t.Errorf("[Test %d] Did not expect dependencies for step: %s", i, step)
			} else {
				if len(deps) != len(expectedDeps) {
					t.Errorf("[Test %d] Dependencies for step (%s) did not have expected length: expected=%d, actual=%d", i, step, len(expectedDeps), len(deps))
				}
				for j, dep := range deps {
					if dep != expectedDeps[j] {
						t.Errorf("[Test %d] Mismatch in dependencies for step (%s) at position %d: expected=%s, actual=%s", i, step, j, dep, expectedDeps[j])
					}
				}
			}
		}
	}
}
