package db

import (
	"context"
	"fmt"
	"time"
	"os"
	"log"
	"strings"
	"path/filepath"
	"encoding/json"

	eve "eve.evalgo.org/common"
	kivik "github.com/go-kivik/kivik/v4"
	_ "github.com/go-kivik/kivik/v4/couchdb" // The CouchDB driver
)

type CouchDBService struct {
	client   *kivik.Client
	database *kivik.DB
	dbName   string
}

func CouchDBAnimals(url string) {
	client, err := kivik.New("couch", url)
	if err != nil {
		panic(err)
	}

	exists, _ := client.DBExists(context.Background(), "animals")
	if !exists {
		err = client.CreateDB(context.Background(), "animals")
		if err != nil {
			fmt.Println(err)
		}
	}
	db := client.DB("animals")

	doc := map[string]interface{}{
		"_id":      "cow",
		"feet":     4,
		"greeting": "moo",
	}

	rev, err := db.Put(context.TODO(), "cow", doc)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Cow inserted with revision %s\n", rev)
}

func CouchDBDocNew(url, db string, doc interface{}) (string, string) {
	client, err := kivik.New("couch", url)
	if err != nil {
		panic(err)
	}
	exists, _ := client.DBExists(context.Background(), db)
	if !exists {
		err = client.CreateDB(context.Background(), db)
		if err != nil {
			fmt.Println(err)
		}
	}
	cdb := client.DB(db)
	docId, revId, err := cdb.CreateDoc(context.TODO(), doc)
	if err != nil {
		panic(err)
	}
	return docId, revId
}

func CouchDBDocGet(url, db, docId string) *kivik.Document {
	client, err := kivik.New("couch", url)
	if err != nil {
		panic(err)
	}
	exists, _ := client.DBExists(context.Background(), db)
	if !exists {
		err = client.CreateDB(context.Background(), db)
		if err != nil {
			fmt.Println(err)
		}
	}
	cdb := client.DB(db)
	return cdb.Get(context.TODO(), docId)
}

func NewCouchDBService(config eve.FlowConfig) (*CouchDBService, error) {
	client, err := kivik.New("couch", config.CouchDBURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to CouchDB: %w", err)
	}

	ctx := context.Background()

	// Create database if it doesn't exist
	exists, err := client.DBExists(ctx, config.DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if database exists: %w", err)
	}

	if !exists {
		err = client.CreateDB(ctx, config.DatabaseName)
		if err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
	}

	db := client.DB(config.DatabaseName)

	return &CouchDBService{
		client:   client,
		database: db,
		dbName:   config.DatabaseName,
	}, nil
}

func (c *CouchDBService) SaveDocument(doc eve.FlowProcessDocument) (*eve.FlowCouchDBResponse, error) {
	ctx := context.Background()

	if doc.ID == "" {
		doc.ID = doc.ProcessID
	}
	doc.UpdatedAt = time.Now()

	// Check if document exists to get revision
	if doc.Rev == "" {
		existingDoc, err := c.GetDocument(doc.ID)
		if err == nil && existingDoc != nil {
			doc.Rev = existingDoc.Rev
			// Preserve created_at from existing document
			doc.CreatedAt = existingDoc.CreatedAt
			// Append to history
			doc.History = append(existingDoc.History, eve.FlowStateChange{
				State:     doc.State,
				Timestamp: time.Now(),
				ErrorMsg:  doc.ErrorMsg,
			})
		} else {
			// New document
			doc.CreatedAt = time.Now()
			doc.History = []eve.FlowStateChange{{
				State:     doc.State,
				Timestamp: time.Now(),
				ErrorMsg:  doc.ErrorMsg,
			}}
		}
	}

	rev, err := c.database.Put(ctx, doc.ID, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to save document: %w", err)
	}

	return &eve.FlowCouchDBResponse{
		OK:  true,
		ID:  doc.ID,
		Rev: rev,
	}, nil
}

func (c *CouchDBService) GetDocument(id string) (*eve.FlowProcessDocument, error) {
	ctx := context.Background()

	row := c.database.Get(ctx, id)
	if row.Err() != nil {
		if kivik.HTTPStatus(row.Err()) == 404 {
			return nil, fmt.Errorf("document not found")
		}
		return nil, fmt.Errorf("failed to get document: %w", row.Err())
	}

	var doc eve.FlowProcessDocument
	if err := row.ScanDoc(&doc); err != nil {
		return nil, fmt.Errorf("failed to scan document: %w", err)
	}

	return &doc, nil
}

func (c *CouchDBService) GetDocumentsByState(state eve.FlowProcessState) ([]eve.FlowProcessDocument, error) {
	ctx := context.Background()

	// Use Mango query (CouchDB's native query language)
	selector := map[string]interface{}{
		"state": string(state),
	}

	rows := c.database.Find(ctx, selector)
	defer rows.Close()

	var docs []eve.FlowProcessDocument
	for rows.Next() {
		var doc eve.FlowProcessDocument
		if err := rows.ScanDoc(&doc); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return docs, nil
}

func (c *CouchDBService) GetAllDocuments() ([]eve.FlowProcessDocument, error) {
	ctx := context.Background()

	rows := c.database.AllDocs(ctx, kivik.Param("include_docs", true))
	defer rows.Close()

	var docs []eve.FlowProcessDocument
	for rows.Next() {
		var doc eve.FlowProcessDocument
		if err := rows.ScanDoc(&doc); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return docs, nil
}

func (c *CouchDBService) DeleteDocument(id, rev string) error {
	ctx := context.Background()

	_, err := c.database.Delete(ctx, id, rev)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

func (c *CouchDBService) Close() error {
	return c.client.Close()
}

func DownloadAllDocuments(url, db, outputDir string) error {
	ctx := context.Background()
	// Connect to CouchDB
	client, err := kivik.New("couch", url)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	// Skip system databases
	fmt.Printf("Processing database: %s\n", db)

	if err := downloadDatabaseDocuments(ctx, client, db, outputDir); err != nil {
		log.Printf("Error processing database %s: %v", db, err)
	}
	return nil
}

func downloadDatabaseDocuments(ctx context.Context, client *kivik.Client, dbName, outputDir string) error {
	// Open database
	db := client.DB(dbName)

	// Create directory for this database
	dbDir := filepath.Join(outputDir, dbName)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Get all documents using _all_docs view
	rows := db.AllDocs(ctx, kivik.Param("include_docs", true))
	defer rows.Close()

	docCount := 0
	for rows.Next() {
		id, err := rows.ID()
		if err != nil {
			log.Printf("Failed to get ID: %v", err)
			continue
		}
		// Skip design documents
		if strings.HasPrefix(id, "_design/") {
			continue
		}

		var doc map[string]interface{}
		if err := rows.ScanDoc(&doc); err != nil {
			id, err := rows.ID()
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Error scanning document %s: %v", id, err)
			continue
		}

		// Save document to file
		id, err = rows.ID()
		if err != nil {
			log.Fatal(err)
		}
		filename := sanitizeFilename(id) + ".json"
		filepath := filepath.Join(dbDir, filename)

		if err := saveDocumentToFile(doc, filepath); err != nil {
			id, err := rows.ID()
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Error saving document %s: %v", id, err)
			continue
		}

		docCount++
		if docCount%100 == 0 {
			fmt.Printf("  Downloaded %d documents from %s\n", docCount, dbName)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating documents: %w", err)
	}

	fmt.Printf("  Completed %s: %d documents downloaded\n", dbName, docCount)
	return nil
}

func saveDocumentToFile(doc map[string]interface{}, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print JSON

	if err := encoder.Encode(doc); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// sanitizeFilename removes or replaces characters that are invalid in filenames
func sanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename

	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Limit length to avoid filesystem issues
	if len(result) > 200 {
		result = result[:200]
	}

	return result
}
