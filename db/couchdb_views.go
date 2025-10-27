package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	kivik "github.com/go-kivik/kivik/v4"
)

// CreateDesignDoc creates or updates a CouchDB design document containing views.
// Design documents contain MapReduce views for efficient querying and aggregation.
//
// Design Document Structure:
//
//	Design documents must have IDs starting with "_design/":
//	- Valid: "_design/graphium"
//	- Invalid: "graphium" (will be auto-prefixed)
//
// Parameters:
//   - designDoc: DesignDoc structure containing ID, language, and views
//
// Returns:
//   - error: Creation, update, or validation errors
//
// Update Behavior:
//
//	If design document exists:
//	- Retrieves current revision automatically
//	- Updates with new view definitions
//	- Preserves other design document fields
//
// Example Usage:
//
//	designDoc := DesignDoc{
//	    ID:       "_design/graphium",
//	    Language: "javascript",
//	    Views: map[string]View{
//	        "containers_by_host": {
//	            Map: `function(doc) {
//	                if (doc['@type'] === 'SoftwareApplication' && doc.hostedOn) {
//	                    emit(doc.hostedOn, {
//	                        name: doc.name,
//	                        status: doc.status
//	                    });
//	                }
//	            }`,
//	        },
//	        "container_count_by_host": {
//	            Map: `function(doc) {
//	                if (doc['@type'] === 'SoftwareApplication' && doc.hostedOn) {
//	                    emit(doc.hostedOn, 1);
//	                }
//	            }`,
//	            Reduce: "_sum",
//	        },
//	    },
//	}
//
//	err := service.CreateDesignDoc(designDoc)
//	if err != nil {
//	    log.Printf("Failed to create design doc: %v", err)
//	}
func (c *CouchDBService) CreateDesignDoc(designDoc DesignDoc) error {
	ctx := context.Background()

	// Ensure ID starts with _design/
	if !strings.HasPrefix(designDoc.ID, "_design/") {
		designDoc.ID = "_design/" + designDoc.ID
	}

	// Set default language if not specified
	if designDoc.Language == "" {
		designDoc.Language = "javascript"
	}

	// Check if design document already exists to get revision
	existingRow := c.database.Get(ctx, designDoc.ID)
	if existingRow.Err() == nil {
		// Design doc exists, get its revision
		var existing map[string]interface{}
		if err := existingRow.ScanDoc(&existing); err == nil {
			if rev, ok := existing["_rev"].(string); ok {
				designDoc.Rev = rev
			}
		}
	}

	// Convert views to the format expected by CouchDB
	viewsMap := make(map[string]interface{})
	for name, view := range designDoc.Views {
		viewDef := map[string]string{
			"map": view.Map,
		}
		if view.Reduce != "" {
			viewDef["reduce"] = view.Reduce
		}
		viewsMap[name] = viewDef
	}

	// Create the design document structure
	docData := map[string]interface{}{
		"_id":      designDoc.ID,
		"language": designDoc.Language,
		"views":    viewsMap,
	}

	if designDoc.Rev != "" {
		docData["_rev"] = designDoc.Rev
	}

	// Save the design document
	_, err := c.database.Put(ctx, designDoc.ID, docData)
	if err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "create_design_doc_failed",
				Reason:     err.Error(),
			}
		}
		return fmt.Errorf("failed to create design document: %w", err)
	}

	return nil
}

// QueryView queries a CouchDB MapReduce view with configurable options.
// Views enable efficient querying by pre-computing indexes of document data.
//
// Parameters:
//   - designName: Design document name (without "_design/" prefix)
//   - viewName: View name within the design document
//   - opts: ViewOptions for configuring the query
//
// Returns:
//   - *ViewResult: Contains rows with keys, values, and optional documents
//   - error: Query execution or parsing errors
//
// View Query Options:
//   - Key: Query for exact key match
//   - StartKey/EndKey: Query for key range
//   - IncludeDocs: Include full document content in results
//   - Limit: Maximum number of results to return
//   - Skip: Number of results to skip for pagination
//   - Descending: Reverse sort order
//   - Reduce: Execute reduce function (if view has one)
//   - Group: Group reduce results by key
//
// Example Usage:
//
//	// Query containers on a specific host
//	opts := ViewOptions{
//	    Key:         "host-123",
//	    IncludeDocs: true,
//	    Limit:       50,
//	}
//	result, err := service.QueryView("graphium", "containers_by_host", opts)
//	if err != nil {
//	    log.Printf("Query failed: %v", err)
//	    return
//	}
//
//	fmt.Printf("Found %d containers\n", len(result.Rows))
//	for _, row := range result.Rows {
//	    fmt.Printf("Container: %s -> %v\n", row.ID, row.Value)
//	}
//
//	// Count containers per host using reduce
//	opts = ViewOptions{
//	    Reduce: true,
//	    Group:  true,
//	}
//	result, _ = service.QueryView("graphium", "container_count_by_host", opts)
//	for _, row := range result.Rows {
//	    fmt.Printf("Host %v has %v containers\n", row.Key, row.Value)
//	}
func (c *CouchDBService) QueryView(designName, viewName string, opts ViewOptions) (*ViewResult, error) {
	ctx := context.Background()

	// Remove _design/ prefix if provided
	designName = strings.TrimPrefix(designName, "_design/")

	// Build query parameters
	params := make(map[string]interface{})

	if opts.Key != nil {
		params["key"] = opts.Key
	}
	if opts.StartKey != nil {
		params["startkey"] = opts.StartKey
	}
	if opts.EndKey != nil {
		params["endkey"] = opts.EndKey
	}
	if opts.IncludeDocs {
		params["include_docs"] = true
	}
	if opts.Limit > 0 {
		params["limit"] = opts.Limit
	}
	if opts.Skip > 0 {
		params["skip"] = opts.Skip
	}
	if opts.Descending {
		params["descending"] = true
	}
	if opts.Reduce {
		params["reduce"] = true
	} else if opts.Key != nil || opts.StartKey != nil || opts.EndKey != nil {
		// Explicitly disable reduce for key queries if not requested
		params["reduce"] = false
	}
	if opts.Group {
		params["group"] = true
	}
	if opts.GroupLevel > 0 {
		params["group_level"] = opts.GroupLevel
	}

	// Query the view
	rows := c.database.Query(ctx, "_design/"+designName, viewName, kivik.Params(params))
	defer rows.Close()

	result := &ViewResult{
		Rows: []ViewRow{},
	}

	// Note: TotalRows and Offset may not be available in all Kivik versions
	// They will remain 0 if not available

	// Iterate through results
	for rows.Next() {
		row := ViewRow{}

		// Get document ID (not available for reduced views)
		id, err := rows.ID()
		if err == nil {
			row.ID = id
		}

		// Get key - Key() returns (interface{}, error)
		key, err := rows.Key()
		if err == nil {
			row.Key = key
		}

		// Get value
		var value interface{}
		if err := rows.ScanValue(&value); err == nil {
			row.Value = value
		}

		// Get document if include_docs was specified
		if opts.IncludeDocs {
			var doc json.RawMessage
			if err := rows.ScanDoc(&doc); err == nil {
				row.Doc = doc
			}
		}

		result.Rows = append(result.Rows, row)
	}

	if err := rows.Err(); err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return nil, &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "query_view_failed",
				Reason:     err.Error(),
			}
		}
		return nil, fmt.Errorf("error querying view: %w", err)
	}

	return result, nil
}

// GetDesignDoc retrieves a design document by name.
// Returns the complete design document including all views.
//
// Parameters:
//   - designName: Design document name (with or without "_design/" prefix)
//
// Returns:
//   - *DesignDoc: Complete design document structure
//   - error: Not found or retrieval errors
//
// Example Usage:
//
//	doc, err := service.GetDesignDoc("graphium")
//	if err != nil {
//	    log.Printf("Design doc not found: %v", err)
//	    return
//	}
//
//	fmt.Printf("Design doc %s has %d views\n", doc.ID, len(doc.Views))
//	for viewName := range doc.Views {
//	    fmt.Printf("  - %s\n", viewName)
//	}
func (c *CouchDBService) GetDesignDoc(designName string) (*DesignDoc, error) {
	ctx := context.Background()

	// Ensure design name has _design/ prefix
	if !strings.HasPrefix(designName, "_design/") {
		designName = "_design/" + designName
	}

	row := c.database.Get(ctx, designName)
	if row.Err() != nil {
		if kivik.HTTPStatus(row.Err()) == 404 {
			return nil, &CouchDBError{
				StatusCode: 404,
				ErrorType:  "not_found",
				Reason:     fmt.Sprintf("design document %s not found", designName),
			}
		}
		return nil, &CouchDBError{
			StatusCode: kivik.HTTPStatus(row.Err()),
			ErrorType:  "get_design_doc_failed",
			Reason:     row.Err().Error(),
		}
	}

	var rawDoc map[string]interface{}
	if err := row.ScanDoc(&rawDoc); err != nil {
		return nil, fmt.Errorf("failed to scan design document: %w", err)
	}

	designDoc := &DesignDoc{
		ID:    designName,
		Views: make(map[string]View),
	}

	// Extract revision
	if rev, ok := rawDoc["_rev"].(string); ok {
		designDoc.Rev = rev
	}

	// Extract language
	if lang, ok := rawDoc["language"].(string); ok {
		designDoc.Language = lang
	} else {
		designDoc.Language = "javascript"
	}

	// Extract views
	if viewsData, ok := rawDoc["views"].(map[string]interface{}); ok {
		for viewName, viewData := range viewsData {
			if viewMap, ok := viewData.(map[string]interface{}); ok {
				view := View{
					Name: viewName,
				}
				if mapFunc, ok := viewMap["map"].(string); ok {
					view.Map = mapFunc
				}
				if reduceFunc, ok := viewMap["reduce"].(string); ok {
					view.Reduce = reduceFunc
				}
				designDoc.Views[viewName] = view
			}
		}
	}

	return designDoc, nil
}

// DeleteDesignDoc deletes a design document by name.
// This removes the design document and all its views.
//
// Parameters:
//   - designName: Design document name (with or without "_design/" prefix)
//
// Returns:
//   - error: Not found, conflict, or deletion errors
//
// Example Usage:
//
//	err := service.DeleteDesignDoc("graphium")
//	if err != nil {
//	    log.Printf("Failed to delete design doc: %v", err)
//	}
func (c *CouchDBService) DeleteDesignDoc(designName string) error {
	ctx := context.Background()

	// Ensure design name has _design/ prefix
	if !strings.HasPrefix(designName, "_design/") {
		designName = "_design/" + designName
	}

	// Get current revision
	row := c.database.Get(ctx, designName)
	if row.Err() != nil {
		if kivik.HTTPStatus(row.Err()) == 404 {
			return &CouchDBError{
				StatusCode: 404,
				ErrorType:  "not_found",
				Reason:     fmt.Sprintf("design document %s not found", designName),
			}
		}
		return &CouchDBError{
			StatusCode: kivik.HTTPStatus(row.Err()),
			ErrorType:  "get_design_doc_failed",
			Reason:     row.Err().Error(),
		}
	}

	var doc map[string]interface{}
	if err := row.ScanDoc(&doc); err != nil {
		return fmt.Errorf("failed to scan design document: %w", err)
	}

	rev, ok := doc["_rev"].(string)
	if !ok {
		return fmt.Errorf("design document has no revision")
	}

	// Delete the design document
	_, err := c.database.Delete(ctx, designName, rev)
	if err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "delete_design_doc_failed",
				Reason:     err.Error(),
			}
		}
		return fmt.Errorf("failed to delete design document: %w", err)
	}

	return nil
}

// ListDesignDocs returns a list of all design documents in the database.
// This is useful for discovering available views and design documents.
//
// Returns:
//   - []string: List of design document IDs (including "_design/" prefix)
//   - error: Query or iteration errors
//
// Example Usage:
//
//	designDocs, err := service.ListDesignDocs()
//	if err != nil {
//	    log.Printf("Failed to list design docs: %v", err)
//	    return
//	}
//
//	fmt.Println("Available design documents:")
//	for _, ddoc := range designDocs {
//	    fmt.Printf("  - %s\n", ddoc)
//	}
func (c *CouchDBService) ListDesignDocs() ([]string, error) {
	ctx := context.Background()

	// Query _all_docs with startkey and endkey to get only design docs
	params := map[string]interface{}{
		"startkey": "_design/",
		"endkey":   "_design/\ufff0", // \ufff0 is high Unicode character
	}

	rows := c.database.AllDocs(ctx, kivik.Params(params))
	defer rows.Close()

	var designDocs []string
	for rows.Next() {
		id, err := rows.ID()
		if err != nil {
			continue
		}
		designDocs = append(designDocs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error listing design documents: %w", err)
	}

	return designDocs, nil
}
