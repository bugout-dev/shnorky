package state

import (
	"database/sql"
	"errors"
	"os"
	"path"

	// sqlite3 driver registered under database/sql on import
	_ "github.com/mattn/go-sqlite3"
)

// DBFileName - Name of SQLite database representing state in the state directory
var DBFileName = "state.sqlite"

// ErrStateDirectoryAlreadyExists - Error returned by Init if a filesystem object already exists at
// the desired state directory path
var ErrStateDirectoryAlreadyExists = errors.New("The given state directory already exists")

// Init initializes a fresh state directory at the given path.
// If an object already exists at the given path on the filesystem, or if Init encounters any
// issues in creating a directory at that path (for example if the process it runs in does hot have
// sufficient permissions), this function returns a non-nil error.
func Init(stateDir string) error {
	_, err := os.Stat(stateDir)
	if err == nil {
		return ErrStateDirectoryAlreadyExists
	}
	if !os.IsNotExist(err) {
		return err
	}

	err = os.MkdirAll(stateDir, 0744)
	if err != nil {
		return err
	}

	stateDBPath := path.Join(stateDir, DBFileName)
	db, err := sql.Open("sqlite3", stateDBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(createTables)
	if err != nil {
		return err
	}

	return nil
}
