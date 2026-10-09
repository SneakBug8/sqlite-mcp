package repository

import (
	"fmt"
	"os"
	"sync"

	"github.com/rvarun11/sqlite-mcp/internal/models"
	"go.uber.org/zap"
)

type Repository interface {
	GetSchema() ([]models.Table, error)
	Query(sqlQuery string) (*models.QueryResult, error)
	Execute(sqlQuery string) (*models.ExecuteResult, error)
	Close() error
}

// Manager opens and caches database connections by file path. Each tool call
// selects the database to explore with the required "database" argument.
type Manager struct {
	mu     sync.Mutex
	dbs    map[string]Repository
	logger *zap.SugaredLogger
}

func NewManager(logger *zap.SugaredLogger) *Manager {
	return &Manager{
		dbs:    make(map[string]Repository),
		logger: logger,
	}
}

// Get returns a cached connection for dbPath, or opens a new one.
func (m *Manager) Get(dbPath string) (Repository, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("database path is required")
	}

	info, err := os.Stat(dbPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access database file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("database path is a directory, not a file")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if db, ok := m.dbs[dbPath]; ok {
		return db, nil
	}

	db, err := NewSQLiteDB(dbPath, m.logger)
	if err != nil {
		return nil, err
	}
	m.dbs[dbPath] = db

	return db, nil
}

// Close closes every open database connection.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for path, db := range m.dbs {
		if err := db.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close database %s: %w", path, err)
		}
	}
	m.dbs = make(map[string]Repository)

	return firstErr
}
