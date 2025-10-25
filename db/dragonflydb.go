package db

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

// DragonflyDBSaveKeyValue stores a key-value pair in DragonflyDB.
// DragonflyDB is compatible with Redis protocol, so this function uses the go-redis client.
// The value is stored with no expiration (TTL = 0).
//
// The function requires the following environment variables:
//   - DRAGONFLYDB_HOST: DragonflyDB server address (e.g., "localhost:6379")
//   - DRAGONFLYDB_PASSWORD: Authentication password (use empty string if no password)
//
// Parameters:
//   - key: The key under which to store the value
//   - value: The value to store as bytes
//
// Returns:
//   - error: Any error encountered during storage operation
//
// Example:
//
//	data := []byte("user profile data")
//	err := DragonflyDBSaveKeyValue("user:1234", data)
//	if err != nil {
//	    log.Fatal(err)
//	}
func DragonflyDBSaveKeyValue(key string, value []byte) error {
	ctx := context.Background()

	// Create Redis client (works with DragonflyDB)
	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("DRAGONFLYDB_HOST"),     // DragonflyDB address
		Password: os.Getenv("DRAGONFLYDB_PASSWORD"), // no password set
		DB:       0,                                 // default DB
	})

	// Close connection
	defer client.Close()

	// Test connection
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		return err
	}
	fmt.Println("Connected:", pong)

	// Example 1: Simple SET
	err = client.Set(ctx, key, value, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

// DragonflyDBGetKey retrieves a value from DragonflyDB by key.
// DragonflyDB is compatible with Redis protocol, so this function uses the go-redis client.
// The function prints the key and connection status for debugging purposes.
//
// The function requires the following environment variables:
//   - DRAGONFLYDB_HOST: DragonflyDB server address (e.g., "localhost:6379")
//   - DRAGONFLYDB_PASSWORD: Authentication password (use empty string if no password)
//
// Parameters:
//   - key: The key to retrieve
//
// Returns:
//   - []byte: The value stored at the key
//   - error: Any error encountered (including redis.Nil if key doesn't exist)
//
// Example:
//
//	data, err := DragonflyDBGetKey("user:1234")
//	if err == redis.Nil {
//	    fmt.Println("Key does not exist")
//	} else if err != nil {
//	    log.Fatal(err)
//	} else {
//	    fmt.Printf("Value: %s\n", data)
//	}
func DragonflyDBGetKey(key string) ([]byte, error) {
	fmt.Println("key:", key)
	ctx := context.Background()

	// Create Redis client (works with DragonflyDB)
	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("DRAGONFLYDB_HOST"),     // DragonflyDB address
		Password: os.Getenv("DRAGONFLYDB_PASSWORD"), // no password set
		DB:       0,                                 // default DB
	})

	// Close connection
	defer client.Close()

	// Test connection
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	fmt.Println("Connected:", pong)

	// Example 3: Get value
	return client.Get(ctx, key).Bytes()
}
