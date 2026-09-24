package sqlite

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

const driverName = "sqlite"

// Open opens a SQLite database through the modernc.org/sqlite driver.
//
// The caller owns the returned database lifetime and transaction boundaries.
func Open(dataSourceName string) (*sql.DB, error) {
	return sql.Open(driverName, dataSourceName)
}
