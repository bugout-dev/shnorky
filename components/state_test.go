package components

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/simiotics/simplex/state"
	"github.com/simiotics/simplex/utils"
)

// TestInsertComponent tests that component insertion works as expected
func TestInsertComponent(t *testing.T) {
	type InsertComponentTest struct {
		metadata         ComponentMetadata
		shouldThrowError bool
		inSelection      bool
	}

	stateDir, err := utils.TempDir("", "simplex-insert-component-tests-", true)
	if err != nil {
		t.Fatalf("Could not create temporary directory: %s", err.Error())
	}

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

// TestGetComponentByID first runs InsertComponent a number of times to load a temporary state
// database with some components. Then it tests various GetComponentByID scenarios.
func TestGetComponentByID(t *testing.T) {
	stateDir, err := utils.TempDir("", "simplex-get-component-by-id-tests-", true)
	if err != nil {
		t.Fatalf("Could not create temporary directory: %s", err.Error())
	}
	defer os.RemoveAll(stateDir)

	err = state.Init(stateDir)
	if err != nil {
		t.Fatalf("Error creating state directory: %s", err.Error())
	}

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
		stateComponent, err := GetComponentByID(db, components[i].ID)
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

	stateComponent, err := GetComponentByID(db, "nonexistent-id")
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
