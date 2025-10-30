package db

import (
	"context"
	"encoding/json"
	"fmt"

	kivik "github.com/go-kivik/kivik/v4"
)

// BulkSaveDocuments saves multiple documents in a single database operation.
// Bulk operations significantly improve performance when saving many documents
// by reducing network round trips and database overhead.
//
// Parameters:
//   - docs: Slice of documents to save (any JSON-serializable type)
//
// Returns:
//   - []BulkResult: Result for each document (success or error)
//   - error: Request execution errors (not individual document errors)
//
// Document Requirements:
//   - Documents can have _id field (explicit ID) or let CouchDB generate UUID
//   - Documents with _rev field are updated, without _rev are created
//   - Each document is processed independently with individual success/failure
//
// Result Handling:
//
//	Each BulkResult indicates success or failure for one document:
//	- OK=true: Document saved successfully, Rev contains new revision
//	- OK=false: Save failed, Error and Reason explain why
//
// Common Errors:
//   - "conflict": Document revision mismatch (concurrent modification)
//   - "forbidden": Insufficient permissions for document
//   - "invalid": Document validation failed
//
// Performance:
//   - Single HTTP request for all documents
//   - Transactional consistency within the bulk operation
//   - Suitable for batch imports and synchronization
//
// Example Usage:
//
//	type Container struct {
//	    ID     string `json:"_id,omitempty"`
//	    Name   string `json:"name"`
//	    Status string `json:"status"`
//	}
//
//	containers := []interface{}{
//	    Container{ID: "c1", Name: "nginx", Status: "running"},
//	    Container{ID: "c2", Name: "redis", Status: "running"},
//	    Container{ID: "c3", Name: "postgres", Status: "stopped"},
//	}
//
//	results, err := service.BulkSaveDocuments(containers)
//	if err != nil {
//	    log.Printf("Bulk save failed: %v", err)
//	    return
//	}
//
//	successCount := 0
//	for _, result := range results {
//	    if result.OK {
//	        fmt.Printf("Saved %s with rev %s\n", result.ID, result.Rev)
//	        successCount++
//	    } else {
//	        fmt.Printf("Failed %s: %s - %s\n", result.ID, result.Error, result.Reason)
//	    }
//	}
//	fmt.Printf("Successfully saved %d/%d documents\n", successCount, len(results))
func (c *CouchDBService) BulkSaveDocuments(docs []interface{}) ([]BulkResult, error) {
	ctx := context.Background()

	if len(docs) == 0 {
		return []BulkResult{}, nil
	}

	// Process each document to map @id to _id (JSON-LD compatibility)
	// This ensures documents with @id use that as their CouchDB _id,
	// preventing duplicate documents with different _id but same @id
	processedDocs := make([]interface{}, len(docs))
	for i, doc := range docs {
		// Convert to map to handle @id → _id mapping
		jsonData, err := json.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal document %d: %w", i, err)
		}

		var docMap map[string]interface{}
		if err := json.Unmarshal(jsonData, &docMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal document %d: %w", i, err)
		}

		// Map @id to _id for CouchDB (same logic as SaveGenericDocument)
		if id, ok := docMap["@id"]; ok && id != nil && id != "" {
			// JSON-LD format uses @id
			docID := fmt.Sprintf("%v", id)
			// Set _id from @id if not already set
			if _, hasID := docMap["_id"]; !hasID || docMap["_id"] == "" {
				docMap["_id"] = docID
			}
			// Keep @id for JSON-LD compatibility
		}

		processedDocs[i] = docMap
	}

	// Use BulkDocs to save all documents
	results, err := c.database.BulkDocs(ctx, processedDocs)
	if err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return nil, &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "bulk_save_failed",
				Reason:     err.Error(),
			}
		}
		return nil, fmt.Errorf("failed to bulk save documents: %w", err)
	}

	// Process results - BulkDocs returns []BulkResult directly
	var bulkResults []BulkResult
	for _, kivikResult := range results {
		result := BulkResult{
			ID: kivikResult.ID,
		}

		if kivikResult.Error != nil {
			// Error occurred
			result.OK = false
			result.Error = "operation_failed"
			result.Reason = kivikResult.Error.Error()
		} else {
			// Success
			result.OK = true
			result.Rev = kivikResult.Rev
		}

		bulkResults = append(bulkResults, result)
	}

	return bulkResults, nil
}

// BulkDeleteDocuments deletes multiple documents in a single database operation.
// This is more efficient than individual delete operations for batch deletions.
//
// Parameters:
//   - docs: Slice of BulkDeleteDoc with ID, Rev, and Deleted=true
//
// Returns:
//   - []BulkResult: Result for each deletion (success or error)
//   - error: Request execution errors
//
// Deletion Requirements:
//   - Each document must have _id and _rev fields
//   - _deleted field must be set to true
//   - Revision must match current document (MVCC conflict detection)
//
// Example Usage:
//
//	// Get documents to delete with their current revisions
//	container1, _ := GetDocument[Container](service, "c1")
//	container2, _ := GetDocument[Container](service, "c2")
//
//	deleteOps := []BulkDeleteDoc{
//	    {ID: container1.ID, Rev: container1.Rev, Deleted: true},
//	    {ID: container2.ID, Rev: container2.Rev, Deleted: true},
//	}
//
//	results, err := service.BulkDeleteDocuments(deleteOps)
//	if err != nil {
//	    log.Printf("Bulk delete failed: %v", err)
//	    return
//	}
//
//	for _, result := range results {
//	    if result.OK {
//	        fmt.Printf("Deleted %s\n", result.ID)
//	    } else {
//	        fmt.Printf("Failed to delete %s: %s\n", result.ID, result.Reason)
//	    }
//	}
func (c *CouchDBService) BulkDeleteDocuments(docs []BulkDeleteDoc) ([]BulkResult, error) {
	ctx := context.Background()

	if len(docs) == 0 {
		return []BulkResult{}, nil
	}

	// Convert to interface{} slice for BulkDocs
	interfaceDocs := make([]interface{}, len(docs))
	for i, doc := range docs {
		interfaceDocs[i] = doc
	}

	// Use BulkDocs to delete all documents
	results, err := c.database.BulkDocs(ctx, interfaceDocs)
	if err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return nil, &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "bulk_delete_failed",
				Reason:     err.Error(),
			}
		}
		return nil, fmt.Errorf("failed to bulk delete documents: %w", err)
	}

	// Process results - BulkDocs returns []BulkResult directly
	var bulkResults []BulkResult
	for _, kivikResult := range results {
		result := BulkResult{
			ID: kivikResult.ID,
		}

		if kivikResult.Error != nil {
			// Error occurred
			result.OK = false
			result.Error = "operation_failed"
			result.Reason = kivikResult.Error.Error()
		} else {
			// Success
			result.OK = true
			result.Rev = kivikResult.Rev
		}

		bulkResults = append(bulkResults, result)
	}

	return bulkResults, nil
}

// BulkGet retrieves multiple documents in a single database operation.
// This is more efficient than individual get operations for batch retrieval.
//
// Type Parameter:
//   - T: Expected document type
//
// Parameters:
//   - ids: Slice of document IDs to retrieve
//
// Returns:
//   - map[string]*T: Map of document ID to document pointer
//   - map[string]error: Map of document ID to error (for failures)
//   - error: Request execution errors
//
// Result Handling:
//   - Successfully retrieved documents appear in the first map
//   - Failed retrievals (not found, etc.) appear in the error map
//   - Request-level errors returned as error parameter
//
// Example Usage:
//
//	ids := []string{"c1", "c2", "c3", "missing"}
//	docs, errors, err := BulkGet[Container](service, ids)
//	if err != nil {
//	    log.Printf("Bulk get failed: %v", err)
//	    return
//	}
//
//	fmt.Printf("Retrieved %d documents\n", len(docs))
//	for id, doc := range docs {
//	    fmt.Printf("  %s: %s (%s)\n", id, doc.Name, doc.Status)
//	}
//
//	if len(errors) > 0 {
//	    fmt.Println("Errors:")
//	    for id, err := range errors {
//	        fmt.Printf("  %s: %v\n", id, err)
//	    }
//	}
func BulkGet[T any](c *CouchDBService, ids []string) (map[string]*T, map[string]error, error) {
	ctx := context.Background()

	if len(ids) == 0 {
		return map[string]*T{}, map[string]error{}, nil
	}

	docs := make(map[string]*T)
	errs := make(map[string]error)

	// CouchDB's _bulk_get endpoint requires a specific format
	// We'll use AllDocs with keys parameter as an alternative
	rows := c.database.AllDocs(ctx, kivik.Params(map[string]interface{}{
		"include_docs": true,
		"keys":         ids,
	}))
	defer rows.Close()

	for rows.Next() {
		id, err := rows.ID()
		if err != nil {
			continue
		}

		// Check if document exists
		if rows.Err() != nil {
			errs[id] = rows.Err()
			continue
		}

		var doc T
		if err := rows.ScanDoc(&doc); err != nil {
			errs[id] = fmt.Errorf("failed to scan document: %w", err)
			continue
		}

		docs[id] = &doc
	}

	if err := rows.Err(); err != nil {
		return docs, errs, fmt.Errorf("error in bulk get: %w", err)
	}

	return docs, errs, nil
}

// BulkUpdate performs a bulk update operation on documents matching a selector.
// This applies an update function to all documents matching the criteria.
//
// Type Parameter:
//   - T: Document type to update
//
// Parameters:
//   - selector: Mango selector for finding documents to update
//   - updateFunc: Function to apply to each document
//
// Returns:
//   - int: Number of documents successfully updated
//   - error: Query or update errors
//
// Update Process:
//  1. Query documents matching selector
//  2. Apply updateFunc to each document
//  3. Bulk save all modified documents
//  4. Return count of successful updates
//
// Example Usage:
//
//	// Stop all running containers in us-east
//	selector := map[string]interface{}{
//	    "status":   "running",
//	    "location": map[string]interface{}{"$regex": "^us-east"},
//	}
//
//	count, err := BulkUpdate[Container](service, selector, func(doc *Container) error {
//	    doc.Status = "stopped"
//	    return nil
//	})
//
//	if err != nil {
//	    log.Printf("Bulk update failed: %v", err)
//	    return
//	}
//	fmt.Printf("Stopped %d containers\n", count)
func BulkUpdate[T any](c *CouchDBService, selector map[string]interface{}, updateFunc func(*T) error) (int, error) {
	// Find documents matching selector
	query := MangoQuery{
		Selector: selector,
	}

	docs, err := FindTyped[T](c, query)
	if err != nil {
		return 0, fmt.Errorf("failed to find documents: %w", err)
	}

	if len(docs) == 0 {
		return 0, nil
	}

	// Apply update function to each document
	var updatedDocs []interface{}
	for idx := range docs {
		if err := updateFunc(&docs[idx]); err != nil {
			return 0, fmt.Errorf("update function failed: %w", err)
		}
		updatedDocs = append(updatedDocs, docs[idx])
	}

	// Bulk save updated documents
	results, err := c.BulkSaveDocuments(updatedDocs)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk save: %w", err)
	}

	// Count successful updates
	successCount := 0
	for _, result := range results {
		if result.OK {
			successCount++
		}
	}

	return successCount, nil
}

// BulkUpsert performs bulk upsert (insert or update) operations.
// Documents are inserted if they don't exist, updated if they do.
//
// Parameters:
//   - docs: Slice of documents to upsert
//   - getIDFunc: Function to extract document ID from each document
//
// Returns:
//   - []BulkResult: Result for each operation
//   - error: Request execution errors
//
// Upsert Process:
//  1. For each document, extract ID using getIDFunc
//  2. Check if document exists and get current revision
//  3. Update document with current revision (if exists)
//  4. Perform bulk save operation
//
// Example Usage:
//
//	containers := []Container{
//	    {ID: "c1", Name: "nginx", Status: "running"},
//	    {ID: "c2", Name: "redis", Status: "running"},
//	}
//
//	results, err := BulkUpsert(service, containers, func(c Container) string {
//	    return c.ID
//	})
//
//	if err != nil {
//	    log.Printf("Bulk upsert failed: %v", err)
//	    return
//	}
//
//	for _, result := range results {
//	    if result.OK {
//	        fmt.Printf("Upserted %s\n", result.ID)
//	    }
//	}
func BulkUpsert[T any](c *CouchDBService, docs []T, getIDFunc func(T) string) ([]BulkResult, error) {
	if len(docs) == 0 {
		return []BulkResult{}, nil
	}

	// Extract IDs and fetch existing documents
	ids := make([]string, len(docs))
	for idx, doc := range docs {
		ids[idx] = getIDFunc(doc)
	}

	// Get existing documents to retrieve revisions
	existingDocs, _, _ := BulkGet[map[string]interface{}](c, ids)

	// Update documents with current revisions where they exist
	var docsToSave []interface{}
	for _, doc := range docs {
		// Convert doc to map to add revision
		jsonData, err := json.Marshal(doc)
		if err != nil {
			continue
		}

		var docMap map[string]interface{}
		if err := json.Unmarshal(jsonData, &docMap); err != nil {
			continue
		}

		// Add revision if document exists
		id := getIDFunc(doc)
		if existing, ok := existingDocs[id]; ok {
			if rev, ok := (*existing)["_rev"]; ok {
				docMap["_rev"] = rev
			}
		}

		docsToSave = append(docsToSave, docMap)
	}

	// Perform bulk save
	return c.BulkSaveDocuments(docsToSave)
}
