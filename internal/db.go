package internal

import (
	"database/sql"
	"path"

	"github.com/simiotics/shnorky/state"
	"github.com/sirupsen/logrus"
)

// OpenStateDB opens a connection to the state database in the given state directory.
// If there is an error opening the database, fatally errors out.
func OpenStateDB(stateDir string, log *logrus.Logger) *sql.DB {
	stateDBPath := path.Join(stateDir, state.DBFileName)
	db, err := sql.Open("sqlite3", stateDBPath)
	if err != nil {
		log.WithFields(logrus.Fields{"stateDBPath": stateDBPath, "error": err}).Fatal("Error opening state database")
	}
	return db
}
