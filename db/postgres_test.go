package db

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestRabbitLog_Structure tests the RabbitLog model structure
func TestRabbitLog_Structure(t *testing.T) {
	t.Run("complete rabbit log", func(t *testing.T) {
		now := time.Now()
		log := RabbitLog{
			Model: gorm.Model{
				ID:        1,
				CreatedAt: now,
				UpdatedAt: now,
			},
			DocumentID: "doc-12345",
			State:      "completed",
			Version:    "v1.0.0",
			Log:        []byte("processing completed successfully"),
		}

		assert.Equal(t, uint(1), log.ID)
		assert.Equal(t, "doc-12345", log.DocumentID)
		assert.Equal(t, "completed", log.State)
		assert.Equal(t, "v1.0.0", log.Version)
		assert.NotEmpty(t, log.Log)
	})

	t.Run("empty log entry", func(t *testing.T) {
		log := RabbitLog{}

		assert.Equal(t, uint(0), log.ID)
		assert.Empty(t, log.DocumentID)
		assert.Empty(t, log.State)
		assert.Empty(t, log.Version)
		assert.Nil(t, log.Log)
	})

	t.Run("log with binary data", func(t *testing.T) {
		binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF}
		log := RabbitLog{
			DocumentID: "doc-binary",
			State:      "processing",
			Version:    "v2.0.0",
			Log:        binaryData,
		}

		assert.Equal(t, binaryData, log.Log)
		assert.Len(t, log.Log, 5)
	})
}

// TestRabbitLog_StateValues tests different state values
func TestRabbitLog_StateValues(t *testing.T) {
	states := []string{
		"started",
		"initialized",
		"running",
		"processing",
		"completed",
		"failed",
		"error",
	}

	for _, state := range states {
		t.Run(state, func(t *testing.T) {
			log := RabbitLog{
				DocumentID: "doc-test",
				State:      state,
				Version:    "v1.0.0",
			}

			assert.Equal(t, state, log.State)
		})
	}
}

// TestRabbitLog_JSONSerialization tests JSON marshaling and unmarshaling
func TestRabbitLog_JSONSerialization(t *testing.T) {
	t.Run("marshal to JSON", func(t *testing.T) {
		log := RabbitLog{
			Model: gorm.Model{
				ID:        42,
				CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				UpdatedAt: time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC),
			},
			DocumentID: "doc-json-test",
			State:      "completed",
			Version:    "v1.2.3",
			Log:        []byte("test log data"),
		}

		data, err := json.Marshal(log)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, "doc-json-test")
		assert.Contains(t, jsonStr, "completed")
		assert.Contains(t, jsonStr, "v1.2.3")
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{
			"ID": 1,
			"CreatedAt": "2024-01-15T10:30:00Z",
			"UpdatedAt": "2024-01-15T10:35:00Z",
			"DeletedAt": null,
			"DocumentID": "doc-unmarshal",
			"State": "running",
			"Version": "v2.0.0",
			"Log": "dGVzdCBsb2cgZGF0YQ=="
		}`

		var log RabbitLog
		err := json.Unmarshal([]byte(jsonData), &log)
		require.NoError(t, err)

		assert.Equal(t, uint(1), log.ID)
		assert.Equal(t, "doc-unmarshal", log.DocumentID)
		assert.Equal(t, "running", log.State)
		assert.Equal(t, "v2.0.0", log.Version)
	})

	t.Run("marshal array of logs", func(t *testing.T) {
		logs := []RabbitLog{
			{
				DocumentID: "doc-1",
				State:      "completed",
				Version:    "v1.0.0",
			},
			{
				DocumentID: "doc-2",
				State:      "failed",
				Version:    "v1.0.0",
			},
		}

		data, err := json.Marshal(logs)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		var decoded []RabbitLog
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Len(t, decoded, 2)
		assert.Equal(t, "doc-1", decoded[0].DocumentID)
		assert.Equal(t, "doc-2", decoded[1].DocumentID)
	})
}

// TestRabbitLog_Base64Encoding tests base64 encoding of log data
func TestRabbitLog_Base64Encoding(t *testing.T) {
	t.Run("encode log data", func(t *testing.T) {
		originalData := []byte("This is test log data with special chars: !@#$%^&*()")
		encoded := base64.StdEncoding.EncodeToString(originalData)

		assert.NotEmpty(t, encoded)
		assert.NotEqual(t, string(originalData), encoded)

		// Verify it can be decoded back
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		require.NoError(t, err)
		assert.Equal(t, originalData, decoded)
	})

	t.Run("encode binary data", func(t *testing.T) {
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
		encoded := base64.StdEncoding.EncodeToString(binaryData)

		decoded, err := base64.StdEncoding.DecodeString(encoded)
		require.NoError(t, err)
		assert.Equal(t, binaryData, decoded)
	})

	t.Run("encode empty data", func(t *testing.T) {
		emptyData := []byte{}
		encoded := base64.StdEncoding.EncodeToString(emptyData)

		assert.Equal(t, "", encoded)

		decoded, err := base64.StdEncoding.DecodeString(encoded)
		require.NoError(t, err)
		assert.Empty(t, decoded)
	})

	t.Run("encode large data", func(t *testing.T) {
		largeData := make([]byte, 10000)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		encoded := base64.StdEncoding.EncodeToString(largeData)
		assert.NotEmpty(t, encoded)

		decoded, err := base64.StdEncoding.DecodeString(encoded)
		require.NoError(t, err)
		assert.Equal(t, largeData, decoded)
	})
}

// TestRabbitLog_VersionFormats tests various version format strings
func TestRabbitLog_VersionFormats(t *testing.T) {
	versions := []string{
		"v1.0.0",
		"v2.5.3",
		"1.0",
		"2024-01-15",
		"latest",
		"snapshot-20240115",
		"dev",
	}

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			log := RabbitLog{
				DocumentID: "doc-version-test",
				State:      "completed",
				Version:    version,
			}

			assert.Equal(t, version, log.Version)
		})
	}
}

// TestRabbitLog_DocumentIDFormats tests various document ID formats
func TestRabbitLog_DocumentIDFormats(t *testing.T) {
	documentIDs := []string{
		"doc-12345",
		"uuid-123e4567-e89b-12d3-a456-426614174000",
		"process-2024-01-15-001",
		"file_name_with_underscores",
		"UPPERCASE-DOC-ID",
		"mixed-Case-123",
	}

	for _, docID := range documentIDs {
		t.Run(docID, func(t *testing.T) {
			log := RabbitLog{
				DocumentID: docID,
				State:      "processing",
				Version:    "v1.0.0",
			}

			assert.Equal(t, docID, log.DocumentID)
		})
	}
}

// TestRabbitLog_Timestamps tests timestamp handling
func TestRabbitLog_Timestamps(t *testing.T) {
	t.Run("creation timestamp", func(t *testing.T) {
		createdAt := time.Now()
		log := RabbitLog{
			Model: gorm.Model{
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
			},
			DocumentID: "doc-time-test",
			State:      "started",
			Version:    "v1.0.0",
		}

		assert.Equal(t, createdAt, log.CreatedAt)
		assert.Equal(t, createdAt, log.UpdatedAt)
	})

	t.Run("update timestamp progression", func(t *testing.T) {
		createdAt := time.Now().Add(-1 * time.Hour)
		updatedAt := time.Now()

		log := RabbitLog{
			Model: gorm.Model{
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			DocumentID: "doc-update-test",
			State:      "completed",
			Version:    "v1.0.0",
		}

		assert.True(t, log.UpdatedAt.After(log.CreatedAt))
		assert.Equal(t, updatedAt, log.UpdatedAt)
	})

	t.Run("soft delete timestamp", func(t *testing.T) {
		deletedAt := time.Now()
		log := RabbitLog{
			Model: gorm.Model{
				DeletedAt: gorm.DeletedAt{
					Time:  deletedAt,
					Valid: true,
				},
			},
			DocumentID: "doc-deleted",
			State:      "archived",
			Version:    "v1.0.0",
		}

		assert.True(t, log.DeletedAt.Valid)
		assert.Equal(t, deletedAt, log.DeletedAt.Time)
	})
}

// TestRabbitLog_SoftDelete tests soft delete functionality
func TestRabbitLog_SoftDelete(t *testing.T) {
	t.Run("non-deleted record", func(t *testing.T) {
		log := RabbitLog{
			DocumentID: "doc-active",
			State:      "running",
			Version:    "v1.0.0",
		}

		assert.False(t, log.DeletedAt.Valid)
	})

	t.Run("soft deleted record", func(t *testing.T) {
		log := RabbitLog{
			Model: gorm.Model{
				DeletedAt: gorm.DeletedAt{
					Time:  time.Now(),
					Valid: true,
				},
			},
			DocumentID: "doc-deleted",
			State:      "archived",
			Version:    "v1.0.0",
		}

		assert.True(t, log.DeletedAt.Valid)
		assert.NotZero(t, log.DeletedAt.Time)
	})
}

// TestRabbitLog_EmptyFields tests handling of empty or nil fields
func TestRabbitLog_EmptyFields(t *testing.T) {
	t.Run("empty document ID", func(t *testing.T) {
		log := RabbitLog{
			DocumentID: "",
			State:      "pending",
			Version:    "v1.0.0",
		}

		assert.Empty(t, log.DocumentID)
	})

	t.Run("empty state", func(t *testing.T) {
		log := RabbitLog{
			DocumentID: "doc-123",
			State:      "",
			Version:    "v1.0.0",
		}

		assert.Empty(t, log.State)
	})

	t.Run("empty version", func(t *testing.T) {
		log := RabbitLog{
			DocumentID: "doc-123",
			State:      "running",
			Version:    "",
		}

		assert.Empty(t, log.Version)
	})

	t.Run("nil log data", func(t *testing.T) {
		log := RabbitLog{
			DocumentID: "doc-123",
			State:      "started",
			Version:    "v1.0.0",
			Log:        nil,
		}

		assert.Nil(t, log.Log)
	})
}

// TestRabbitLog_LogDataTypes tests different types of log data
func TestRabbitLog_LogDataTypes(t *testing.T) {
	t.Run("text log data", func(t *testing.T) {
		textData := "Simple text log message"
		log := RabbitLog{
			DocumentID: "doc-text",
			State:      "completed",
			Version:    "v1.0.0",
			Log:        []byte(textData),
		}

		assert.Equal(t, textData, string(log.Log))
	})

	t.Run("JSON log data", func(t *testing.T) {
		jsonData := `{"status":"success","result":"data processed"}`
		log := RabbitLog{
			DocumentID: "doc-json",
			State:      "completed",
			Version:    "v1.0.0",
			Log:        []byte(jsonData),
		}

		var parsed map[string]interface{}
		err := json.Unmarshal(log.Log, &parsed)
		require.NoError(t, err)
		assert.Equal(t, "success", parsed["status"])
	})

	t.Run("multiline log data", func(t *testing.T) {
		multilineData := `Line 1
Line 2
Line 3`
		log := RabbitLog{
			DocumentID: "doc-multiline",
			State:      "completed",
			Version:    "v1.0.0",
			Log:        []byte(multilineData),
		}

		assert.Contains(t, string(log.Log), "Line 1")
		assert.Contains(t, string(log.Log), "Line 2")
		assert.Contains(t, string(log.Log), "Line 3")
	})

	t.Run("unicode log data", func(t *testing.T) {
		unicodeData := "Unicode test: 你好 🚀 Ñoño"
		log := RabbitLog{
			DocumentID: "doc-unicode",
			State:      "completed",
			Version:    "v1.0.0",
			Log:        []byte(unicodeData),
		}

		assert.Equal(t, unicodeData, string(log.Log))
	})
}

// TestRabbitLog_IDAutoIncrement tests ID field behavior
func TestRabbitLog_IDAutoIncrement(t *testing.T) {
	t.Run("zero ID for new record", func(t *testing.T) {
		log := RabbitLog{
			DocumentID: "doc-new",
			State:      "started",
			Version:    "v1.0.0",
		}

		assert.Equal(t, uint(0), log.ID)
	})

	t.Run("assigned ID", func(t *testing.T) {
		log := RabbitLog{
			Model: gorm.Model{
				ID: 42,
			},
			DocumentID: "doc-existing",
			State:      "running",
			Version:    "v1.0.0",
		}

		assert.Equal(t, uint(42), log.ID)
	})
}

// TestRabbitLog_StateTransitions tests state transition scenarios
func TestRabbitLog_StateTransitions(t *testing.T) {
	t.Run("typical state progression", func(t *testing.T) {
		states := []string{"started", "running", "processing", "completed"}

		for _, state := range states {
			log := RabbitLog{
				DocumentID: "doc-transition",
				State:      state,
				Version:    "v1.0.0",
			}

			assert.Equal(t, state, log.State)
		}
	})

	t.Run("error state transition", func(t *testing.T) {
		log := RabbitLog{
			DocumentID: "doc-error",
			State:      "failed",
			Version:    "v1.0.0",
			Log:        []byte("Error: processing failed due to timeout"),
		}

		assert.Equal(t, "failed", log.State)
		assert.Contains(t, string(log.Log), "Error")
	})
}

// TestRabbitLog_ComplexScenarios tests complex real-world scenarios
func TestRabbitLog_ComplexScenarios(t *testing.T) {
	t.Run("complete processing lifecycle", func(t *testing.T) {
		// Initial log entry
		log := RabbitLog{
			Model: gorm.Model{
				ID:        1,
				CreatedAt: time.Now(),
			},
			DocumentID: "doc-lifecycle",
			State:      "started",
			Version:    "v1.0.0",
			Log:        []byte("Processing initiated"),
		}

		assert.Equal(t, "started", log.State)

		// Simulate update to running state
		log.State = "running"
		log.UpdatedAt = time.Now()
		log.Log = []byte("Processing in progress...")

		assert.Equal(t, "running", log.State)

		// Simulate final completion
		log.State = "completed"
		log.UpdatedAt = time.Now()
		log.Log = []byte("Processing completed successfully")

		assert.Equal(t, "completed", log.State)
		assert.True(t, log.UpdatedAt.After(log.CreatedAt))
	})

	t.Run("version upgrade tracking", func(t *testing.T) {
		versions := []string{"v1.0.0", "v1.1.0", "v1.2.0"}

		for _, version := range versions {
			log := RabbitLog{
				DocumentID: "doc-upgrade",
				State:      "completed",
				Version:    version,
				Log:        []byte("Processed with version " + version),
			}

			assert.Equal(t, version, log.Version)
			assert.Contains(t, string(log.Log), version)
		}
	})
}

// BenchmarkRabbitLog_JSONMarshal benchmarks JSON marshaling
func BenchmarkRabbitLog_JSONMarshal(b *testing.B) {
	log := RabbitLog{
		Model: gorm.Model{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		DocumentID: "doc-bench",
		State:      "completed",
		Version:    "v1.0.0",
		Log:        []byte("Benchmark test log data"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(log)
	}
}

// BenchmarkRabbitLog_Base64Encoding benchmarks base64 encoding
func BenchmarkRabbitLog_Base64Encoding(b *testing.B) {
	testData := []byte("This is test log data that will be base64 encoded multiple times for benchmarking purposes")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = base64.StdEncoding.EncodeToString(testData)
	}
}

// BenchmarkRabbitLog_Base64Decoding benchmarks base64 decoding
func BenchmarkRabbitLog_Base64Decoding(b *testing.B) {
	testData := []byte("This is test log data that will be base64 encoded")
	encoded := base64.StdEncoding.EncodeToString(testData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = base64.StdEncoding.DecodeString(encoded)
	}
}
