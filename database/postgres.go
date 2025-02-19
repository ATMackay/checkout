package database

import (
	"fmt"

	"gorm.io/driver/postgres"
)

const DBName = "checkout"

// NewPostgresDB returns a new GormDB with PostgreSQL engine.
func NewPostgresDB(host, user, password string, port int) (*GormDB, error) {
	// Open the PostgreSQL connection with constructed  PostgreSQL DSN
	dl := postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC", host, user, password, DBName, port))
	return NewGormDB(dl, false)
}
