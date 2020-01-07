package components

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrComponentNotFound - signifies that a single row lookup against a state database returned
// no rows
var ErrComponentNotFound = errors.New("Could not find the specified component")

// InsertComponent creates a new row in the components table with the given component information.
func InsertComponent(db *sql.DB, component ComponentMetadata) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		insertComponent,
		component.ID,
		component.ComponentType,
		component.ComponentPath,
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

// GetComponentByID gets component metadata from the given state database using the given ID.
// If no component with the given ID is found, returns ErrComponentNotFound in the error position.
func GetComponentByID(db *sql.DB, id string) (ComponentMetadata, error) {
	var rowID, componentType, componentPath, specificationPath string
	var createdAt int64
	row := db.QueryRow(selectComponentByID, id)
	err := row.Scan(&rowID, &componentType, &componentPath, &specificationPath, &createdAt)
	if err == sql.ErrNoRows {
		return ComponentMetadata{}, ErrComponentNotFound
	}
	if err != nil {
		return ComponentMetadata{}, err
	}
	if rowID != id {
		return ComponentMetadata{}, fmt.Errorf("Result had unexpected row ID: expected=%s, actual=%s", id, rowID)
	}
	return ComponentMetadata{ID: rowID, ComponentType: componentType, ComponentPath: componentPath, SpecificationPath: specificationPath, CreatedAt: time.Unix(createdAt, 0)}, nil
}
