package builds

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrBuildNotFound - signifies that a single row lookup against the builds table in a state
// database returned no rows
var ErrBuildNotFound = errors.New("Could not find the specified build")

// InsertBuild inserts the build represented by the given build metadata into the given simplex
// state database
func InsertBuild(db *sql.DB, buildMetadata BuildMetadata) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		insertBuild,
		buildMetadata.ID,
		buildMetadata.ComponentID,
		buildMetadata.CreatedAt.Unix(),
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// SelectBuildByID gets build metadata from the given state database using the given ID.
// If no build with the given ID is found, returns ErrBuildNotFound in the error position.
func SelectBuildByID(db *sql.DB, id string) (BuildMetadata, error) {
	var rowID, componentID string
	var createdAt int64
	row := db.QueryRow(selectBuildByID, id)
	err := row.Scan(&rowID, &componentID, &createdAt)
	if err == sql.ErrNoRows {
		return BuildMetadata{}, ErrBuildNotFound
	}
	if err != nil {
		return BuildMetadata{}, err
	}
	if rowID != id {
		return BuildMetadata{}, fmt.Errorf("Result had unexpected row ID: expected=%s, actual=%s", id, rowID)
	}
	return BuildMetadata{ID: rowID, ComponentID: componentID, CreatedAt: time.Unix(createdAt, 0)}, nil
}
