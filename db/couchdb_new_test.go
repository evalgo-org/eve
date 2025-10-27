package db

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCouchDBError tests the CouchDBError type and its methods
func TestCouchDBError(t *testing.T) {
	t.Run("Error method", func(t *testing.T) {
		err := &CouchDBError{
			StatusCode: 404,
			ErrorType:  "not_found",
			Reason:     "missing",
		}

		expected := "CouchDB error (status 404): not_found - missing"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("IsNotFound", func(t *testing.T) {
		err := &CouchDBError{StatusCode: 404, ErrorType: "not_found"}
		assert.True(t, err.IsNotFound())

		err = &CouchDBError{StatusCode: 409}
		assert.False(t, err.IsNotFound())
	})

	t.Run("IsConflict", func(t *testing.T) {
		err := &CouchDBError{StatusCode: 409, ErrorType: "conflict"}
		assert.True(t, err.IsConflict())

		err = &CouchDBError{StatusCode: 404}
		assert.False(t, err.IsConflict())
	})

	t.Run("IsUnauthorized", func(t *testing.T) {
		err := &CouchDBError{StatusCode: 401, ErrorType: "unauthorized"}
		assert.True(t, err.IsUnauthorized())

		err = &CouchDBError{StatusCode: 403, ErrorType: "forbidden"}
		assert.True(t, err.IsUnauthorized())

		err = &CouchDBError{StatusCode: 404}
		assert.False(t, err.IsUnauthorized())
	})
}

// TestMangoQuery_toParams tests the MangoQuery parameter conversion
func TestMangoQuery_toParams(t *testing.T) {
	t.Run("all parameters set", func(t *testing.T) {
		query := MangoQuery{
			Selector: map[string]interface{}{"status": "active"},
			Fields:   []string{"_id", "name", "status"},
			Sort:     []map[string]string{{"name": "asc"}},
			Limit:    50,
			Skip:     10,
			UseIndex: "status-index",
		}

		params := query.toParams()

		assert.Equal(t, []string{"_id", "name", "status"}, params["fields"])
		assert.Equal(t, []map[string]string{{"name": "asc"}}, params["sort"])
		assert.Equal(t, 50, params["limit"])
		assert.Equal(t, 10, params["skip"])
		assert.Equal(t, "status-index", params["use_index"])
	})

	t.Run("minimal parameters", func(t *testing.T) {
		query := MangoQuery{
			Selector: map[string]interface{}{"@type": "Test"},
		}

		params := query.toParams()

		assert.Empty(t, params)
	})

	t.Run("only fields", func(t *testing.T) {
		query := MangoQuery{
			Fields: []string{"name", "value"},
		}

		params := query.toParams()

		assert.Contains(t, params, "fields")
		assert.Equal(t, []string{"name", "value"}, params["fields"])
		assert.NotContains(t, params, "limit")
		assert.NotContains(t, params, "skip")
	})
}

// TestQueryBuilder tests the QueryBuilder fluent API
func TestQueryBuilder(t *testing.T) {
	t.Run("simple equality", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("status", "eq", "running").
			Build()

		assert.Equal(t, "running", query.Selector["status"])
	})

	t.Run("multiple conditions with AND", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("status", "eq", "running").
			And().
			Where("location", "regex", "^us-east").
			Build()

		assert.Contains(t, query.Selector, "$and")
		conditions := query.Selector["$and"].([]map[string]interface{})
		assert.Len(t, conditions, 2)
	})

	t.Run("multiple conditions with OR", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("status", "eq", "running").
			Or().
			Where("status", "eq", "pending").
			Build()

		assert.Contains(t, query.Selector, "$or")
		conditions := query.Selector["$or"].([]map[string]interface{})
		assert.Len(t, conditions, 2)
	})

	t.Run("comparison operators", func(t *testing.T) {
		tests := []struct {
			operator string
			expected string
		}{
			{"gt", "$gt"},
			{"gte", "$gte"},
			{"lt", "$lt"},
			{"lte", "$lte"},
			{"ne", "$ne"},
		}

		for _, tt := range tests {
			t.Run(tt.operator, func(t *testing.T) {
				query := NewQueryBuilder().
					Where("count", tt.operator, 10).
					Build()

				countCond := query.Selector["count"].(map[string]interface{})
				assert.Contains(t, countCond, tt.expected)
				assert.Equal(t, 10, countCond[tt.expected])
			})
		}
	})

	t.Run("regex operator", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("location", "regex", "^us-").
			Build()

		locationCond := query.Selector["location"].(map[string]interface{})
		assert.Contains(t, locationCond, "$regex")
		assert.Equal(t, "^us-", locationCond["$regex"])
	})

	t.Run("in operator", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("status", "in", []string{"running", "pending"}).
			Build()

		statusCond := query.Selector["status"].(map[string]interface{})
		assert.Contains(t, statusCond, "$in")
		assert.Equal(t, []string{"running", "pending"}, statusCond["$in"])
	})

	t.Run("exists operator", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("optionalField", "exists", true).
			Build()

		fieldCond := query.Selector["optionalField"].(map[string]interface{})
		assert.Contains(t, fieldCond, "$exists")
		assert.Equal(t, true, fieldCond["$exists"])
	})

	t.Run("select fields", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("status", "eq", "active").
			Select("_id", "name", "status").
			Build()

		assert.Equal(t, []string{"_id", "name", "status"}, query.Fields)
	})

	t.Run("sort ascending", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("status", "eq", "active").
			Sort("name", "asc").
			Build()

		assert.Len(t, query.Sort, 1)
		assert.Equal(t, "asc", query.Sort[0]["name"])
	})

	t.Run("sort descending", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("status", "eq", "active").
			Sort("createdAt", "desc").
			Build()

		assert.Len(t, query.Sort, 1)
		assert.Equal(t, "desc", query.Sort[0]["createdAt"])
	})

	t.Run("limit and skip", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("status", "eq", "active").
			Limit(50).
			Skip(100).
			Build()

		assert.Equal(t, 50, query.Limit)
		assert.Equal(t, 100, query.Skip)
	})

	t.Run("use index", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("status", "eq", "active").
			UseIndex("status-index").
			Build()

		assert.Equal(t, "status-index", query.UseIndex)
	})

	t.Run("complex query", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("@type", "eq", "SoftwareApplication").
			And().
			Where("status", "eq", "running").
			And().
			Where("cpu", "gte", 4).
			Select("_id", "name", "status", "cpu").
			Sort("name", "asc").
			Limit(100).
			UseIndex("status-cpu-index").
			Build()

		assert.Contains(t, query.Selector, "$and")
		assert.Equal(t, []string{"_id", "name", "status", "cpu"}, query.Fields)
		assert.Equal(t, 100, query.Limit)
		assert.Equal(t, "status-cpu-index", query.UseIndex)
	})

	t.Run("empty query", func(t *testing.T) {
		query := NewQueryBuilder().Build()

		assert.Empty(t, query.Selector)
		assert.Empty(t, query.Fields)
		assert.Equal(t, 0, query.Limit)
	})

	t.Run("single condition no logical op", func(t *testing.T) {
		query := NewQueryBuilder().
			Where("status", "eq", "active").
			Build()

		// Single condition should not wrap in $and
		assert.NotContains(t, query.Selector, "$and")
		assert.Equal(t, "active", query.Selector["status"])
	})
}

// TestIndex tests Index structure
func TestIndex(t *testing.T) {
	t.Run("create index", func(t *testing.T) {
		index := Index{
			Name:   "status-index",
			Fields: []string{"status", "location"},
			Type:   "json",
		}

		assert.Equal(t, "status-index", index.Name)
		assert.Equal(t, []string{"status", "location"}, index.Fields)
		assert.Equal(t, "json", index.Type)
	})

	t.Run("default type", func(t *testing.T) {
		index := Index{
			Name:   "simple-index",
			Fields: []string{"field1"},
		}

		assert.Empty(t, index.Type)
	})
}

// TestDesignDoc tests DesignDoc and View structures
func TestDesignDoc(t *testing.T) {
	t.Run("create design doc", func(t *testing.T) {
		designDoc := DesignDoc{
			ID:       "_design/test",
			Language: "javascript",
			Views: map[string]View{
				"by_status": {
					Name: "by_status",
					Map: `function(doc) {
						if (doc.status) {
							emit(doc.status, 1);
						}
					}`,
					Reduce: "_count",
				},
			},
		}

		assert.Equal(t, "_design/test", designDoc.ID)
		assert.Equal(t, "javascript", designDoc.Language)
		assert.Contains(t, designDoc.Views, "by_status")
		assert.Equal(t, "_count", designDoc.Views["by_status"].Reduce)
	})
}

// TestViewOptions tests ViewOptions structure
func TestViewOptions(t *testing.T) {
	t.Run("all options", func(t *testing.T) {
		opts := ViewOptions{
			Key:         "test-key",
			StartKey:    "a",
			EndKey:      "z",
			IncludeDocs: true,
			Limit:       100,
			Skip:        50,
			Descending:  true,
			Reduce:      true,
			Group:       true,
			GroupLevel:  2,
		}

		assert.Equal(t, "test-key", opts.Key)
		assert.Equal(t, "a", opts.StartKey)
		assert.Equal(t, "z", opts.EndKey)
		assert.True(t, opts.IncludeDocs)
		assert.Equal(t, 100, opts.Limit)
		assert.Equal(t, 50, opts.Skip)
		assert.True(t, opts.Descending)
		assert.True(t, opts.Reduce)
		assert.True(t, opts.Group)
		assert.Equal(t, 2, opts.GroupLevel)
	})
}

// TestChangesFeedOptions tests ChangesFeedOptions structure
func TestChangesFeedOptions(t *testing.T) {
	t.Run("continuous feed", func(t *testing.T) {
		opts := ChangesFeedOptions{
			Since:       "now",
			Feed:        "continuous",
			IncludeDocs: true,
			Heartbeat:   60000,
			Timeout:     30000,
			Limit:       100,
			Descending:  false,
			Filter:      "_selector",
			Selector: map[string]interface{}{
				"@type": "SoftwareApplication",
			},
		}

		assert.Equal(t, "now", opts.Since)
		assert.Equal(t, "continuous", opts.Feed)
		assert.True(t, opts.IncludeDocs)
		assert.Equal(t, 60000, opts.Heartbeat)
		assert.Equal(t, 30000, opts.Timeout)
		assert.Contains(t, opts.Selector, "@type")
	})
}

// TestTraversalOptions tests TraversalOptions structure
func TestTraversalOptions(t *testing.T) {
	t.Run("forward traversal", func(t *testing.T) {
		opts := TraversalOptions{
			StartID:       "node-1",
			RelationField: "dependsOn",
			Direction:     "forward",
			Depth:         5,
			Filter: map[string]interface{}{
				"@type": "SoftwareApplication",
			},
		}

		assert.Equal(t, "node-1", opts.StartID)
		assert.Equal(t, "dependsOn", opts.RelationField)
		assert.Equal(t, "forward", opts.Direction)
		assert.Equal(t, 5, opts.Depth)
		assert.Equal(t, "SoftwareApplication", opts.Filter["@type"])
	})

	t.Run("reverse traversal", func(t *testing.T) {
		opts := TraversalOptions{
			StartID:       "node-5",
			RelationField: "hostedOn",
			Direction:     "reverse",
			Depth:         10,
		}

		assert.Equal(t, "node-5", opts.StartID)
		assert.Equal(t, "hostedOn", opts.RelationField)
		assert.Equal(t, "reverse", opts.Direction)
		assert.Equal(t, 10, opts.Depth)
	})
}

// TestBulkResult tests BulkResult structure
func TestBulkResult(t *testing.T) {
	t.Run("successful result", func(t *testing.T) {
		result := BulkResult{
			OK:  true,
			ID:  "doc-123",
			Rev: "1-abc",
		}

		assert.True(t, result.OK)
		assert.Equal(t, "doc-123", result.ID)
		assert.Equal(t, "1-abc", result.Rev)
		assert.Empty(t, result.Error)
	})

	t.Run("error result", func(t *testing.T) {
		result := BulkResult{
			OK:     false,
			ID:     "doc-456",
			Error:  "conflict",
			Reason: "Document update conflict",
		}

		assert.False(t, result.OK)
		assert.Equal(t, "doc-456", result.ID)
		assert.Equal(t, "conflict", result.Error)
		assert.Equal(t, "Document update conflict", result.Reason)
		assert.Empty(t, result.Rev)
	})
}

// TestRelationshipGraph tests RelationshipGraph structure
func TestRelationshipGraph(t *testing.T) {
	t.Run("build graph", func(t *testing.T) {
		node1 := json.RawMessage(`{"@id":"node-1","name":"Node 1"}`)
		node2 := json.RawMessage(`{"@id":"node-2","name":"Node 2"}`)
		node3 := json.RawMessage(`{"@id":"node-3","name":"Node 3"}`)

		graph := RelationshipGraph{
			Nodes: map[string]json.RawMessage{
				"node-1": node1,
				"node-2": node2,
				"node-3": node3,
			},
			Edges: []RelationshipEdge{
				{
					From: "node-1",
					To:   "node-2",
					Type: "dependsOn",
				},
				{
					From: "node-2",
					To:   "node-3",
					Type: "dependsOn",
				},
			},
		}

		assert.Len(t, graph.Nodes, 3)
		assert.Len(t, graph.Edges, 2)
		assert.Equal(t, "node-1", graph.Edges[0].From)
		assert.Equal(t, "node-2", graph.Edges[0].To)
		assert.Equal(t, "dependsOn", graph.Edges[0].Type)
	})
}

// TestChange tests Change and ChangeRev structures
func TestChange(t *testing.T) {
	t.Run("change with document", func(t *testing.T) {
		docJSON := json.RawMessage(`{"_id":"test","name":"Test Doc"}`)
		change := Change{
			Seq:     "123-abc",
			ID:      "test",
			Deleted: false,
			Changes: []ChangeRev{
				{Rev: "1-xyz"},
			},
			Doc: docJSON,
		}

		assert.Equal(t, "123-abc", change.Seq)
		assert.Equal(t, "test", change.ID)
		assert.False(t, change.Deleted)
		assert.Len(t, change.Changes, 1)
		assert.Equal(t, "1-xyz", change.Changes[0].Rev)
		assert.NotNil(t, change.Doc)

		var doc map[string]interface{}
		err := json.Unmarshal(change.Doc, &doc)
		require.NoError(t, err)
		assert.Equal(t, "Test Doc", doc["name"])
	})

	t.Run("deleted change", func(t *testing.T) {
		change := Change{
			Seq:     "456-def",
			ID:      "deleted-doc",
			Deleted: true,
			Changes: []ChangeRev{
				{Rev: "2-deleted"},
			},
		}

		assert.True(t, change.Deleted)
		assert.Nil(t, change.Doc)
	})
}

// TestDatabaseInfo tests DatabaseInfo structure
func TestDatabaseInfo(t *testing.T) {
	t.Run("database info", func(t *testing.T) {
		info := DatabaseInfo{
			DBName:    "testdb",
			DocCount:  1000,
			UpdateSeq: "5000-abc",
			DataSize:  1024000,
		}

		assert.Equal(t, "testdb", info.DBName)
		assert.Equal(t, int64(1000), info.DocCount)
		assert.Equal(t, "5000-abc", info.UpdateSeq)
		assert.Equal(t, int64(1024000), info.DataSize)
	})
}

// TestCouchDBConfig tests CouchDBConfig structure
func TestCouchDBConfig(t *testing.T) {
	t.Run("minimal config", func(t *testing.T) {
		config := CouchDBConfig{
			URL:      "http://localhost:5984",
			Database: "testdb",
		}

		assert.Equal(t, "http://localhost:5984", config.URL)
		assert.Equal(t, "testdb", config.Database)
		assert.Empty(t, config.Username)
		assert.Empty(t, config.Password)
		assert.Equal(t, 0, config.MaxConnections)
		assert.Equal(t, 0, config.Timeout)
		assert.False(t, config.CreateIfMissing)
	})

	t.Run("full config", func(t *testing.T) {
		config := CouchDBConfig{
			URL:             "http://localhost:5984",
			Database:        "testdb",
			Username:        "admin",
			Password:        "secret",
			MaxConnections:  10,
			Timeout:         30,
			CreateIfMissing: true,
			TLS: &TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: false,
				CertFile:           "/path/to/cert",
				KeyFile:            "/path/to/key",
				CAFile:             "/path/to/ca",
			},
		}

		assert.Equal(t, "admin", config.Username)
		assert.Equal(t, "secret", config.Password)
		assert.Equal(t, 10, config.MaxConnections)
		assert.Equal(t, 30, config.Timeout)
		assert.True(t, config.CreateIfMissing)
		assert.NotNil(t, config.TLS)
		assert.True(t, config.TLS.Enabled)
		assert.False(t, config.TLS.InsecureSkipVerify)
	})
}
