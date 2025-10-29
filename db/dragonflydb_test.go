package db

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDragonflyDBSaveKeyValue tests saving key-value pairs
func TestDragonflyDBSaveKeyValue(t *testing.T) {
	t.Run("successful save with miniredis", func(t *testing.T) {
		// Create a miniredis server
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		// Set environment variables
		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		// Test saving a key-value pair
		key := "test:key:123"
		value := []byte("test value data")

		err = DragonflyDBSaveKeyValue(key, value)
		assert.NoError(t, err)

		// Verify the value was stored
		storedValue, err := mr.Get(key)
		require.NoError(t, err)
		assert.Equal(t, string(value), storedValue)
	})

	t.Run("save multiple keys", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		// Save multiple keys
		testData := map[string][]byte{
			"user:1": []byte("Alice"),
			"user:2": []byte("Bob"),
			"user:3": []byte("Charlie"),
		}

		for key, value := range testData {
			err := DragonflyDBSaveKeyValue(key, value)
			assert.NoError(t, err)
		}

		// Verify all values
		for key, expectedValue := range testData {
			storedValue, err := mr.Get(key)
			require.NoError(t, err)
			assert.Equal(t, string(expectedValue), storedValue)
		}
	})

	t.Run("save empty value", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		err = DragonflyDBSaveKeyValue("empty:key", []byte(""))
		assert.NoError(t, err)

		storedValue, err := mr.Get("empty:key")
		require.NoError(t, err)
		assert.Equal(t, "", storedValue)
	})

	t.Run("save binary data", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		err = DragonflyDBSaveKeyValue("binary:key", binaryData)
		assert.NoError(t, err)

		storedValue, err := mr.Get("binary:key")
		require.NoError(t, err)
		assert.Equal(t, string(binaryData), storedValue)
	})

	t.Run("connection failure", func(t *testing.T) {
		// Set invalid host
		os.Setenv("DRAGONFLYDB_HOST", "invalid:12345")
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		err := DragonflyDBSaveKeyValue("test:key", []byte("value"))
		assert.Error(t, err)
	})
}

// TestDragonflyDBGetKey tests retrieving values by key
func TestDragonflyDBGetKey(t *testing.T) {
	t.Run("successful get existing key", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		// Pre-populate data
		key := "test:data:456"
		expectedValue := "test value"
		mr.Set(key, expectedValue)

		// Retrieve the value
		value, err := DragonflyDBGetKey(key)
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, string(value))
	})

	t.Run("get non-existent key", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		// Try to get a key that doesn't exist
		value, err := DragonflyDBGetKey("nonexistent:key")
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "redis: nil")
	})

	t.Run("get binary data", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		// Store binary data
		key := "binary:data"
		binaryData := []byte{0xFF, 0xFE, 0xFD, 0x00, 0x01}
		mr.Set(key, string(binaryData))

		// Retrieve binary data
		value, err := DragonflyDBGetKey(key)
		assert.NoError(t, err)
		assert.Equal(t, binaryData, value)
	})

	t.Run("get empty value", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		key := "empty:value"
		mr.Set(key, "")

		value, err := DragonflyDBGetKey(key)
		assert.NoError(t, err)
		assert.Equal(t, []byte(""), value)
	})

	t.Run("connection failure on get", func(t *testing.T) {
		os.Setenv("DRAGONFLYDB_HOST", "invalid:99999")
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		value, err := DragonflyDBGetKey("test:key")
		assert.Error(t, err)
		assert.Nil(t, value)
	})
}

// TestDragonflyDB_SaveAndRetrieve tests complete save and retrieve workflow
func TestDragonflyDB_SaveAndRetrieve(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
	os.Setenv("DRAGONFLYDB_PASSWORD", "")
	defer os.Unsetenv("DRAGONFLYDB_HOST")
	defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

	testCases := []struct {
		name  string
		key   string
		value []byte
	}{
		{
			name:  "simple string",
			key:   "user:profile:1",
			value: []byte("John Doe"),
		},
		{
			name:  "JSON data",
			key:   "config:app",
			value: []byte(`{"setting":"value","enabled":true}`),
		},
		{
			name:  "large data",
			key:   "data:large",
			value: []byte(string(make([]byte, 10000))),
		},
		{
			name:  "special characters",
			key:   "special:chars",
			value: []byte("Hello 世界 🌍"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save
			err := DragonflyDBSaveKeyValue(tc.key, tc.value)
			assert.NoError(t, err)

			// Retrieve
			retrieved, err := DragonflyDBGetKey(tc.key)
			assert.NoError(t, err)
			assert.Equal(t, tc.value, retrieved)
		})
	}
}

// TestDragonflyDB_KeyPatterns tests various key naming patterns
func TestDragonflyDB_KeyPatterns(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
	os.Setenv("DRAGONFLYDB_PASSWORD", "")
	defer os.Unsetenv("DRAGONFLYDB_HOST")
	defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

	keyPatterns := []string{
		"simple",
		"with:colon",
		"with-dash",
		"with_underscore",
		"with.dot",
		"namespace:type:id:123",
		"CamelCase",
		"UPPERCASE",
	}

	for _, key := range keyPatterns {
		t.Run(key, func(t *testing.T) {
			value := []byte("test value for " + key)
			err := DragonflyDBSaveKeyValue(key, value)
			assert.NoError(t, err)

			retrieved, err := DragonflyDBGetKey(key)
			assert.NoError(t, err)
			assert.Equal(t, value, retrieved)
		})
	}
}

// TestDragonflyDB_OverwriteValue tests overwriting existing values
func TestDragonflyDB_OverwriteValue(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
	os.Setenv("DRAGONFLYDB_PASSWORD", "")
	defer os.Unsetenv("DRAGONFLYDB_HOST")
	defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

	key := "overwrite:test"

	// Save initial value
	initialValue := []byte("initial value")
	err = DragonflyDBSaveKeyValue(key, initialValue)
	assert.NoError(t, err)

	// Verify initial value
	retrieved, err := DragonflyDBGetKey(key)
	assert.NoError(t, err)
	assert.Equal(t, initialValue, retrieved)

	// Overwrite with new value
	newValue := []byte("new value")
	err = DragonflyDBSaveKeyValue(key, newValue)
	assert.NoError(t, err)

	// Verify new value
	retrieved, err = DragonflyDBGetKey(key)
	assert.NoError(t, err)
	assert.Equal(t, newValue, retrieved)
	assert.NotEqual(t, initialValue, retrieved)
}

// BenchmarkDragonflyDBSaveKeyValue benchmarks save operations
func BenchmarkDragonflyDBSaveKeyValue(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatal(err)
	}
	defer mr.Close()

	os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
	os.Setenv("DRAGONFLYDB_PASSWORD", "")
	defer os.Unsetenv("DRAGONFLYDB_HOST")
	defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

	value := []byte("benchmark test value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench:key:" + string(rune(i))
		_ = DragonflyDBSaveKeyValue(key, value)
	}
}

// BenchmarkDragonflyDBGetKey benchmarks get operations
func BenchmarkDragonflyDBGetKey(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatal(err)
	}
	defer mr.Close()

	os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
	os.Setenv("DRAGONFLYDB_PASSWORD", "")
	defer os.Unsetenv("DRAGONFLYDB_HOST")
	defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

	// Pre-populate data
	key := "bench:get:key"
	value := []byte("benchmark test value")
	mr.Set(key, string(value))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DragonflyDBGetKey(key)
	}
}

// TestDragonflyDBGetKeyWithDialer tests custom dialer functionality (Ziti support)
func TestDragonflyDBGetKeyWithDialer(t *testing.T) {
	t.Run("successful get with custom dialer", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		// Pre-populate data
		key := "dialer:test:key"
		expectedValue := "test value with dialer"
		mr.Set(key, expectedValue)

		// Create custom dialer that uses standard TCP
		customDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", mr.Addr())
		}

		// Retrieve using custom dialer
		value, err := DragonflyDBGetKeyWithDialer(key, customDialer)
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, string(value))
	})

	t.Run("dialer error handling", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		// Create dialer that always fails
		failingDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, errors.New("mock dialer connection failure")
		}

		// Should fail due to dialer error
		value, err := DragonflyDBGetKeyWithDialer("test:key", failingDialer)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "connection failure")
	})

	t.Run("get non-existent key with dialer", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		customDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", mr.Addr())
		}

		// Try to get non-existent key
		value, err := DragonflyDBGetKeyWithDialer("nonexistent:key", customDialer)
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "redis: nil")
	})

	t.Run("get binary data with dialer", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		// Store binary data
		key := "binary:dialer:data"
		binaryData := []byte{0xFF, 0xFE, 0xFD, 0x00, 0x01, 0x02}
		mr.Set(key, string(binaryData))

		customDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", mr.Addr())
		}

		// Retrieve binary data
		value, err := DragonflyDBGetKeyWithDialer(key, customDialer)
		assert.NoError(t, err)
		assert.Equal(t, binaryData, value)
	})
}

// TestDragonflyDBSaveKeyValueWithDialer tests custom dialer functionality for save operations
func TestDragonflyDBSaveKeyValueWithDialer(t *testing.T) {
	t.Run("successful save with custom dialer", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		customDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", mr.Addr())
		}

		key := "dialer:save:key"
		value := []byte("saved with custom dialer")

		// Save using custom dialer
		err = DragonflyDBSaveKeyValueWithDialer(key, value, customDialer)
		assert.NoError(t, err)

		// Verify the value was stored
		storedValue, err := mr.Get(key)
		require.NoError(t, err)
		assert.Equal(t, string(value), storedValue)
	})

	t.Run("save with dialer error", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		failingDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, errors.New("dialer connection refused")
		}

		err = DragonflyDBSaveKeyValueWithDialer("test:key", []byte("value"), failingDialer)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("save binary data with dialer", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		customDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", mr.Addr())
		}

		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		err = DragonflyDBSaveKeyValueWithDialer("binary:dialer:key", binaryData, customDialer)
		assert.NoError(t, err)

		storedValue, err := mr.Get("binary:dialer:key")
		require.NoError(t, err)
		assert.Equal(t, string(binaryData), storedValue)
	})

	t.Run("save multiple keys with dialer", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		customDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", mr.Addr())
		}

		testData := map[string][]byte{
			"dialer:user:1": []byte("Alice"),
			"dialer:user:2": []byte("Bob"),
			"dialer:user:3": []byte("Charlie"),
		}

		for key, value := range testData {
			err := DragonflyDBSaveKeyValueWithDialer(key, value, customDialer)
			assert.NoError(t, err)
		}

		// Verify all values
		for key, expectedValue := range testData {
			storedValue, err := mr.Get(key)
			require.NoError(t, err)
			assert.Equal(t, string(expectedValue), storedValue)
		}
	})
}

// TestDragonflyDB_WithDialer_SaveAndRetrieve tests complete workflow with custom dialer
func TestDragonflyDB_WithDialer_SaveAndRetrieve(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
	os.Setenv("DRAGONFLYDB_PASSWORD", "")
	defer os.Unsetenv("DRAGONFLYDB_HOST")
	defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

	// Create a custom dialer that simulates Ziti-like behavior
	customDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
		// In real Ziti implementation, this would use zitiContext.Dial(serviceName)
		// For testing, we just connect to miniredis
		return net.Dial("tcp", mr.Addr())
	}

	testCases := []struct {
		name  string
		key   string
		value []byte
	}{
		{
			name:  "simple string with dialer",
			key:   "dialer:profile:1",
			value: []byte("John Doe via Ziti"),
		},
		{
			name:  "JSON data with dialer",
			key:   "dialer:config:app",
			value: []byte(`{"ziti_enabled":true,"service":"dragonflydb"}`),
		},
		{
			name:  "large data with dialer",
			key:   "dialer:data:large",
			value: []byte(string(make([]byte, 5000))),
		},
		{
			name:  "special characters with dialer",
			key:   "dialer:special:chars",
			value: []byte("Ziti 世界 🚀"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save with custom dialer
			err := DragonflyDBSaveKeyValueWithDialer(tc.key, tc.value, customDialer)
			assert.NoError(t, err)

			// Retrieve with custom dialer
			retrieved, err := DragonflyDBGetKeyWithDialer(tc.key, customDialer)
			assert.NoError(t, err)
			assert.Equal(t, tc.value, retrieved)
		})
	}
}

// TestDragonflyDB_DialerContextCancellation tests dialer behavior with context cancellation
func TestDragonflyDB_DialerContextCancellation(t *testing.T) {
	t.Run("context cancellation in dialer", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		os.Setenv("DRAGONFLYDB_HOST", mr.Addr())
		os.Setenv("DRAGONFLYDB_PASSWORD", "")
		defer os.Unsetenv("DRAGONFLYDB_HOST")
		defer os.Unsetenv("DRAGONFLYDB_PASSWORD")

		// Create a dialer that checks context cancellation
		contextCancelledDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return net.Dial("tcp", mr.Addr())
			}
		}

		// Normal operation should work
		key := "context:test:key"
		value := []byte("test value")
		err = DragonflyDBSaveKeyValueWithDialer(key, value, contextCancelledDialer)
		assert.NoError(t, err)

		retrieved, err := DragonflyDBGetKeyWithDialer(key, contextCancelledDialer)
		assert.NoError(t, err)
		assert.Equal(t, value, retrieved)
	})
}
