package db

import (
	"context"
	"encoding/json"
	"fmt"

	kivik "github.com/go-kivik/kivik/v4"
)

// Find executes a Mango query and returns results as json.RawMessage.
// Mango queries provide MongoDB-style declarative filtering without MapReduce views.
//
// Parameters:
//   - query: MangoQuery structure with selector, fields, sort, and pagination
//
// Returns:
//   - []json.RawMessage: Array of matching documents as raw JSON
//   - error: Query execution or parsing errors
//
// Mango Query Language:
//
//	Supports MongoDB-style operators:
//	- $eq, $ne: Equality operators
//	- $gt, $gte, $lt, $lte: Comparison operators
//	- $and, $or, $not: Logical operators
//	- $in, $nin: Array membership
//	- $regex: Regular expression matching
//	- $exists: Field existence check
//
// Index Usage:
//
//	For optimal performance, create indexes on queried fields:
//	index := Index{Name: "status-index", Fields: []string{"status"}, Type: "json"}
//	service.CreateIndex(index)
//
// Example Usage:
//
//	// Find running containers in us-east
//	query := MangoQuery{
//	    Selector: map[string]interface{}{
//	        "$and": []interface{}{
//	            map[string]interface{}{"status": "running"},
//	            map[string]interface{}{"location": map[string]interface{}{
//	                "$regex": "^us-east",
//	            }},
//	        },
//	    },
//	    Fields: []string{"_id", "name", "status"},
//	    Sort: []map[string]string{
//	        {"name": "asc"},
//	    },
//	    Limit: 100,
//	}
//
//	results, err := service.Find(query)
//	if err != nil {
//	    log.Printf("Query failed: %v", err)
//	    return
//	}
//
//	for _, result := range results {
//	    var doc map[string]interface{}
//	    json.Unmarshal(result, &doc)
//	    fmt.Printf("Found: %+v\n", doc)
//	}
func (c *CouchDBService) Find(query MangoQuery) ([]json.RawMessage, error) {
	ctx := context.Background()

	rows := c.database.Find(ctx, query.Selector, kivik.Params(query.toParams()))
	defer rows.Close()

	var results []json.RawMessage
	for rows.Next() {
		var doc json.RawMessage
		if err := rows.ScanDoc(&doc); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		results = append(results, doc)
	}

	if err := rows.Err(); err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return nil, &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "find_failed",
				Reason:     err.Error(),
			}
		}
		return nil, fmt.Errorf("error executing find query: %w", err)
	}

	return results, nil
}

// FindTyped executes a Mango query with typed results using generics.
// This provides compile-time type safety for query results.
//
// Type Parameter:
//   - T: Expected document type
//
// Parameters:
//   - query: MangoQuery structure with filtering and options
//
// Returns:
//   - []T: Slice of documents matching type T
//   - error: Query execution or parsing errors
//
// Example Usage:
//
//	type Container struct {
//	    ID       string `json:"_id"`
//	    Name     string `json:"name"`
//	    Status   string `json:"status"`
//	    HostedOn string `json:"hostedOn"`
//	}
//
//	query := MangoQuery{
//	    Selector: map[string]interface{}{
//	        "status": "running",
//	        "@type": "SoftwareApplication",
//	    },
//	    Limit: 50,
//	}
//
//	containers, err := FindTyped[Container](service, query)
//	if err != nil {
//	    log.Printf("Query failed: %v", err)
//	    return
//	}
//
//	for _, container := range containers {
//	    fmt.Printf("Container %s is %s\n", container.Name, container.Status)
//	}
func FindTyped[T any](c *CouchDBService, query MangoQuery) ([]T, error) {
	ctx := context.Background()

	rows := c.database.Find(ctx, query.Selector, kivik.Params(query.toParams()))
	defer rows.Close()

	var results []T
	for rows.Next() {
		var doc T
		if err := rows.ScanDoc(&doc); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		results = append(results, doc)
	}

	if err := rows.Err(); err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return nil, &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "find_failed",
				Reason:     err.Error(),
			}
		}
		return nil, fmt.Errorf("error executing find query: %w", err)
	}

	return results, nil
}

// toParams converts MangoQuery to Kivik parameters.
// This internal helper method converts query options to CouchDB parameters.
func (q *MangoQuery) toParams() map[string]interface{} {
	params := make(map[string]interface{})

	if len(q.Fields) > 0 {
		params["fields"] = q.Fields
	}
	if len(q.Sort) > 0 {
		params["sort"] = q.Sort
	}
	if q.Limit > 0 {
		params["limit"] = q.Limit
	}
	if q.Skip > 0 {
		params["skip"] = q.Skip
	}
	if q.UseIndex != "" {
		params["use_index"] = q.UseIndex
	}

	return params
}

// QueryBuilder provides a fluent API for constructing complex Mango queries.
// This builder pattern simplifies query construction with method chaining.
//
// Example Usage:
//
//	query := NewQueryBuilder().
//	    Where("status", "eq", "running").
//	    And().
//	    Where("location", "regex", "^us-east").
//	    Select("_id", "name", "status").
//	    Limit(50).
//	    Build()
//
//	results, _ := service.Find(query)
type QueryBuilder struct {
	conditions     []map[string]interface{}
	logicalOp      string // "and" or "or"
	fields         []string
	sortFields     []map[string]string
	limitValue     int
	skipValue      int
	useIndexValue  string
	currentCondSet []map[string]interface{} // Current set of conditions
}

// NewQueryBuilder creates a new QueryBuilder instance.
// Returns a builder ready for method chaining.
//
// Example Usage:
//
//	qb := NewQueryBuilder()
//	query := qb.Where("status", "eq", "running").Build()
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		conditions:     []map[string]interface{}{},
		currentCondSet: []map[string]interface{}{},
		logicalOp:      "and",
	}
}

// Where adds a condition to the query.
// Supports various operators for flexible filtering.
//
// Parameters:
//   - field: Document field name to filter on
//   - operator: Comparison operator ("eq", "ne", "gt", "gte", "lt", "lte", "regex", "in", "nin", "exists")
//   - value: Value to compare against
//
// Returns:
//   - *QueryBuilder: Builder instance for method chaining
//
// Supported Operators:
//   - "eq": Equal to (default)
//   - "ne": Not equal to
//   - "gt": Greater than
//   - "gte": Greater than or equal
//   - "lt": Less than
//   - "lte": Less than or equal
//   - "regex": Regular expression match
//   - "in": In array
//   - "nin": Not in array
//   - "exists": Field exists (value should be bool)
//
// Example Usage:
//
//	qb.Where("status", "eq", "running")
//	qb.Where("count", "gt", 10)
//	qb.Where("location", "regex", "^us-")
//	qb.Where("tags", "in", []string{"production", "critical"})
func (qb *QueryBuilder) Where(field string, operator string, value interface{}) *QueryBuilder {
	var condition map[string]interface{}

	switch operator {
	case "eq", "=", "==":
		// Simple equality
		condition = map[string]interface{}{
			field: value,
		}
	case "ne", "!=":
		condition = map[string]interface{}{
			field: map[string]interface{}{"$ne": value},
		}
	case "gt", ">":
		condition = map[string]interface{}{
			field: map[string]interface{}{"$gt": value},
		}
	case "gte", ">=":
		condition = map[string]interface{}{
			field: map[string]interface{}{"$gte": value},
		}
	case "lt", "<":
		condition = map[string]interface{}{
			field: map[string]interface{}{"$lt": value},
		}
	case "lte", "<=":
		condition = map[string]interface{}{
			field: map[string]interface{}{"$lte": value},
		}
	case "regex", "~=":
		condition = map[string]interface{}{
			field: map[string]interface{}{"$regex": value},
		}
	case "in":
		condition = map[string]interface{}{
			field: map[string]interface{}{"$in": value},
		}
	case "nin":
		condition = map[string]interface{}{
			field: map[string]interface{}{"$nin": value},
		}
	case "exists":
		condition = map[string]interface{}{
			field: map[string]interface{}{"$exists": value},
		}
	default:
		// Default to equality
		condition = map[string]interface{}{
			field: value,
		}
	}

	qb.currentCondSet = append(qb.currentCondSet, condition)
	return qb
}

// And specifies that subsequent conditions should be AND'd together.
// This is the default logical operator.
//
// Returns:
//   - *QueryBuilder: Builder instance for method chaining
//
// Example Usage:
//
//	qb.Where("status", "eq", "running").
//	    And().
//	    Where("location", "eq", "us-east")
func (qb *QueryBuilder) And() *QueryBuilder {
	if len(qb.currentCondSet) > 0 {
		qb.conditions = append(qb.conditions, qb.currentCondSet...)
		qb.currentCondSet = []map[string]interface{}{}
	}
	qb.logicalOp = "and"
	return qb
}

// Or specifies that subsequent conditions should be OR'd together.
// Changes the logical operator for combining conditions.
//
// Returns:
//   - *QueryBuilder: Builder instance for method chaining
//
// Example Usage:
//
//	qb.Where("status", "eq", "running").
//	    Or().
//	    Where("status", "eq", "starting")
func (qb *QueryBuilder) Or() *QueryBuilder {
	if len(qb.currentCondSet) > 0 {
		qb.conditions = append(qb.conditions, qb.currentCondSet...)
		qb.currentCondSet = []map[string]interface{}{}
	}
	qb.logicalOp = "or"
	return qb
}

// Select specifies which fields to return in query results.
// This provides projection to reduce bandwidth and processing.
//
// Parameters:
//   - fields: Field names to include in results
//
// Returns:
//   - *QueryBuilder: Builder instance for method chaining
//
// Example Usage:
//
//	qb.Select("_id", "name", "status", "hostedOn")
func (qb *QueryBuilder) Select(fields ...string) *QueryBuilder {
	qb.fields = fields
	return qb
}

// Sort specifies sort order for query results.
// Multiple sort fields can be specified for multi-level sorting.
//
// Parameters:
//   - field: Field name to sort by
//   - direction: Sort direction ("asc" or "desc")
//
// Returns:
//   - *QueryBuilder: Builder instance for method chaining
//
// Example Usage:
//
//	qb.Sort("status", "asc").Sort("name", "asc")
func (qb *QueryBuilder) Sort(field, direction string) *QueryBuilder {
	sortField := map[string]string{
		field: direction,
	}
	qb.sortFields = append(qb.sortFields, sortField)
	return qb
}

// Limit sets the maximum number of results to return.
// Used for pagination and result size control.
//
// Parameters:
//   - n: Maximum number of results
//
// Returns:
//   - *QueryBuilder: Builder instance for method chaining
//
// Example Usage:
//
//	qb.Limit(100)
func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	qb.limitValue = n
	return qb
}

// Skip sets the number of results to skip (for pagination).
// Used with Limit to implement pagination.
//
// Parameters:
//   - n: Number of results to skip
//
// Returns:
//   - *QueryBuilder: Builder instance for method chaining
//
// Example Usage:
//
//	// Get second page (results 51-100)
//	qb.Skip(50).Limit(50)
func (qb *QueryBuilder) Skip(n int) *QueryBuilder {
	qb.skipValue = n
	return qb
}

// UseIndex hints which index to use for the query.
// Improves performance by explicitly selecting an index.
//
// Parameters:
//   - indexName: Name of the index to use
//
// Returns:
//   - *QueryBuilder: Builder instance for method chaining
//
// Example Usage:
//
//	qb.UseIndex("status-location-index")
func (qb *QueryBuilder) UseIndex(indexName string) *QueryBuilder {
	qb.useIndexValue = indexName
	return qb
}

// Build constructs the final MangoQuery from the builder.
// Returns a MangoQuery ready for execution.
//
// Returns:
//   - MangoQuery: Complete query structure
//
// Example Usage:
//
//	query := NewQueryBuilder().
//	    Where("status", "eq", "running").
//	    And().
//	    Where("location", "regex", "^us-east").
//	    Select("_id", "name", "status").
//	    Sort("name", "asc").
//	    Limit(50).
//	    Build()
//
//	results, _ := service.Find(query)
func (qb *QueryBuilder) Build() MangoQuery {
	// Add any remaining conditions
	if len(qb.currentCondSet) > 0 {
		qb.conditions = append(qb.conditions, qb.currentCondSet...)
	}

	selector := make(map[string]interface{})

	if len(qb.conditions) == 0 {
		// No conditions, match all documents
		selector = map[string]interface{}{}
	} else if len(qb.conditions) == 1 {
		// Single condition
		selector = qb.conditions[0]
	} else {
		// Multiple conditions - use logical operator
		if qb.logicalOp == "or" {
			selector = map[string]interface{}{
				"$or": qb.conditions,
			}
		} else {
			// Default to AND
			selector = map[string]interface{}{
				"$and": qb.conditions,
			}
		}
	}

	query := MangoQuery{
		Selector: selector,
		Fields:   qb.fields,
		Sort:     qb.sortFields,
		Limit:    qb.limitValue,
		Skip:     qb.skipValue,
		UseIndex: qb.useIndexValue,
	}

	return query
}

// Count returns the count of documents matching the selector.
// This is a convenience method that executes a query and returns the count.
//
// Parameters:
//   - selector: Mango selector (same format as MangoQuery.Selector)
//
// Returns:
//   - int: Number of matching documents
//   - error: Query execution errors
//
// Example Usage:
//
//	selector := map[string]interface{}{
//	    "status": "running",
//	    "@type": "SoftwareApplication",
//	}
//	count, err := service.Count(selector)
//	fmt.Printf("Found %d running containers\n", count)
func (c *CouchDBService) Count(selector map[string]interface{}) (int, error) {
	ctx := context.Background()

	rows := c.database.Find(ctx, selector)
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error counting documents: %w", err)
	}

	return count, nil
}
