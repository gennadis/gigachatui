package storage

import (
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite" // sqlite driver
)

// NewSqliteDB creates a new sqlite database
func NewSqliteDB(file string) (*sqlx.DB, error) {
	return sqlx.Connect("sqlite", file)
}
