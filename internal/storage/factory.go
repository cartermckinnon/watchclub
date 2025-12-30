package storage

import (
	"fmt"
	"strings"
)

// NewStorage creates a new storage implementation based on the provided URI
// Supported URIs:
// - "memory" - in-memory storage
// - "sqlite://path/to/db.db" - SQLite storage
func NewStorage(uri string) (Storage, error) {
	if uri == "" || uri == "memory" {
		return NewMemoryStorage(), nil
	}

	if strings.HasPrefix(uri, "sqlite://") {
		dbPath := strings.TrimPrefix(uri, "sqlite://")
		if dbPath == "" {
			return nil, fmt.Errorf("sqlite URI must include a file path")
		}
		return NewSQLiteStorage(dbPath)
	}

	return nil, fmt.Errorf("unsupported storage URI: %s (supported: memory, sqlite://path)", uri)
}
