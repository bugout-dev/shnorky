package builds

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

// TestSelectBuildByID first runs InsertBuild a number of times to load a temporary state database
// with some builds. Then it tests various SelectBuildByID scenarios.
func TestSelectBuildByID(t *testing.T) {
	stateDir, err := ioutil.TempDir("", "simplex-select-build-by-id-tests-")
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
	builds := make([]BuildMetadata, 10)
	for i = 0; i < 10; i++ {
		build, err := GenerateBuildMetadata(fmt.Sprintf("component-%d", i))
		if err != nil {
			t.Fatalf("[Build %d] Error creating build metadata: %s", i, err.Error())
		}
		builds[i] = build
		err = InsertBuild(db, build)
		if err != nil {
			t.Fatalf("[Build %d] Error inserting build into state database: %s", i, err.Error())
		}
	}

	for i = 0; i < 10; i++ {
		stateBuild, err := SelectBuildByID(db, builds[i].ID)
		if err != nil {
			t.Errorf("[Test %d] Received error when trying to get inserted build: %s", i, err.Error())
		}
		if stateBuild.ID != builds[i].ID {
			t.Errorf("[Test %d] Unexpected ID retrieved from state database: expected=%s, actual=%s", i, builds[i].ID, stateBuild.ID)
		}
		if stateBuild.ComponentID != builds[i].ComponentID {
			t.Errorf("[Test %d] Unexpected ComponentID retrieved from state database: expected=%s, actual=%s", i, builds[i].ComponentID, stateBuild.ComponentID)
		}
		expectedCreatedAt := time.Unix(builds[i].CreatedAt.Unix(), 0)
		if stateBuild.CreatedAt != expectedCreatedAt {
			t.Errorf("[Test %d] Unexpected CreatedAt retrieved from state database: expected=%s, actual=%s", i, expectedCreatedAt, stateBuild.CreatedAt)
		}
	}

	stateBuild, err := SelectBuildByID(db, "nonexistent-id")
	if err != ErrBuildNotFound {
		t.Error("[Test 11] Was expecting error ErrBuildNotFound for GetBuildByID on unregistered ID, but did not get it")
	}
	if stateBuild.ID != "" {
		t.Errorf("[Test 11] GetBuildByID on unregistered ID returned non-empty ID: %s", stateBuild.ID)
	}
	if stateBuild.ComponentID != "" {
		t.Errorf("[Test 11] GetBuildByID on unregistered ID returned non-empty ComponentID: %s", stateBuild.ComponentID)
	}
	if !stateBuild.CreatedAt.IsZero() {
		t.Errorf("[Test 11] GetBuildByID on unregistered ID returned non-zero CreatedAt: %v", stateBuild.CreatedAt)
	}
}
