package database

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_SQLite_Database(t *testing.T) {
	t.Run("in-memory", func(t *testing.T) {
		d, err := NewSQLiteDB(InMemoryDSN)
		require.NoError(t, err)
		require.NotNil(t, d)
	})
	t.Run("file-db", func(t *testing.T) {
		dir := t.TempDir()
		d, err := NewSQLiteDB(filepath.Join(dir, "sqlite"))
		require.NoError(t, err)
		require.NotNil(t, d)
	})
}
