package sqlite

import (
	"database/sql"

	"github.com/alokMIPL/students-api/internal/config"
)

type Sqlite struct {
	Db *sql.DB
}

type Sqlites struct {
	Db *sql.DB
}

func New(cfg config.Config) (*Sqlite, error) {
	db, err := sql.Open("sqlite3", cfg.StoragePath)
	if err != nil {
		return nil, err
	}
}
