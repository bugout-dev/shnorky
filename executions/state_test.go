package executions

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/simiotics/simplex/components"
	"github.com/simiotics/simplex/state"
)

// TestInsertExecution tests that execution insertion works as expected
func TestInsertExecution(t *testing.T) {
	type InsertExecutionTest struct {
		metadata         ExecutionMetadata
		shouldThrowError bool
		inSelection      bool
	}
	stateDir, err := ioutil.TempDir("", "simplex-insert-execution-tests-")
	if err != nil {
		t.Fatalf("Could not create temporary directory: %s", err.Error())
	}
	os.RemoveAll(stateDir)

	err = state.Init(stateDir)
	if err != nil {
		t.Fatalf("Could not initialize state directory: %s", stateDir)
	}
	defer os.RemoveAll(stateDir)

	stateDBPath := path.Join(stateDir, state.DBFileName)
	db, err := sql.Open("sqlite3", stateDBPath)
	if err != nil {
		t.Fatal("Error opening state database file")
	}
	defer db.Close()

	build := components.BuildMetadata{
		ID:          "simplex/good:latest",
		ComponentID: "some-component",
		CreatedAt:   time.Now(),
	}

	components.InsertBuild(db, build)

	tests := []InsertExecutionTest{
		{
			metadata: ExecutionMetadata{
				ID:          "good-execution",
				BuildID:     "simplex/good:latest",
				ComponentID: build.ComponentID,
				CreatedAt:   time.Now(),
			},
			shouldThrowError: false,
			inSelection:      true,
		},
	}

	for i, test := range tests {
		err = InsertExecution(db, test.metadata)
		if test.shouldThrowError && err == nil {
			t.Errorf("[Test %d] Expected error but did not receive one", i)
		} else if !test.shouldThrowError && err != nil {
			t.Errorf("[Test %d] Expected no error, but received: %s", i, err.Error())
		}
	}

	buildSelection := "SELECT id, build_id, component_id, created_at, IFNULL(flow_id, '') as non_null_flow_id FROM executions;"
	rows, err := db.Query(buildSelection)
	defer rows.Close()
	if err != nil {
		t.Fatalf("Error selecting builds from state database: %s", err.Error())
	}

	for i, test := range tests {
		if test.inSelection {
			if !rows.Next() {
				t.Fatalf("[Test %d] Expected result in result set, but found none", i)
			}

			var id, buildID, componentID, flowID string
			var createdAt int64
			err = rows.Scan(&id, &buildID, &componentID, &createdAt, &flowID)
			if err != nil {
				t.Errorf("[Test %d] Error scanning row: %s", i, err.Error())
			}

			if id != test.metadata.ID {
				t.Errorf("[Test %d] Unexpected execution ID: expected=%s, actual=%s", i, test.metadata.ID, id)
			}
			if buildID != test.metadata.BuildID {
				t.Errorf("[Test %d] Unexpected execution BuildID: expected=%s, actual=%s", i, test.metadata.BuildID, buildID)
			}
			if componentID != test.metadata.ComponentID {
				t.Errorf("[Test %d] Unexpected execution ComponentID: expected=%s, actual=%s", i, test.metadata.ComponentID, componentID)
			}
			if createdAt != test.metadata.CreatedAt.Unix() {
				t.Errorf("[Test %d] Unexpected execution CreatedAt: expected=%d, actual=%d", i, test.metadata.CreatedAt.Unix(), createdAt)
			}
			if flowID != test.metadata.FlowID {
				t.Errorf("[Test %d] Unexpected execution FlowID: expected=%s, actual=%s", i, test.metadata.FlowID, flowID)
			}
		}
	}

	if rows.Next() {
		t.Fatal("More rows in builds table than expected")
	}
}
