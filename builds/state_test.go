package builds

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/simiotics/simplex/state"
)

// TestInsertBuild tests that build insertion works as expected
func TestInsertBuild(t *testing.T) {
	type InsertBuildTest struct {
		metadata         BuildMetadata
		shouldThrowError bool
		inSelection      bool
	}

	stateDir, err := ioutil.TempDir("", "simplex-insert-build-tests-")
	if err != nil {
		t.Fatalf("Could not create temporary directory: %s", err.Error())
	}
	os.RemoveAll(stateDir)

	err = state.Init(stateDir)
	if err != nil {
		t.Fatalf("Could not initialize state directory: %s", stateDir)
	}
	defer os.RemoveAll(stateDir)

	tests := []InsertBuildTest{
		{
			metadata: BuildMetadata{
				ID:          "lol",
				ComponentID: "component-lol",
				CreatedAt:   time.Now(),
			},
			shouldThrowError: false,
			inSelection:      true,
		},
		{
			metadata: BuildMetadata{
				ID:          "rofl",
				ComponentID: "component-rofl",
				CreatedAt:   time.Now(),
			},
			shouldThrowError: false,
			inSelection:      true,
		},
		{
			metadata: BuildMetadata{
				ID:          "lol",
				ComponentID: "some-other-component",
				CreatedAt:   time.Now(),
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
		err = InsertBuild(db, test.metadata)
		if test.shouldThrowError && err == nil {
			t.Errorf("[Test %d] Expected error but did not receive one", i)
		} else if !test.shouldThrowError && err != nil {
			t.Errorf("[Test %d] Expected no error, but received: %s", i, err.Error())
		}
	}

	buildSelection := "SELECT * FROM builds;"
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

			var id, componentID string
			var createdAt int64
			err = rows.Scan(&id, &componentID, &createdAt)
			if err != nil {
				t.Errorf("[Test %d] Error scanning row: %s", i, err.Error())
			}

			if id != test.metadata.ID {
				t.Errorf("[Test %d] Unexpected build ID: expected=%s, actual=%s", i, test.metadata.ID, id)
			}
			if componentID != test.metadata.ComponentID {
				t.Errorf("[Test %d] Unexpected build ComponentID: expected=%s, actual=%s", i, test.metadata.ComponentID, componentID)
			}
			if createdAt != test.metadata.CreatedAt.Unix() {
				t.Errorf("[Test %d] Unexpected build CreatedAt: expected=%d, actual=%d", i, test.metadata.CreatedAt.Unix(), createdAt)
			}
		}
	}

	if rows.Next() {
		t.Fatal("More rows in builds table than expected")
	}
}
