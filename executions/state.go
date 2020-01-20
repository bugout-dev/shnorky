package executions

import (
	"database/sql"
)

// SQL statements
var insertExecutionWithNoFlowID = "INSERT INTO executions (id, build_id, component_id, created_at) VALUES(?, ?, ?, ?);"
var insertExecution = "INSERT INTO executions (id, build_id, component_id, created_at, flow_id) VALUES(?, ?, ?, ?, ?);"

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
