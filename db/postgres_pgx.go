package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDB wraps PostgreSQL connection pool with helper methods using pgx driver.
// This provides a lightweight alternative to GORM for applications that need
// direct SQL access with connection pooling.
//
// Use Cases:
//   - High-performance metric storage
//   - Time-series data operations
//   - Custom SQL queries
//   - Bulk operations
//
// Comparison to GORM:
//   - Faster for bulk operations
//   - No ORM overhead
//   - Direct SQL control
//   - Better for time-series workloads
type PostgresDB struct {
	pool *pgxpool.Pool
}

// NewPostgresDB creates a new PostgreSQL database connection using pgx.
// The connection string format is standard PostgreSQL:
//
//	postgresql://[user[:password]@][host][:port][/dbname][?param1=value1&...]
//
// Example:
//
//	db, err := NewPostgresDB("postgresql://user:pass@localhost:5432/mydb?sslmode=disable")
//
// Connection Pooling:
//   - Automatic connection pooling via pgxpool
//   - Default pool configuration applied
//   - Configurable via connection string parameters
func NewPostgresDB(connString string) (*PostgresDB, error) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresDB{pool: pool}, nil
}

// Close closes the database connection pool.
func (db *PostgresDB) Close() {
	db.pool.Close()
}

// Exec executes a SQL statement.
// Returns error if execution fails.
func (db *PostgresDB) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := db.pool.Exec(ctx, sql, args...)
	return err
}

// Query executes a query that returns rows.
// Caller must call rows.Close() when done.
func (db *PostgresDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns a single row.
// Row scanning should be done immediately as the connection is released after scanning.
func (db *PostgresDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

// Pool returns the underlying connection pool for advanced operations.
// Use this for transactions, batch operations, or custom connection management.
func (db *PostgresDB) Pool() *pgxpool.Pool {
	return db.pool
}
