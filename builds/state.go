package builds

import (
	"database/sql"
)

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
