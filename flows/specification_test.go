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

func TestValidateSpecification(t *testing.T) {
	type ValidateSpecificationTest struct {
		specification FlowSpecification
		returnsError  bool
	}

	testCases := []ValidateSpecificationTest{
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
			returnsError: false,
		},
		{
			specification: FlowSpecification{
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
			specification: FlowSpecification{
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
			specification: FlowSpecification{
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
		err := ValidateSpecification(testCase.specification)
		if err != nil && !testCase.returnsError {
			t.Errorf("[Test %d] Received error when none was expected: %s", i, err.Error())
		} else if err == nil && testCase.returnsError {
			t.Errorf("[Test %d] No error was thrown but one was expected", i)
		}
	}
}
