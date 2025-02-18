package database

import "gorm.io/driver/sqlite"

const InMemoryDSN = ":memory:"

// NewSQLiteDB returns a new GormDB with SQLite engine.
func NewSQLiteDB(databaseDSN string) (*GormDB, error) {
	dl := sqlite.Open(databaseDSN)
	return NewGormDB(dl)
}
