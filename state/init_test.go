package state

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// State initialization should fail if caller tries to initialize state in an existing directory
func TestInitExistingDirectoryReturnsError(t *testing.T) {
	stateDir, err := ioutil.TempDir("", "simplex-initialize-tests-")
	if err != nil {
		t.Fatalf("Could not create temporary directory: %s", err.Error())
	}
	defer os.RemoveAll(stateDir)

	err = Init(stateDir)
	if err == nil {
		t.Fatal("Initialization attempt over existing directory did not return an error as expected")
	} else if err != ErrStateDirectoryAlreadyExists {
		t.Fatalf("Initialization attempt over existing directory did not return the expected error: expected=%s, actual=%s", err.Error(), ErrStateDirectoryAlreadyExists.Error())
	}
}

// State initialization should behave as expected on a non-existent directory
func TestInit(t *testing.T) {
	// We create a temporary directory and immediately remove it to get a path guaranteed to not
	// exist within the "/tmp" equivalent on the machine running tests.
	stateDir, err := ioutil.TempDir("", "simplex-initialize-tests-")
	if err != nil {
		t.Fatalf("Could not create temporary directory: %s", err.Error())
	}
	os.RemoveAll(stateDir)

	err = Init(stateDir)
	defer os.RemoveAll(stateDir)
	if err != nil {
		t.Fatalf("Expected initialization to complete with no errors. Received error: %s", err.Error())
	}

	stateDBPath := path.Join(stateDir, DBFileName)
	db, err := sql.Open("sqlite3", stateDBPath)
	if err != nil {
		t.Fatal("Error opening state database file")
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		t.Fatalf("Could not ping database: %s", err.Error())
	}

	expectedTables := map[string][]string{
		"components":      {"id", "component_type", "component_path", "specification_path", "created_at"},
		"flows":           {"id", "specification_path", "created_at"},
		"flow_components": {"flow_id", "component_id", "created_at"},
		"builds":          {"id", "component_id", "created_at"},
		"executions":      {"id", "execution_type", "target_id", "created_at"},
	}
	for table, expectedColumns := range expectedTables {
		selection := fmt.Sprintf("SELECT * FROM %s;", table)
		rows, err := db.Query(selection)
		if err != nil {
			t.Errorf("Selection from table %s resulted in error: %s", table, err.Error())
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			t.Errorf("Could not read column names for table %s: %s", table, err.Error())
		}
		if len(columns) != len(expectedColumns) {
			t.Errorf("Unexpected number of columns in table %s: expected=%d, actual=%d", table, len(expectedColumns), len(columns))
		}
		for i, column := range columns {
			if column != expectedColumns[i] {
				t.Errorf("Mismatch between actual and expected column name for column %d in table %s: expected=%s, actual=%s", i, table, expectedColumns[i], column)
			}
		}

		if rows.Next() {
			t.Errorf("Unexpected row in table %s", table)
		}
	}
}
