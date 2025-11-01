//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupPostgresContainer starts a PostgreSQL container for testing
func setupPostgresContainer(t *testing.T) (string, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start PostgreSQL container")

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable", host, port.Port())

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return dsn, cleanup
}

// TestPostgreSQL_Integration_Connection tests database connection and setup
func TestPostgreSQL_Integration_Connection(t *testing.T) {
	dsn, cleanup := setupPostgresContainer(t)
	defer cleanup()

	// Test connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to PostgreSQL")

	// Get underlying sql.DB
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	// Test ping
	err = sqlDB.Ping()
	assert.NoError(t, err, "Failed to ping database")

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Verify settings
	stats := sqlDB.Stats()
	assert.LessOrEqual(t, stats.Idle, 10, "Idle connections should not exceed max idle")
	assert.GreaterOrEqual(t, stats.Idle, 0, "Idle connections should be non-negative")
}

// TestPostgreSQL_Integration_AutoMigrate tests schema migration
func TestPostgreSQL_Integration_AutoMigrate(t *testing.T) {
	dsn, cleanup := setupPostgresContainer(t)
	defer cleanup()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Run auto migration
	err = db.AutoMigrate(&RabbitLog{})
	require.NoError(t, err, "Auto migration should succeed")

	// Verify table exists
	var tableExists bool
	err = db.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'rabbit_logs')").Scan(&tableExists).Error
	require.NoError(t, err)
	assert.True(t, tableExists, "rabbit_logs table should exist")

	// Verify columns
	var columns []string
	err = db.Raw("SELECT column_name FROM information_schema.columns WHERE table_name = 'rabbit_logs' ORDER BY ordinal_position").Pluck("column_name", &columns).Error
	require.NoError(t, err)

	expectedColumns := []string{"id", "created_at", "updated_at", "deleted_at", "document_id", "state", "version", "log"}
	assert.Equal(t, expectedColumns, columns, "Table should have expected columns")
}

// TestPostgreSQL_Integration_CreateRabbitLog tests creating log entries
func TestPostgreSQL_Integration_CreateRabbitLog(t *testing.T) {
	dsn, cleanup := setupPostgresContainer(t)
	defer cleanup()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Migrate schema
	err = db.AutoMigrate(&RabbitLog{})
	require.NoError(t, err)

	t.Run("create single log entry", func(t *testing.T) {
		log := RabbitLog{
			DocumentID: "doc-001",
			State:      "started",
			Version:    "v1.0.0",
			Log:        []byte("Processing started"),
		}

		result := db.Create(&log)
		require.NoError(t, result.Error)
		assert.Equal(t, int64(1), result.RowsAffected)
		assert.NotZero(t, log.ID, "ID should be auto-generated")
		assert.NotZero(t, log.CreatedAt, "CreatedAt should be auto-set")
	})

	t.Run("create multiple log entries", func(t *testing.T) {
		logs := []RabbitLog{
			{DocumentID: "doc-002", State: "running", Version: "v1.0.0", Log: []byte("Step 1")},
			{DocumentID: "doc-002", State: "running", Version: "v1.0.0", Log: []byte("Step 2")},
			{DocumentID: "doc-002", State: "completed", Version: "v1.0.0", Log: []byte("Finished")},
		}

		result := db.Create(&logs)
		require.NoError(t, result.Error)
		assert.Equal(t, int64(3), result.RowsAffected)
	})
}

// TestPostgreSQL_Integration_QueryRabbitLogs tests querying log entries
func TestPostgreSQL_Integration_QueryRabbitLogs(t *testing.T) {
	dsn, cleanup := setupPostgresContainer(t)
	defer cleanup()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&RabbitLog{})
	require.NoError(t, err)

	// Create test data
	testLogs := []RabbitLog{
		{DocumentID: "doc-query-001", State: "started", Version: "v1.0.0"},
		{DocumentID: "doc-query-001", State: "running", Version: "v1.0.0"},
		{DocumentID: "doc-query-001", State: "completed", Version: "v1.0.0"},
		{DocumentID: "doc-query-002", State: "started", Version: "v1.0.0"},
		{DocumentID: "doc-query-002", State: "failed", Version: "v1.0.0"},
	}
	db.Create(&testLogs)

	t.Run("find by document ID", func(t *testing.T) {
		var logs []RabbitLog
		result := db.Where("document_id = ?", "doc-query-001").Find(&logs)
		require.NoError(t, result.Error)
		assert.Len(t, logs, 3, "Should find 3 logs for doc-query-001")
	})

	t.Run("find by state", func(t *testing.T) {
		var logs []RabbitLog
		result := db.Where("state = ?", "started").Find(&logs)
		require.NoError(t, result.Error)
		assert.Len(t, logs, 2, "Should find 2 logs with state 'started'")
	})

	t.Run("find with ordering", func(t *testing.T) {
		var logs []RabbitLog
		result := db.Where("document_id = ?", "doc-query-001").Order("created_at asc").Find(&logs)
		require.NoError(t, result.Error)
		assert.Equal(t, "started", logs[0].State)
		assert.Equal(t, "running", logs[1].State)
		assert.Equal(t, "completed", logs[2].State)
	})

	t.Run("count logs", func(t *testing.T) {
		var count int64
		result := db.Model(&RabbitLog{}).Where("state = ?", "completed").Count(&count)
		require.NoError(t, result.Error)
		assert.Equal(t, int64(1), count)
	})
}

// TestPostgreSQL_Integration_UpdateRabbitLog tests updating log entries
func TestPostgreSQL_Integration_UpdateRabbitLog(t *testing.T) {
	dsn, cleanup := setupPostgresContainer(t)
	defer cleanup()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&RabbitLog{})
	require.NoError(t, err)

	log := RabbitLog{
		DocumentID: "doc-update-001",
		State:      "started",
		Version:    "v1.0.0",
		Log:        []byte("Initial log"),
	}
	db.Create(&log)

	t.Run("update state", func(t *testing.T) {
		log.State = "completed"
		result := db.Save(&log)
		require.NoError(t, result.Error)

		var updated RabbitLog
		db.First(&updated, log.ID)
		assert.Equal(t, "completed", updated.State)
		assert.True(t, updated.UpdatedAt.After(updated.CreatedAt), "UpdatedAt should be later than CreatedAt")
	})

	t.Run("update log data", func(t *testing.T) {
		log.Log = []byte("Updated log message")
		result := db.Save(&log)
		require.NoError(t, result.Error)

		var updated RabbitLog
		db.First(&updated, log.ID)
		assert.Equal(t, []byte("Updated log message"), updated.Log)
	})
}

// TestPostgreSQL_Integration_DeleteRabbitLog tests soft delete
func TestPostgreSQL_Integration_DeleteRabbitLog(t *testing.T) {
	dsn, cleanup := setupPostgresContainer(t)
	defer cleanup()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&RabbitLog{})
	require.NoError(t, err)

	log := RabbitLog{
		DocumentID: "doc-delete-001",
		State:      "completed",
		Version:    "v1.0.0",
	}
	db.Create(&log)

	t.Run("soft delete", func(t *testing.T) {
		result := db.Delete(&log)
		require.NoError(t, result.Error)
		assert.Equal(t, int64(1), result.RowsAffected)

		// Verify it's not found in normal query
		var found RabbitLog
		result = db.First(&found, log.ID)
		assert.Error(t, result.Error, "Should not find soft-deleted record")

		// Verify it exists with Unscoped
		result = db.Unscoped().First(&found, log.ID)
		require.NoError(t, result.Error)
		assert.True(t, found.DeletedAt.Valid, "DeletedAt should be set")
	})

	t.Run("permanent delete", func(t *testing.T) {
		log2 := RabbitLog{
			DocumentID: "doc-delete-002",
			State:      "failed",
			Version:    "v1.0.0",
		}
		db.Create(&log2)

		// Permanent delete
		result := db.Unscoped().Delete(&log2)
		require.NoError(t, result.Error)

		// Verify it's completely gone
		var found RabbitLog
		result = db.Unscoped().First(&found, log2.ID)
		assert.Error(t, result.Error, "Should not find permanently deleted record")
	})
}

// TestPostgreSQL_Integration_BinaryLogData tests handling of binary log data
func TestPostgreSQL_Integration_BinaryLogData(t *testing.T) {
	dsn, cleanup := setupPostgresContainer(t)
	defer cleanup()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&RabbitLog{})
	require.NoError(t, err)

	t.Run("store and retrieve binary data", func(t *testing.T) {
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		log := RabbitLog{
			DocumentID: "doc-binary-001",
			State:      "completed",
			Version:    "v1.0.0",
			Log:        binaryData,
		}

		db.Create(&log)

		var retrieved RabbitLog
		db.First(&retrieved, log.ID)
		assert.Equal(t, binaryData, retrieved.Log, "Binary data should be preserved")
	})

	t.Run("store large log data", func(t *testing.T) {
		// Create 10KB of log data
		largeData := make([]byte, 10*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		log := RabbitLog{
			DocumentID: "doc-large-001",
			State:      "completed",
			Version:    "v1.0.0",
			Log:        largeData,
		}

		db.Create(&log)

		var retrieved RabbitLog
		db.First(&retrieved, log.ID)
		assert.Equal(t, largeData, retrieved.Log, "Large binary data should be preserved")
		assert.Len(t, retrieved.Log, 10*1024)
	})
}

// TestPostgreSQL_Integration_Transactions tests transaction support
func TestPostgreSQL_Integration_Transactions(t *testing.T) {
	dsn, cleanup := setupPostgresContainer(t)
	defer cleanup()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&RabbitLog{})
	require.NoError(t, err)

	t.Run("successful transaction", func(t *testing.T) {
		err := db.Transaction(func(tx *gorm.DB) error {
			log1 := RabbitLog{DocumentID: "doc-tx-001", State: "started", Version: "v1.0.0"}
			if err := tx.Create(&log1).Error; err != nil {
				return err
			}

			log2 := RabbitLog{DocumentID: "doc-tx-002", State: "started", Version: "v1.0.0"}
			if err := tx.Create(&log2).Error; err != nil {
				return err
			}

			return nil
		})

		require.NoError(t, err)

		// Verify both records exist
		var count int64
		db.Model(&RabbitLog{}).Where("document_id IN ?", []string{"doc-tx-001", "doc-tx-002"}).Count(&count)
		assert.Equal(t, int64(2), count)
	})

	t.Run("rolled back transaction", func(t *testing.T) {
		err := db.Transaction(func(tx *gorm.DB) error {
			log := RabbitLog{DocumentID: "doc-tx-rollback", State: "started", Version: "v1.0.0"}
			if err := tx.Create(&log).Error; err != nil {
				return err
			}

			// Force rollback
			return fmt.Errorf("simulated error")
		})

		assert.Error(t, err)

		// Verify record doesn't exist
		var found RabbitLog
		result := db.Where("document_id = ?", "doc-tx-rollback").First(&found)
		assert.Error(t, result.Error, "Record should not exist after rollback")
	})
}
