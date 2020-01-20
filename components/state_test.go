package components

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/simiotics/simplex/state"
)

// TestInsertComponent tests that component insertion works as expected
func TestInsertComponent(t *testing.T) {
	type InsertComponentTest struct {
		metadata         ComponentMetadata
		shouldThrowError bool
		inSelection      bool
	}

	stateDir, err := ioutil.TempDir("", "simplex-insert-component-tests-")
	if err != nil {
		t.Fatalf("Could not create temporary directory: %s", err.Error())
	}
	os.RemoveAll(stateDir)

	err = state.Init(stateDir)
	if err != nil {
		t.Fatalf("Could not initialize state directory: %s", stateDir)
	}
	defer os.RemoveAll(stateDir)

	tests := []InsertComponentTest{
		{
			metadata: ComponentMetadata{
				ID:                "lol",
				ComponentType:     Task,
				ComponentPath:     "/tmp/components/lol",
				SpecificationPath: "/tmp/components/lol/component.json",
				CreatedAt:         time.Now(),
			},
			shouldThrowError: false,
			inSelection:      true,
		},
		{
			metadata: ComponentMetadata{
				ID:                "rofl",
				ComponentType:     Task,
				ComponentPath:     "/tmp/components/rofl",
				SpecificationPath: "/tmp/components/rofl/component.json",
				CreatedAt:         time.Now(),
			},
			shouldThrowError: false,
			inSelection:      true,
		},
		{
			metadata: ComponentMetadata{
				ID:                "lol",
				ComponentType:     Task,
				ComponentPath:     "/tmp/components/lol",
				SpecificationPath: "/tmp/components/lol/component.json",
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
		err = InsertComponent(db, test.metadata)
		if test.shouldThrowError && err == nil {
			t.Errorf("[Test %d] Expected error but did not receive one", i)
		} else if !test.shouldThrowError && err != nil {
			t.Errorf("[Test %d] Expected no error, but received: %s", i, err.Error())
		}
	}

	componentSelection := "SELECT * FROM components;"
	rows, err := db.Query(componentSelection)
	defer rows.Close()
	if err != nil {
		t.Fatalf("Error selecting components from state database: %s", err.Error())
	}

	for i, test := range tests {
		if test.inSelection {
			if !rows.Next() {
				t.Fatalf("[Test %d] Expected result in result set, but found none", i)
			}

			var id, componentType, componentPath, specificationPath string
			var createdAt int64
			err = rows.Scan(&id, &componentType, &componentPath, &specificationPath, &createdAt)
			if err != nil {
				t.Errorf("[Test %d] Error scanning row: %s", i, err.Error())
			}

			if id != test.metadata.ID {
				t.Errorf("[Test %d] Unexpected component ID: expected=%s, actual=%s", i, test.metadata.ID, id)
			}
			if componentType != test.metadata.ComponentType {
				t.Errorf("[Test %d] Unexpected component ComponentType: expected=%s, actual=%s", i, test.metadata.ComponentType, componentType)
			}
			if componentPath != test.metadata.ComponentPath {
				t.Errorf("[Test %d] Unexpected component ComponentPath: expected=%s, actual=%s", i, test.metadata.ComponentPath, componentPath)
			}
			if specificationPath != test.metadata.SpecificationPath {
				t.Errorf("[Test %d] Unexpected component SpecificationPath: expected=%s, actual=%s", i, test.metadata.SpecificationPath, specificationPath)
			}
			if createdAt != test.metadata.CreatedAt.Unix() {
				t.Errorf("[Test %d] Unexpected component CreatedAt: expected=%d, actual=%d", i, test.metadata.CreatedAt.Unix(), createdAt)
			}
		}
	}

	if rows.Next() {
		t.Fatal("More rows in components table than expected")
	}
}

// TestSelectComponentByID first runs InsertComponent a number of times to load a temporary state
// database with some components. Then it tests various SelectComponentByID scenarios.
func TestSelectComponentByID(t *testing.T) {
	stateDir, err := ioutil.TempDir("", "simplex-select-component-by-id-tests-")
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
	components := make([]ComponentMetadata, 10)
	for i = 0; i < 10; i++ {
		component, err := GenerateComponentMetadata(
			fmt.Sprintf("component-%d", i),
			Task,
			fmt.Sprintf("component-%d-dir", i),
			fmt.Sprintf("component-%d.json", i),
		)
		if err != nil {
			t.Fatalf("[Component %d] Error creating component metadata: %s", i, err.Error())
		}
		components[i] = component
		err = InsertComponent(db, component)
		if err != nil {
			t.Fatalf("[Component %d] Error inserting component into state database: %s", i, err.Error())
		}
	}

	for i = 0; i < 10; i++ {
		stateComponent, err := SelectComponentByID(db, components[i].ID)
		if err != nil {
			t.Errorf("[Test %d] Received error when trying to get inserted component: %s", i, err.Error())
		}
		if stateComponent.ID != components[i].ID {
			t.Errorf("[Test %d] Unexpected ID retrieved from state database: expected=%s, actual=%s", i, components[i].ID, stateComponent.ID)
		}
		if stateComponent.ComponentType != components[i].ComponentType {
			t.Errorf("[Test %d] Unexpected ComponentType retrieved from state database: expected=%s, actual=%s", i, components[i].ComponentType, stateComponent.ComponentType)
		}
		if stateComponent.ComponentPath != components[i].ComponentPath {
			t.Errorf("[Test %d] Unexpected ComponentPath retrieved from state database: expected=%s, actual=%s", i, components[i].ComponentPath, stateComponent.ComponentPath)
		}
		if stateComponent.SpecificationPath != components[i].SpecificationPath {
			t.Errorf("[Test %d] Unexpected SpecificationPath retrieved from state database: expected=%s, actual=%s", i, components[i].SpecificationPath, stateComponent.SpecificationPath)
		}
		expectedCreatedAt := time.Unix(components[i].CreatedAt.Unix(), 0)
		if stateComponent.CreatedAt != expectedCreatedAt {
			t.Errorf("[Test %d] Unexpected CreatedAt retrieved from state database: expected=%s, actual=%s", i, expectedCreatedAt, stateComponent.CreatedAt)
		}
	}

	stateComponent, err := SelectComponentByID(db, "nonexistent-id")
	if err != ErrComponentNotFound {
		t.Error("[Test 11] Was expecting error ErrComponentNotFound for GetComponentByID on unregistered ID, but did not get it")
	}
	if stateComponent.ID != "" {
		t.Errorf("[Test 11] GetComponentByID on unregistered ID returned non-empty ID: %s", stateComponent.ID)
	}
	if stateComponent.ComponentType != "" {
		t.Errorf("[Test 11] GetComponentByID on unregistered ID returned non-empty ComponentType: %s", stateComponent.ComponentType)
	}
	if stateComponent.ComponentPath != "" {
		t.Errorf("[Test 11] GetComponentByID on unregistered ID returned non-empty ComponentPath: %s", stateComponent.ComponentPath)
	}
	if stateComponent.SpecificationPath != "" {
		t.Errorf("[Test 11] GetComponentByID on unregistered ID returned non-empty SpecificationPath: %s", stateComponent.SpecificationPath)
	}
	if !stateComponent.CreatedAt.IsZero() {
		t.Errorf("[Test 11] GetComponentByID on unregistered ID returned non-zero CreatedAt: %v", stateComponent.CreatedAt)
	}
}

// TestDeleteComponentByID first runs InsertComponent a number of times to load a temporary state
// database with some components. Then it tests various DeleteComponentByID scenarios.
func TestDeleteComponentByID(t *testing.T) {
	stateDir, err := ioutil.TempDir("", "simplex-delete-component-by-id-tests-")
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
	components := make([]ComponentMetadata, 10)
	for i = 0; i < 10; i++ {
		component, err := GenerateComponentMetadata(
			fmt.Sprintf("component-%d", i),
			Task,
			fmt.Sprintf("component-%d-dir", i),
			fmt.Sprintf("component-%d.json", i),
		)
		if err != nil {
			t.Fatalf("[Component %d] Error creating component metadata: %s", i, err.Error())
		}
		components[i] = component
		err = InsertComponent(db, component)
		if err != nil {
			t.Fatalf("[Component %d] Error inserting component into state database: %s", i, err.Error())
		}
	}

	err = DeleteComponentByID(db, components[0].ID)
	if err != nil {
		t.Fatalf("[Test 0] Could not delete component: %s", err.Error())
	}

	rows, err := db.Query(selectComponents)
	if err != nil {
		t.Fatalf("Could not select rows from components table: %s", err.Error())
	}
	defer rows.Close()
	for i = 1; i < 10; i++ {
		ok := rows.Next()
		if !ok {
			t.Fatal("Not enough rows in components selection")
		}
		var id, componentType, componentPath, specificationPath string
		var createdAt int64
		err = rows.Scan(&id, &componentType, &componentPath, &specificationPath, &createdAt)
		if err != nil {
			t.Errorf("[Test %d] Could not parse row from components selection: %s", i, err.Error())
		}
		if id != components[i].ID {
			t.Errorf("[Test %d] Unexpected ID from current row in selection: expected=%s, actual=%s", i, components[i].ID, id)
		}
	}
	ok := rows.Next()
	if ok {
		t.Fatal("Too many rows in components selection")
	}

	err = DeleteComponentByID(db, "nonexistent-component-id")
	if err != nil {
		t.Fatal("DeleteComponentByID should not error out when asked to delete a row with a non-existent ID")
	}
}

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

	build := BuildMetadata{
		ID:          "simplex/good:latest",
		ComponentID: "some-component",
		CreatedAt:   time.Now(),
	}

	InsertBuild(db, build)

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
