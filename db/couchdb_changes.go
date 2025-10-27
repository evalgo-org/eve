package db

import (
	"context"
	"encoding/json"
	"fmt"

	kivik "github.com/go-kivik/kivik/v4"
)

// ListenChanges starts listening to database changes and calls handler for each change.
// This enables real-time notifications of document modifications for WebSocket support
// and live synchronization.
//
// Parameters:
//   - opts: ChangesFeedOptions configuring the feed type and filtering
//   - handler: Function called for each change event
//
// Returns:
//   - error: Connection, parsing, or handler errors
//
// Feed Types:
//   - "normal": Return all changes since sequence and close
//   - "longpoll": Wait for changes, return when available, close
//   - "continuous": Keep connection open, stream changes indefinitely
//
// Change Handling:
//
//	Handler function receives each change event:
//	- Called synchronously for each change
//	- Handler errors stop the feed
//	- Long-running handlers block subsequent changes
//
// Heartbeat:
//
//	For continuous feeds, heartbeat keeps connection alive:
//	- Sent as newline characters during idle periods
//	- Prevents timeout on long-running connections
//	- Recommended: 60000ms (60 seconds)
//
// Example Usage:
//
//	// Monitor all container changes
//	opts := ChangesFeedOptions{
//	    Since:       "now",
//	    Feed:        "continuous",
//	    IncludeDocs: true,
//	    Heartbeat:   60000,
//	    Selector: map[string]interface{}{
//	        "@type": "SoftwareApplication",
//	    },
//	}
//
//	err := service.ListenChanges(opts, func(change Change) {
//	    if change.Deleted {
//	        fmt.Printf("Container %s was deleted\n", change.ID)
//	        return
//	    }
//
//	    fmt.Printf("Container %s changed to rev %s\n",
//	        change.ID, change.Changes[0].Rev)
//
//	    if change.Doc != nil {
//	        var container map[string]interface{}
//	        json.Unmarshal(change.Doc, &container)
//	        fmt.Printf("  Status: %s\n", container["status"])
//	    }
//	})
//
//	if err != nil {
//	    log.Printf("Changes feed error: %v", err)
//	}
func (c *CouchDBService) ListenChanges(opts ChangesFeedOptions, handler func(Change)) error {
	ctx := context.Background()

	// Build changes parameters
	params := make(map[string]interface{})

	if opts.Since != "" {
		params["since"] = opts.Since
	}
	if opts.Feed != "" {
		params["feed"] = opts.Feed
	} else {
		params["feed"] = "continuous" // Default to continuous
	}
	if opts.IncludeDocs {
		params["include_docs"] = true
	}
	if opts.Heartbeat > 0 {
		params["heartbeat"] = opts.Heartbeat
	}
	if opts.Timeout > 0 {
		params["timeout"] = opts.Timeout
	}
	if opts.Limit > 0 {
		params["limit"] = opts.Limit
	}
	if opts.Descending {
		params["descending"] = true
	}
	if opts.Filter != "" {
		params["filter"] = opts.Filter
	}
	if opts.Selector != nil {
		// Selector requires JSON encoding
		selectorJSON, err := json.Marshal(opts.Selector)
		if err == nil {
			params["filter"] = "_selector"
			params["selector"] = string(selectorJSON)
		}
	}

	// Get changes feed
	changes := c.database.Changes(ctx, kivik.Params(params))
	defer changes.Close()

	// Process changes
	for changes.Next() {
		change := Change{}

		// Get sequence
		change.Seq = changes.Seq()

		// Get ID
		change.ID = changes.ID()

		// Get deleted flag
		change.Deleted = changes.Deleted()

		// Get changes (revisions) - scan the document for change details
		var changesData []ChangeRev
		var rawDoc map[string]interface{}
		if err := changes.ScanDoc(&rawDoc); err == nil {
			if changesArray, ok := rawDoc["changes"].([]interface{}); ok {
				for _, chg := range changesArray {
					if chgMap, ok := chg.(map[string]interface{}); ok {
						if rev, ok := chgMap["rev"].(string); ok {
							changesData = append(changesData, ChangeRev{Rev: rev})
						}
					}
				}
			}
			change.Changes = changesData

			// Get document if include_docs was specified
			if opts.IncludeDocs && !change.Deleted {
				if docData, ok := rawDoc["doc"]; ok {
					docJSON, _ := json.Marshal(docData)
					change.Doc = docJSON
				}
			}
		}

		// Call handler
		handler(change)
	}

	if err := changes.Err(); err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "changes_feed_failed",
				Reason:     err.Error(),
			}
		}
		return fmt.Errorf("changes feed error: %w", err)
	}

	return nil
}

// GetChanges retrieves changes without continuous listening.
// This is useful for one-time synchronization or batch processing.
//
// Parameters:
//   - opts: ChangesFeedOptions (should use feed="normal" or omit)
//
// Returns:
//   - []Change: Slice of change events
//   - string: Last sequence ID for resuming
//   - error: Query or parsing errors
//
// Usage Pattern:
//
//	For polling-based synchronization:
//	1. Call GetChanges with Since=lastSeq
//	2. Process returned changes
//	3. Store returned sequence for next call
//	4. Repeat periodically
//
// Example Usage:
//
//	lastSeq := "0"
//	for {
//	    opts := ChangesFeedOptions{
//	        Since:       lastSeq,
//	        Feed:        "normal",
//	        IncludeDocs: true,
//	        Limit:       100,
//	    }
//
//	    changes, newSeq, err := service.GetChanges(opts)
//	    if err != nil {
//	        log.Printf("Error getting changes: %v", err)
//	        time.Sleep(10 * time.Second)
//	        continue
//	    }
//
//	    for _, change := range changes {
//	        fmt.Printf("Processing change: %s\n", change.ID)
//	        // Process change
//	    }
//
//	    lastSeq = newSeq
//	    time.Sleep(5 * time.Second)
//	}
func (c *CouchDBService) GetChanges(opts ChangesFeedOptions) ([]Change, string, error) {
	ctx := context.Background()

	// Build parameters
	params := make(map[string]interface{})

	if opts.Since != "" {
		params["since"] = opts.Since
	}
	// Force normal feed for GetChanges
	params["feed"] = "normal"

	if opts.IncludeDocs {
		params["include_docs"] = true
	}
	if opts.Limit > 0 {
		params["limit"] = opts.Limit
	}
	if opts.Descending {
		params["descending"] = true
	}
	if opts.Filter != "" {
		params["filter"] = opts.Filter
	}
	if opts.Selector != nil {
		selectorJSON, err := json.Marshal(opts.Selector)
		if err == nil {
			params["filter"] = "_selector"
			params["selector"] = string(selectorJSON)
		}
	}

	// Get changes
	rows := c.database.Changes(ctx, kivik.Params(params))
	defer rows.Close()

	var changes []Change
	lastSeq := ""

	for rows.Next() {
		change := Change{}

		// Get sequence
		change.Seq = rows.Seq()
		lastSeq = change.Seq

		// Get ID
		change.ID = rows.ID()

		// Get deleted flag
		change.Deleted = rows.Deleted()

		// Get changes (revisions) - scan the document for change details
		var changesData []ChangeRev
		var rawDoc map[string]interface{}
		if err := rows.ScanDoc(&rawDoc); err == nil {
			if changesArray, ok := rawDoc["changes"].([]interface{}); ok {
				for _, chg := range changesArray {
					if chgMap, ok := chg.(map[string]interface{}); ok {
						if rev, ok := chgMap["rev"].(string); ok {
							changesData = append(changesData, ChangeRev{Rev: rev})
						}
					}
				}
			}
			change.Changes = changesData

			// Get document if include_docs was specified
			if opts.IncludeDocs && !change.Deleted {
				if docData, ok := rawDoc["doc"]; ok {
					docJSON, _ := json.Marshal(docData)
					change.Doc = docJSON
				}
			}
		}

		changes = append(changes, change)
	}

	if err := rows.Err(); err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return nil, lastSeq, &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "get_changes_failed",
				Reason:     err.Error(),
			}
		}
		return nil, lastSeq, fmt.Errorf("error getting changes: %w", err)
	}

	return changes, lastSeq, nil
}

// WatchChanges provides a channel-based interface for change notifications.
// This enables Go-idiomatic concurrent processing of changes.
//
// Parameters:
//   - opts: ChangesFeedOptions configuration
//
// Returns:
//   - <-chan Change: Read-only channel for receiving changes
//   - <-chan error: Read-only channel for receiving errors
//   - func(): Stop function to close the changes feed
//
// Usage Pattern:
//
//	The function returns immediately with channels:
//	- Changes are sent to the first channel
//	- Errors are sent to the second channel
//	- Call stop() to gracefully shutdown
//
// Example Usage:
//
//	opts := ChangesFeedOptions{
//	    Since:       "now",
//	    Feed:        "continuous",
//	    IncludeDocs: true,
//	}
//
//	changeChan, errChan, stop := service.WatchChanges(opts)
//	defer stop()
//
//	for {
//	    select {
//	    case change := <-changeChan:
//	        fmt.Printf("Change: %s\n", change.ID)
//	        // Process change
//	    case err := <-errChan:
//	        log.Printf("Error: %v\n", err)
//	        return
//	    case <-time.After(60 * time.Second):
//	        fmt.Println("No changes for 60 seconds")
//	    }
//	}
func (c *CouchDBService) WatchChanges(opts ChangesFeedOptions) (<-chan Change, <-chan error, func()) {
	changeChan := make(chan Change, 100)
	errChan := make(chan error, 1)
	stopChan := make(chan struct{})

	go func() {
		defer close(changeChan)
		defer close(errChan)

		err := c.ListenChanges(opts, func(change Change) {
			select {
			case changeChan <- change:
			case <-stopChan:
				return
			}
		})

		if err != nil {
			select {
			case errChan <- err:
			case <-stopChan:
			}
		}
	}()

	stopFunc := func() {
		close(stopChan)
	}

	return changeChan, errChan, stopFunc
}

// GetLastSequence retrieves the current database sequence ID.
// Useful for starting change feeds from the current point.
//
// Returns:
//   - string: Current sequence ID
//   - error: Database query errors
//
// Example Usage:
//
//	seq, err := service.GetLastSequence()
//	if err != nil {
//	    log.Printf("Failed to get sequence: %v", err)
//	    return
//	}
//
//	opts := ChangesFeedOptions{
//	    Since: seq,
//	    Feed:  "continuous",
//	}
//	service.ListenChanges(opts, handler)
func (c *CouchDBService) GetLastSequence() (string, error) {
	info, err := c.GetDatabaseInfo()
	if err != nil {
		return "", err
	}
	return info.UpdateSeq, nil
}
