package db

import (
	"context"
	"encoding/json"
	"fmt"

	kivik "github.com/go-kivik/kivik/v4"
)

// SaveDocument saves a generic document to CouchDB with automatic ID and revision management.
// This function uses Go generics to provide type-safe document operations for any struct type.
//
// Type Parameter:
//   - T: Any struct type that represents a CouchDB document
//
// Document Requirements:
//   - Document should have "_id" and "_rev" fields (optional, can use struct tags)
//   - Struct fields should use `json:` tags for proper serialization
//   - Use `json:"_id"` and `json:"_rev"` tags for ID and revision fields
//
// ID and Revision Handling:
//   - If document has no ID, CouchDB generates a UUID automatically
//   - If document has ID but no revision, creates new document or updates existing
//   - If document has both ID and revision, updates the specific version
//   - Returns new revision for subsequent updates
//
// Parameters:
//   - c: CouchDBService instance
//   - doc: Document to save (any struct type)
//
// Returns:
//   - *CouchDBResponse: Contains document ID and new revision
//   - error: Save failures, conflicts, or validation errors
//
// Example Usage:
//
//	type Container struct {
//	    ID       string `json:"_id,omitempty"`
//	    Rev      string `json:"_rev,omitempty"`
//	    Type     string `json:"@type"`
//	    Name     string `json:"name"`
//	    Status   string `json:"status"`
//	    HostedOn string `json:"hostedOn"`
//	}
//
//	container := Container{
//	    ID:       "container-123",
//	    Type:     "SoftwareApplication",
//	    Name:     "nginx",
//	    Status:   "running",
//	    HostedOn: "host-456",
//	}
//
//	response, err := SaveDocument(service, container)
//	if err != nil {
//	    log.Printf("Save failed: %v", err)
//	    return
//	}
//	fmt.Printf("Saved with revision: %s\n", response.Rev)
func SaveDocument[T any](c *CouchDBService, doc T) (*CouchDBResponse, error) {
	ctx := context.Background()

	// Convert to map to extract/map ID fields
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &docMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	// Get document ID - check both @id (JSON-LD) and _id (CouchDB) fields
	var docID string
	if id, ok := docMap["@id"]; ok && id != nil && id != "" {
		// JSON-LD format uses @id
		docID = fmt.Sprintf("%v", id)
		// Map @id to _id for CouchDB
		docMap["_id"] = docID
		// Keep @id for JSON-LD compatibility
	} else if id, ok := docMap["_id"]; ok && id != nil && id != "" {
		// Standard CouchDB format
		docID = fmt.Sprintf("%v", id)
	}

	// Save the modified docMap (with _id field) instead of original doc
	var rev string
	if docID != "" {
		// Document has ID, use Put
		rev, err = c.database.Put(ctx, docID, docMap)
	} else {
		// No ID, use CreateDoc to let CouchDB generate UUID
		var revID string
		docID, revID, err = c.database.CreateDoc(ctx, docMap)
		rev = revID
	}

	if err != nil {
		// Check if it's a CouchDB error
		if kivik.HTTPStatus(err) != 0 {
			return nil, &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "save_failed",
				Reason:     err.Error(),
			}
		}
		return nil, fmt.Errorf("failed to save document: %w", err)
	}

	return &CouchDBResponse{
		OK:  true,
		ID:  docID,
		Rev: rev,
	}, nil
}

// GetDocument retrieves a generic document from CouchDB by ID.
// This function uses Go generics to return strongly-typed document results.
//
// Type Parameter:
//   - T: Expected document type (must match stored document structure)
//
// Parameters:
//   - id: Document identifier to retrieve
//
// Returns:
//   - *T: Pointer to the retrieved document of type T
//   - error: Document not found, access, or parsing errors
//
// Error Handling:
//   - Returns CouchDBError for HTTP errors (404, 401, etc.)
//   - Returns parsing error if document structure doesn't match type T
//
// Example Usage:
//
//	type Container struct {
//	    ID       string `json:"_id"`
//	    Rev      string `json:"_rev"`
//	    Name     string `json:"name"`
//	    Status   string `json:"status"`
//	}
//
//	container, err := GetDocument[Container](service, "container-123")
//	if err != nil {
//	    if couchErr, ok := err.(*CouchDBError); ok && couchErr.IsNotFound() {
//	        fmt.Println("Container not found")
//	        return
//	    }
//	    log.Printf("Error: %v", err)
//	    return
//	}
//	fmt.Printf("Container %s is %s\n", container.Name, container.Status)
func GetDocument[T any](c *CouchDBService, id string) (*T, error) {
	ctx := context.Background()

	row := c.database.Get(ctx, id)
	if row.Err() != nil {
		statusCode := kivik.HTTPStatus(row.Err())
		if statusCode == 404 {
			return nil, &CouchDBError{
				StatusCode: 404,
				ErrorType:  "not_found",
				Reason:     fmt.Sprintf("document %s not found", id),
			}
		}
		return nil, &CouchDBError{
			StatusCode: statusCode,
			ErrorType:  "get_failed",
			Reason:     row.Err().Error(),
		}
	}

	var doc T
	if err := row.ScanDoc(&doc); err != nil {
		return nil, fmt.Errorf("failed to scan document: %w", err)
	}

	return &doc, nil
}

// DeleteGenericDocument deletes a generic document from CouchDB.
// Requires both document ID and current revision for conflict detection.
//
// Parameters:
//   - c: CouchDBService instance
//   - id: Document identifier to delete
//   - rev: Current document revision (for MVCC conflict detection)
//
// Returns:
//   - error: Deletion failures or conflicts
//
// MVCC Conflict Handling:
//
//	If revision doesn't match current document:
//	- Returns CouchDBError with status 409
//	- Application should retrieve latest revision and retry
//
// Example Usage:
//
//	err := DeleteGenericDocument(service, "container-123", "2-abc123")
//	if err != nil {
//	    if couchErr, ok := err.(*CouchDBError); ok && couchErr.IsConflict() {
//	        fmt.Println("Conflict - document was modified")
//	        // Retrieve latest and retry
//	        return
//	    }
//	    log.Printf("Delete failed: %v", err)
//	}
func DeleteGenericDocument(c *CouchDBService, id, rev string) error {
	ctx := context.Background()

	_, err := c.database.Delete(ctx, id, rev)
	if err != nil {
		statusCode := kivik.HTTPStatus(err)
		if statusCode != 0 {
			return &CouchDBError{
				StatusCode: statusCode,
				ErrorType:  "delete_failed",
				Reason:     err.Error(),
			}
		}
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// GetAllDocuments retrieves all documents from the database with optional type filtering.
// This function uses Go generics to return strongly-typed document results.
//
// Type Parameter:
//   - T: Expected document type (will skip documents that don't match)
//
// Parameters:
//   - docType: Optional document type filter (e.g., "@type" field value)
//     Pass empty string to retrieve all documents
//
// Returns:
//   - []T: Slice of documents of type T
//   - error: Query execution or parsing errors
//
// Type Filtering:
//
//	If docType is provided, only returns documents with matching "@type" field.
//	Documents without "@type" field or with different type are skipped.
//
// Performance Considerations:
//   - Retrieves all documents using _all_docs view
//   - Memory usage scales with database size
//   - Consider pagination for very large databases
//
// Example Usage:
//
//	// Get all documents of any type
//	allDocs, err := GetAllDocuments[map[string]interface{}](service, "")
//
//	// Get only Container documents
//	containers, err := GetAllDocuments[Container](service, "SoftwareApplication")
//	for _, container := range containers {
//	    fmt.Printf("Container: %s (%s)\n", container.Name, container.Status)
//	}
func GetAllDocuments[T any](c *CouchDBService, docType string) ([]T, error) {
	ctx := context.Background()

	rows := c.database.AllDocs(ctx, kivik.Param("include_docs", true))
	defer rows.Close()

	var docs []T
	for rows.Next() {
		var doc T
		if err := rows.ScanDoc(&doc); err != nil {
			// Skip documents that don't match type T
			continue
		}

		// If docType filter is specified, check @type field
		if docType != "" {
			jsonData, err := json.Marshal(doc)
			if err != nil {
				continue
			}
			var docMap map[string]interface{}
			if err := json.Unmarshal(jsonData, &docMap); err != nil {
				continue
			}
			if typeVal, ok := docMap["@type"]; ok {
				if typeStr, ok := typeVal.(string); ok && typeStr == docType {
					docs = append(docs, doc)
				}
			}
		} else {
			docs = append(docs, doc)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return docs, nil
}

// GetDocumentsByType retrieves documents filtered by @type field using Mango query.
// This is more efficient than GetAllDocuments for type-filtered queries on large databases.
//
// Type Parameter:
//   - T: Expected document type
//
// Parameters:
//   - docType: Value of the "@type" field to filter by
//
// Returns:
//   - []T: Slice of matching documents
//   - error: Query execution or parsing errors
//
// Index Recommendation:
//
//	For optimal performance, create an index on the @type field:
//	index := Index{Name: "type-index", Fields: []string{"@type"}, Type: "json"}
//	service.CreateIndex(index)
//
// Example Usage:
//
//	containers, err := GetDocumentsByType[Container](service, "SoftwareApplication")
//	if err != nil {
//	    log.Printf("Query failed: %v", err)
//	    return
//	}
//
//	for _, container := range containers {
//	    fmt.Printf("Found container: %s\n", container.Name)
//	}
func GetDocumentsByType[T any](c *CouchDBService, docType string) ([]T, error) {
	ctx := context.Background()

	selector := map[string]interface{}{
		"@type": docType,
	}

	rows := c.database.Find(ctx, selector)
	defer rows.Close()

	var docs []T
	for rows.Next() {
		var doc T
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

// SaveGenericDocument saves a document using interface{} for maximum flexibility.
// This function doesn't use generics and accepts any type that can be marshaled to JSON.
//
// Parameters:
//   - doc: Document to save (any JSON-serializable type)
//
// Returns:
//   - *CouchDBResponse: Contains document ID and new revision
//   - error: Save failures or validation errors
//
// Example Usage:
//
//	doc := map[string]interface{}{
//	    "_id":     "mydoc",
//	    "name":    "example",
//	    "value":   42,
//	}
//	response, err := service.SaveGenericDocument(doc)
func (c *CouchDBService) SaveGenericDocument(doc interface{}) (*CouchDBResponse, error) {
	ctx := context.Background()

	// Convert to map to extract/map ID fields
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &docMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	// Get document ID - check both @id (JSON-LD) and _id (CouchDB) fields
	var docID string
	if id, ok := docMap["@id"]; ok && id != nil && id != "" {
		// JSON-LD format uses @id
		docID = fmt.Sprintf("%v", id)
		// Map @id to _id for CouchDB
		docMap["_id"] = docID
		// Keep @id for JSON-LD compatibility
	} else if id, ok := docMap["_id"]; ok && id != nil && id != "" {
		// Standard CouchDB format
		docID = fmt.Sprintf("%v", id)
	}

	// Save the modified docMap (with _id field) instead of original doc
	var rev string
	if docID != "" {
		// Document has ID, use Put
		rev, err = c.database.Put(ctx, docID, docMap)
	} else {
		// No ID, use CreateDoc to generate UUID
		var revID string
		docID, revID, err = c.database.CreateDoc(ctx, docMap)
		rev = revID
	}

	if err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return nil, &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "save_failed",
				Reason:     err.Error(),
			}
		}
		return nil, fmt.Errorf("failed to save document: %w", err)
	}

	return &CouchDBResponse{
		OK:  true,
		ID:  docID,
		Rev: rev,
	}, nil
}

// GetGenericDocument retrieves a document as a map for maximum flexibility.
// This function doesn't use generics and returns the document as map[string]interface{}.
//
// Parameters:
//   - id: Document identifier to retrieve
//   - result: Pointer to store the document (typically *map[string]interface{})
//
// Returns:
//   - error: Document not found or retrieval errors
//
// Example Usage:
//
//	var doc map[string]interface{}
//	err := service.GetGenericDocument("mydoc", &doc)
//	if err != nil {
//	    log.Printf("Error: %v", err)
//	    return
//	}
//	fmt.Printf("Document: %+v\n", doc)
func (c *CouchDBService) GetGenericDocument(id string, result interface{}) error {
	ctx := context.Background()

	row := c.database.Get(ctx, id)
	if row.Err() != nil {
		statusCode := kivik.HTTPStatus(row.Err())
		if statusCode == 404 {
			return &CouchDBError{
				StatusCode: 404,
				ErrorType:  "not_found",
				Reason:     fmt.Sprintf("document %s not found", id),
			}
		}
		return &CouchDBError{
			StatusCode: statusCode,
			ErrorType:  "get_failed",
			Reason:     row.Err().Error(),
		}
	}

	if err := row.ScanDoc(result); err != nil {
		return fmt.Errorf("failed to scan document: %w", err)
	}

	return nil
}

// GetAllGenericDocuments retrieves all documents as a slice of maps.
// This function provides untyped access to all database documents.
//
// Parameters:
//   - docType: Optional type filter (checks "@type" field), empty string for all
//   - result: Pointer to slice for storing results (typically *[]map[string]interface{})
//
// Returns:
//   - error: Query execution or parsing errors
//
// Example Usage:
//
//	var docs []map[string]interface{}
//	err := service.GetAllGenericDocuments("SoftwareApplication", &docs)
//	for _, doc := range docs {
//	    fmt.Printf("Document: %+v\n", doc)
//	}
func (c *CouchDBService) GetAllGenericDocuments(docType string, result interface{}) error {
	ctx := context.Background()

	rows := c.database.AllDocs(ctx, kivik.Param("include_docs", true))
	defer rows.Close()

	var docs []map[string]interface{}
	for rows.Next() {
		var doc map[string]interface{}
		if err := rows.ScanDoc(&doc); err != nil {
			continue
		}

		// Filter by @type if specified
		if docType != "" {
			if typeVal, ok := doc["@type"]; ok {
				if typeStr, ok := typeVal.(string); ok && typeStr == docType {
					docs = append(docs, doc)
				}
			}
		} else {
			docs = append(docs, doc)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// Convert to result type using JSON marshaling
	jsonData, err := json.Marshal(docs)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := json.Unmarshal(jsonData, result); err != nil {
		return fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return nil
}

// CouchDBResponse represents a successful CouchDB operation response.
// This is the generic response structure used by SaveDocument and related operations.
//
// Fields:
//   - OK: Boolean indicating success (typically true)
//   - ID: Document identifier
//   - Rev: New document revision
type CouchDBResponse struct {
	OK  bool   `json:"ok"`
	ID  string `json:"id"`
	Rev string `json:"rev"`
}
