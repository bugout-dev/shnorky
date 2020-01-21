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

// ErrBuildNotFound - signifies that a single row lookup against the builds table in a state
// database returned no rows
var ErrBuildNotFound = errors.New("Could not find the specified build")

// SQL statements
var insertComponent = "INSERT INTO components (id, component_type, component_path, specification_path, created_at) VALUES(?, ?, ?, ?, ?);"
var selectComponents = "SELECT * FROM components;"
var selectComponentByID = "SELECT * FROM components WHERE id=?;"
var deleteComponentByID = "DELETE FROM components WHERE id=?;"
var insertBuild = "INSERT INTO builds (id, component_id, created_at) VALUES(?, ?, ?);"
var selectBuilds = "SELECT * FROM builds;"
var selectBuildByID = "SELECT * FROM builds WHERE id=?;"
var selectBuildsByComponentID = "SELECT * FROM builds WHERE component_id=?;"
var selectMostRecentBuildForComponent = "SELECT * FROM builds WHERE component_id=? ORDER BY created_at DESC LIMIT 1;"
var deleteBuildByID = "DELETE FROM builds WHERE id=?;"
var deleteBuildsByComponentID = "DELETE FROM builds WHERE component_id=?"
var insertExecutionWithNoFlowID = "INSERT INTO executions (id, build_id, component_id, created_at) VALUES(?, ?, ?, ?);"
var insertExecution = "INSERT INTO executions (id, build_id, component_id, created_at, flow_id) VALUES(?, ?, ?, ?, ?);"

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

// SelectComponentByID gets component metadata from the given state database using the given ID.
// If no component with the given ID is found, returns ErrComponentNotFound in the error position.
func SelectComponentByID(db *sql.DB, id string) (ComponentMetadata, error) {
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

// DeleteComponentByID creates a new row in the components table with the given component information.
func DeleteComponentByID(db *sql.DB, id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(deleteComponentByID, id)
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

// SelectMostRecentBuildForComponent gets build metadata from the given state database for the most
// recent build for the component with the given componentID
func SelectMostRecentBuildForComponent(db *sql.DB, componentID string) (BuildMetadata, error) {
	var id, rowComponentID string
	var createdAt int64
	row := db.QueryRow(selectMostRecentBuildForComponent, componentID)
	err := row.Scan(&id, &rowComponentID, &createdAt)
	if err == sql.ErrNoRows {
		return BuildMetadata{}, ErrBuildNotFound
	}
	if err != nil {
		return BuildMetadata{}, err
	}
	if rowComponentID != componentID {
		return BuildMetadata{}, fmt.Errorf("Result had unexpected component ID: expected=%s, actual=%s", componentID, rowComponentID)
	}
	return BuildMetadata{ID: id, ComponentID: rowComponentID, CreatedAt: time.Unix(createdAt, 0)}, nil
}

// InsertExecution inserts an execution row into the state database
func InsertExecution(db *sql.DB, executionMetadata ExecutionMetadata) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if executionMetadata.FlowID == "" {
		_, err = tx.Exec(
			insertExecutionWithNoFlowID,
			executionMetadata.ID,
			executionMetadata.BuildID,
			executionMetadata.ComponentID,
			executionMetadata.CreatedAt.Unix(),
		)
	} else {
		_, err = tx.Exec(
			insertExecution,
			executionMetadata.ID,
			executionMetadata.BuildID,
			executionMetadata.ComponentID,
			executionMetadata.CreatedAt.Unix(),
			executionMetadata.FlowID,
		)
	}
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
