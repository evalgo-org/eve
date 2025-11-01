package bolt

import (
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

// DB wraps bbolt database with helper methods
type DB struct {
	*bolt.DB
}

// Open opens or creates a bbolt database
func Open(path string) (*DB, error) {
	boltDB, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &DB{boltDB}, nil
}

// CreateBucket creates a bucket if it doesn't exist
func (db *DB) CreateBucket(name string) error {
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(name))
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", name, err)
		}
		return nil
	})
}

// PutJSON stores a value as JSON in the specified bucket
func (db *DB) PutJSON(bucket, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket not found: %s", bucket)
		}
		return b.Put([]byte(key), data)
	})
}

// GetJSON retrieves a value as JSON from the specified bucket
func (db *DB) GetJSON(bucket, key string, value interface{}) error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket not found: %s", bucket)
		}

		data := b.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("key not found: %s", key)
		}

		return json.Unmarshal(data, value)
	})
}

// Delete removes a key from the specified bucket
func (db *DB) Delete(bucket, key string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket not found: %s", bucket)
		}
		return b.Delete([]byte(key))
	})
}

// List returns all keys in the specified bucket
func (db *DB) List(bucket string) ([]string, error) {
	var keys []string

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket not found: %s", bucket)
		}

		return b.ForEach(func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})

	return keys, err
}

// ForEach iterates over all key-value pairs in a bucket
func (db *DB) ForEach(bucket string, fn func(key, value []byte) error) error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket not found: %s", bucket)
		}

		return b.ForEach(fn)
	})
}

// ForEachJSON iterates over all values as JSON in a bucket
func (db *DB) ForEachJSON(bucket string, fn func(key string, value interface{}) error, valueType func() interface{}) error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket not found: %s", bucket)
		}

		return b.ForEach(func(k, v []byte) error {
			value := valueType()
			if err := json.Unmarshal(v, value); err != nil {
				return fmt.Errorf("failed to unmarshal %s: %w", k, err)
			}
			return fn(string(k), value)
		})
	})
}
