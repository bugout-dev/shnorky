package flows

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/simiotics/simplex/state"
)

// TestInsertFlow tests that flow insertion works as expected
func TestInsertFlow(t *testing.T) {
	type InsertFlowTest struct {
		metadata         FlowMetadata
		shouldThrowError bool
		inSelection      bool
	}

	stateDir, err := ioutil.TempDir("", "simplex-insert-flow-tests-")
	if err != nil {
		t.Fatalf("Could not create temporary directory: %s", err.Error())
	}
	os.RemoveAll(stateDir)

	err = state.Init(stateDir)
	if err != nil {
		t.Fatalf("Could not initialize state directory: %s", stateDir)
	}
	defer os.RemoveAll(stateDir)

	tests := []InsertFlowTest{
		{
			metadata: FlowMetadata{
				ID:                "lol",
				SpecificationPath: "/tmp/flows/lol/flow.json",
				CreatedAt:         time.Now(),
			},
			shouldThrowError: false,
			inSelection:      true,
		},
		{
			metadata: FlowMetadata{
				ID:                "rofl",
				SpecificationPath: "/tmp/flows/rofl/flow.json",
				CreatedAt:         time.Now(),
			},
			shouldThrowError: false,
			inSelection:      true,
		},
		{
			metadata: FlowMetadata{
				ID:                "lol",
				SpecificationPath: "/tmp/flows/lol/flow.json",
				CreatedAt:         time.Now(),
			},
			shouldThrowError: true,
			inSelection:      false,
		},
	}

	stateDBPath := path.Join(stateDir, state.DBFileName)
	db, err := sql.Open("sqlite3", stateDBPath)
	if err != nil {
		t.Fatal("Error opening state database file")
	}
	defer db.Close()

	for i, test := range tests {
		err = InsertFlow(db, test.metadata)
		if test.shouldThrowError && err == nil {
			t.Errorf("[Test %d] Expected error but did not receive one", i)
		} else if !test.shouldThrowError && err != nil {
			t.Errorf("[Test %d] Expected no error, but received: %s", i, err.Error())
		}
	}

	flowSelection := "SELECT * FROM flows;"
	rows, err := db.Query(flowSelection)
	defer rows.Close()
	if err != nil {
		t.Fatalf("Error selecting flows from state database: %s", err.Error())
	}

	for i, test := range tests {
		if test.inSelection {
			if !rows.Next() {
				t.Fatalf("[Test %d] Expected result in result set, but found none", i)
			}

			var id, specificationPath string
			var createdAt int64
			err = rows.Scan(&id, &specificationPath, &createdAt)
			if err != nil {
				t.Errorf("[Test %d] Error scanning row: %s", i, err.Error())
			}

			if id != test.metadata.ID {
				t.Errorf("[Test %d] Unexpected flow ID: expected=%s, actual=%s", i, test.metadata.ID, id)
			}
			if specificationPath != test.metadata.SpecificationPath {
				t.Errorf("[Test %d] Unexpected flow SpecificationPath: expected=%s, actual=%s", i, test.metadata.SpecificationPath, specificationPath)
			}
			if createdAt != test.metadata.CreatedAt.Unix() {
				t.Errorf("[Test %d] Unexpected flow CreatedAt: expected=%d, actual=%d", i, test.metadata.CreatedAt.Unix(), createdAt)
			}
		}
	}

	if rows.Next() {
		t.Fatal("More rows in flows table than expected")
	}
}

// TestSelectFlowByID first runs InsertFlow a number of times to load a temporary state database
// with some flows. Then it tests various SelectFlowByID scenarios.
func TestSelectFlowByID(t *testing.T) {
	stateDir, err := ioutil.TempDir("", "simplex-select-flow-by-id-tests-")
	if err != nil {
		t.Fatalf("Could not create temporary directory: %s", err.Error())
	}
	os.RemoveAll(stateDir)

	err = state.Init(stateDir)
	if err != nil {
		t.Fatalf("Error creating state directory: %s", err.Error())
	}
	defer os.RemoveAll(stateDir)

	stateDBPath := path.Join(stateDir, state.DBFileName)
	db, err := sql.Open("sqlite3", stateDBPath)
	if err != nil {
		t.Fatal("Error opening state database file")
	}
	defer db.Close()

	var i int
	flows := make([]FlowMetadata, 10)
	for i = 0; i < 10; i++ {
		flow, err := GenerateFlowMetadata(fmt.Sprintf("flow-%d", i), fmt.Sprintf("flow-%d.json", i))
		if err != nil {
			t.Fatalf("[Flow %d] Error creating flow metadata: %s", i, err.Error())
		}
		flows[i] = flow
		err = InsertFlow(db, flow)
		if err != nil {
			t.Fatalf("[Flow %d] Error inserting flow into state database: %s", i, err.Error())
		}
	}

	for i = 0; i < 10; i++ {
		stateFlow, err := SelectFlowByID(db, flows[i].ID)
		if err != nil {
			t.Errorf("[Test %d] Received error when trying to get inserted flow: %s", i, err.Error())
		}
		if stateFlow.ID != flows[i].ID {
			t.Errorf("[Test %d] Unexpected ID retrieved from state database: expected=%s, actual=%s", i, flows[i].ID, stateFlow.ID)
		}
		if stateFlow.SpecificationPath != flows[i].SpecificationPath {
			t.Errorf("[Test %d] Unexpected SpecificationPath retrieved from state database: expected=%s, actual=%s", i, flows[i].SpecificationPath, stateFlow.SpecificationPath)
		}
		expectedCreatedAt := time.Unix(flows[i].CreatedAt.Unix(), 0)
		if stateFlow.CreatedAt != expectedCreatedAt {
			t.Errorf("[Test %d] Unexpected CreatedAt retrieved from state database: expected=%s, actual=%s", i, expectedCreatedAt, stateFlow.CreatedAt)
		}
	}

	stateFlow, err := SelectFlowByID(db, "nonexistent-id")
	if err != ErrFlowNotFound {
		t.Error("[Test 11] Was expecting error ErrFlowNotFound for GetFlowByID on unregistered ID, but did not get it")
	}
	if stateFlow.ID != "" {
		t.Errorf("[Test 11] GetFlowByID on unregistered ID returned non-empty ID: %s", stateFlow.ID)
	}
	if stateFlow.SpecificationPath != "" {
		t.Errorf("[Test 11] GetFlowByID on unregistered ID returned non-empty SpecificationPath: %s", stateFlow.SpecificationPath)
	}
	if !stateFlow.CreatedAt.IsZero() {
		t.Errorf("[Test 11] GetFlowByID on unregistered ID returned non-zero CreatedAt: %v", stateFlow.CreatedAt)
	}
}
