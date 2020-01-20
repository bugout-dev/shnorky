package flows

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrFlowNotFound - signifies that a single row lookup against a state database returned
// no rows
var ErrFlowNotFound = errors.New("Could not find the specified flow")

var insertFlow = "INSERT INTO flows (id, specification_path, created_at) VALUES(?, ?, ?);"
var selectFlowByID = "SELECT * FROM flows WHERE id=?;"

// InsertFlow creates a new row in the components table with the given component information.
func InsertFlow(db *sql.DB, component FlowMetadata) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		insertFlow,
		component.ID,
		component.SpecificationPath,
		component.CreatedAt.Unix(),
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

// SelectFlowByID gets flow metadata from the given state database using the given ID.
// If no flow with the given ID is found, returns ErrFlowNotFound in the error position.
func SelectFlowByID(db *sql.DB, id string) (FlowMetadata, error) {
	var rowID, specificationPath string
	var createdAt int64
	row := db.QueryRow(selectFlowByID, id)
	err := row.Scan(&rowID, &specificationPath, &createdAt)
	if err == sql.ErrNoRows {
		return FlowMetadata{}, ErrFlowNotFound
	}
	if err != nil {
		return FlowMetadata{}, err
	}
	if rowID != id {
		return FlowMetadata{}, fmt.Errorf("Result had unexpected row ID: expected=%s, actual=%s", id, rowID)
	}
	return FlowMetadata{ID: rowID, SpecificationPath: specificationPath, CreatedAt: time.Unix(createdAt, 0)}, nil
}
