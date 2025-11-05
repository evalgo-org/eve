// Package storage provides common storage and database utilities for EVE services.
// This package includes standard database connection management, configuration patterns,
// and common operations used across the EVE ecosystem.
package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	kivik "github.com/go-kivik/kivik/v4"
	_ "github.com/go-kivik/kivik/v4/couchdb" // CouchDB driver
)

// DatabaseConfig contains common database configuration options
type DatabaseConfig struct {
	URL             string        // Database server URL
	Database        string        // Database name
	Username        string        // Authentication username
	Password        string        // Authentication password
	Timeout         time.Duration // Operation timeout
	CreateIfMissing bool          // Auto-create database if it doesn't exist
}

// DefaultDatabaseConfig returns a database config with sensible defaults
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		URL:             "http://localhost:5984",
		Database:        "",
		Username:        "",
		Password:        "",
		Timeout:         30 * time.Second,
		CreateIfMissing: true,
	}
}

// CouchDBClient wraps a Kivik client with common utilities
type CouchDBClient struct {
	client   *kivik.Client
	database *kivik.DB
	dbName   string
	config   DatabaseConfig
}

// NewCouchDBClient creates a new CouchDB client with the provided configuration
func NewCouchDBClient(config DatabaseConfig) (*CouchDBClient, error) {
	// Build connection URL with authentication if provided
	connectionURL, err := buildConnectionURL(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build connection URL: %w", err)
	}

	// Create Kivik client
	client, err := kivik.New("couch", connectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create CouchDB client: %w", err)
	}

	ctx := context.Background()
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	// Check if database exists
	exists, err := client.DBExists(ctx, config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to check database existence: %w", err)
	}

	// Create database if needed
	if !exists {
		if config.CreateIfMissing {
			if err := client.CreateDB(ctx, config.Database); err != nil {
				return nil, fmt.Errorf("failed to create database %s: %w", config.Database, err)
			}
		} else {
			return nil, fmt.Errorf("database %s does not exist", config.Database)
		}
	}

	// Get database handle
	db := client.DB(config.Database)

	return &CouchDBClient{
		client:   client,
		database: db,
		dbName:   config.Database,
		config:   config,
	}, nil
}

// buildConnectionURL constructs the connection URL with authentication
func buildConnectionURL(config DatabaseConfig) (string, error) {
	if config.URL == "" {
		return "", fmt.Errorf("database URL cannot be empty")
	}

	// If no credentials, return URL as-is
	if config.Username == "" && config.Password == "" {
		return config.URL, nil
	}

	// Parse URL to inject credentials
	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		return "", fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Set credentials
	if config.Username != "" {
		parsedURL.User = url.UserPassword(config.Username, config.Password)
	}

	return parsedURL.String(), nil
}

// GetDocument retrieves a document by ID
func (c *CouchDBClient) GetDocument(ctx context.Context, id string, dest interface{}) error {
	row := c.database.Get(ctx, id)
	if row.Err() != nil {
		if kivik.HTTPStatus(row.Err()) == 404 {
			return fmt.Errorf("document not found: %s", id)
		}
		return fmt.Errorf("failed to get document: %w", row.Err())
	}

	if err := row.ScanDoc(dest); err != nil {
		return fmt.Errorf("failed to scan document: %w", err)
	}

	return nil
}

// PutDocument creates or updates a document
func (c *CouchDBClient) PutDocument(ctx context.Context, id string, doc interface{}) (string, error) {
	rev, err := c.database.Put(ctx, id, doc)
	if err != nil {
		return "", fmt.Errorf("failed to put document: %w", err)
	}
	return rev, nil
}

// DeleteDocument deletes a document by ID and revision
func (c *CouchDBClient) DeleteDocument(ctx context.Context, id, rev string) error {
	_, err := c.database.Delete(ctx, id, rev)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

// CreateDocument creates a new document with auto-generated ID
func (c *CouchDBClient) CreateDocument(ctx context.Context, doc interface{}) (string, string, error) {
	docID, rev, err := c.database.CreateDoc(ctx, doc)
	if err != nil {
		return "", "", fmt.Errorf("failed to create document: %w", err)
	}
	return docID, rev, nil
}

// AllDocs retrieves all documents from the database
func (c *CouchDBClient) AllDocs(ctx context.Context) ([]interface{}, error) {
	rows := c.database.AllDocs(ctx, kivik.Param("include_docs", true))
	defer rows.Close()

	var docs []interface{}
	for rows.Next() {
		var doc interface{}
		if err := rows.ScanDoc(&doc); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	return docs, nil
}

// Find executes a Mango query to find documents
func (c *CouchDBClient) Find(ctx context.Context, selector map[string]interface{}, dest interface{}) error {
	rows := c.database.Find(ctx, selector)
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no documents found")
	}

	if err := rows.ScanDoc(dest); err != nil {
		return fmt.Errorf("failed to scan document: %w", err)
	}

	return nil
}

// FindAll executes a Mango query and returns all matching documents
func (c *CouchDBClient) FindAll(ctx context.Context, selector map[string]interface{}) ([]interface{}, error) {
	rows := c.database.Find(ctx, selector)
	defer rows.Close()

	var docs []interface{}
	for rows.Next() {
		var doc interface{}
		if err := rows.ScanDoc(&doc); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	return docs, nil
}

// DatabaseExists checks if a database exists
func (c *CouchDBClient) DatabaseExists(ctx context.Context, dbName string) (bool, error) {
	exists, err := c.client.DBExists(ctx, dbName)
	if err != nil {
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}
	return exists, nil
}

// CreateDatabase creates a new database
func (c *CouchDBClient) CreateDatabase(ctx context.Context, dbName string) error {
	if err := c.client.CreateDB(ctx, dbName); err != nil {
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}
	return nil
}

// DeleteDatabase deletes a database
func (c *CouchDBClient) DeleteDatabase(ctx context.Context, dbName string) error {
	if err := c.client.DestroyDB(ctx, dbName); err != nil {
		return fmt.Errorf("failed to delete database %s: %w", dbName, err)
	}
	return nil
}

// Close closes the database connection
func (c *CouchDBClient) Close() error {
	return c.client.Close()
}

// GetDatabase returns the database handle for advanced operations
func (c *CouchDBClient) GetDatabase() *kivik.DB {
	return c.database
}

// GetClient returns the Kivik client for advanced operations
func (c *CouchDBClient) GetClient() *kivik.Client {
	return c.client
}

// DatabaseName returns the current database name
func (c *CouchDBClient) DatabaseName() string {
	return c.dbName
}

// Stats returns database statistics
func (c *CouchDBClient) Stats(ctx context.Context) (*DatabaseStats, error) {
	stats, err := c.database.Stats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", err)
	}

	return &DatabaseStats{
		DBName:      c.dbName,
		DocCount:    stats.DocCount,
		DocDelCount: stats.DeletedCount,
		UpdateSeq:   stats.UpdateSeq,
		DiskSize:    stats.DiskSize,
		DataSize:    stats.ActiveSize,
	}, nil
}

// DatabaseStats contains database statistics
type DatabaseStats struct {
	DBName      string // Database name
	DocCount    int64  // Number of active documents
	DocDelCount int64  // Number of deleted documents
	UpdateSeq   string // Update sequence for change tracking
	DiskSize    int64  // Disk space used
	DataSize    int64  // Active data size
}

// Compact triggers database compaction
func (c *CouchDBClient) Compact(ctx context.Context) error {
	if err := c.database.Compact(ctx); err != nil {
		return fmt.Errorf("failed to compact database: %w", err)
	}
	return nil
}

// DocumentStore defines a generic interface for document storage operations
// This interface allows for easy mocking and testing
type DocumentStore interface {
	GetDocument(ctx context.Context, id string, dest interface{}) error
	PutDocument(ctx context.Context, id string, doc interface{}) (string, error)
	DeleteDocument(ctx context.Context, id, rev string) error
	CreateDocument(ctx context.Context, doc interface{}) (string, string, error)
	FindAll(ctx context.Context, selector map[string]interface{}) ([]interface{}, error)
	Close() error
}

// Ensure CouchDBClient implements DocumentStore
var _ DocumentStore = (*CouchDBClient)(nil)
