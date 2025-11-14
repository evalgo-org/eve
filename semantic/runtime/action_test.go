package runtime

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRuntimeAction_MarshalUnmarshal(t *testing.T) {
	// Test that unmarshaling and marshaling preserves all fields
	inputJSON := `{
		"@context": "https://schema.org",
		"@type": "SearchAction",
		"identifier": "get-empolis-data-sparql",
		"name": "Get Empolis Data from PoolParty",
		"description": "Execute SPARQL query to fetch concept data",
		"query": {
			"@type": "SearchAction",
			"queryInput": "SELECT * WHERE { ?s ?p ?o }"
		},
		"target": {
			"@type": "EntryPoint",
			"url": "registry://sparqlservice/v1/api/semantic/action",
			"httpMethod": "POST",
			"contentType": "application/ld+json",
			"additionalProperty": {
				"sparql_endpoint": "https://example.com/sparql",
				"custom_field": "custom_value"
			}
		},
		"result": {
			"@type": "MediaObject",
			"identifier": "empolis-raw-data",
			"contentUrl": "/tmp/output.xml",
			"encodingFormat": "application/rdf+xml"
		},
		"requires": ["render-empolis-query"],
		"isPartOf": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
		"exampleOfWork": "iqs-cache-empolis-jsons/get-empolis-data-sparql",
		"actionStatus": "PotentialActionStatus",
		"customField1": "value1",
		"customField2": {
			"nested": "value2"
		}
	}`

	// Unmarshal
	var action RuntimeAction
	if err := json.Unmarshal([]byte(inputJSON), &action); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify typed fields
	if action.Type != "SearchAction" {
		t.Errorf("Expected Type=SearchAction, got %s", action.Type)
	}
	if action.Identifier != "get-empolis-data-sparql" {
		t.Errorf("Expected Identifier=get-empolis-data-sparql, got %s", action.Identifier)
	}
	if action.ActionStatus != "PotentialActionStatus" {
		t.Errorf("Expected ActionStatus=PotentialActionStatus, got %s", action.ActionStatus)
	}
	if len(action.Requires) != 1 || action.Requires[0] != "render-empolis-query" {
		t.Errorf("Expected Requires=[render-empolis-query], got %v", action.Requires)
	}

	// Verify AllFields contains all fields
	if action.AllFields == nil {
		t.Fatal("AllFields is nil")
	}
	if _, ok := action.AllFields["customField1"]; !ok {
		t.Error("AllFields missing customField1")
	}
	if _, ok := action.AllFields["customField2"]; !ok {
		t.Error("AllFields missing customField2")
	}

	// Marshal back
	output, err := json.Marshal(&action)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal output to compare
	var outputMap map[string]interface{}
	if err := json.Unmarshal(output, &outputMap); err != nil {
		t.Fatalf("Failed to unmarshal output: %v", err)
	}

	// Verify custom fields are preserved
	if outputMap["customField1"] != "value1" {
		t.Errorf("customField1 lost or changed")
	}
	if customField2, ok := outputMap["customField2"].(map[string]interface{}); !ok {
		t.Error("customField2 lost or not an object")
	} else if customField2["nested"] != "value2" {
		t.Error("customField2.nested lost or changed")
	}

	// Verify typed fields in output
	if outputMap["@type"] != "SearchAction" {
		t.Error("@type lost or changed")
	}
	if outputMap["identifier"] != "get-empolis-data-sparql" {
		t.Error("identifier lost or changed")
	}
}

func TestRuntimeAction_GetField(t *testing.T) {
	action := &RuntimeAction{
		AllFields: map[string]interface{}{
			"result": map[string]interface{}{
				"contentUrl": "/tmp/output.xml",
				"nested": map[string]interface{}{
					"deep": "value",
				},
			},
			"simple": "simple_value",
		},
	}

	tests := []struct {
		name    string
		path    string
		want    interface{}
		wantErr bool
	}{
		{
			name: "simple field",
			path: "simple",
			want: "simple_value",
		},
		{
			name: "nested field",
			path: "result.contentUrl",
			want: "/tmp/output.xml",
		},
		{
			name: "deep nested field",
			path: "result.nested.deep",
			want: "value",
		},
		{
			name:    "non-existent field",
			path:    "missing",
			wantErr: true,
		},
		{
			name:    "non-existent nested field",
			path:    "result.missing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := action.GetField(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("GetField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRuntimeAction_SetField(t *testing.T) {
	action := &RuntimeAction{
		AllFields: make(map[string]interface{}),
	}

	// Set simple field
	if err := action.SetField("simple", "value"); err != nil {
		t.Errorf("Failed to set simple field: %v", err)
	}

	// Set nested field (creates intermediate maps)
	if err := action.SetField("result.contentUrl", "/tmp/output.xml"); err != nil {
		t.Errorf("Failed to set nested field: %v", err)
	}

	// Verify
	if action.AllFields["simple"] != "value" {
		t.Error("simple field not set correctly")
	}

	result, ok := action.AllFields["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	if result["contentUrl"] != "/tmp/output.xml" {
		t.Error("result.contentUrl not set correctly")
	}
}

func TestRuntimeAction_DeepCopy(t *testing.T) {
	original := &RuntimeAction{
		Type:       "SearchAction",
		Identifier: "test-action",
		AllFields: map[string]interface{}{
			"customField": "value",
			"nested": map[string]interface{}{
				"field": "nested_value",
			},
		},
	}

	copy := original.DeepCopy()

	// Verify copy has same values
	if copy.Type != original.Type {
		t.Error("Type not copied")
	}
	if copy.Identifier != original.Identifier {
		t.Error("Identifier not copied")
	}
	if copy.AllFields["customField"] != "value" {
		t.Error("customField not copied")
	}

	// Modify copy
	copy.Type = "UpdateAction"
	copy.AllFields["customField"] = "modified"

	// Verify original is unchanged
	if original.Type != "SearchAction" {
		t.Error("Modifying copy affected original Type")
	}
	if original.AllFields["customField"] != "value" {
		t.Error("Modifying copy affected original AllFields")
	}
}

func TestRuntimeAction_TimestampHandling(t *testing.T) {
	now := time.Now()

	action := &RuntimeAction{
		Type:         "SearchAction",
		DateCreated:  now,
		DateModified: now,
	}

	// Marshal
	data, err := json.Marshal(action)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var unmarshaled RuntimeAction
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify timestamps preserved (with tolerance for RFC3339 precision)
	if unmarshaled.DateCreated.Unix() != now.Unix() {
		t.Errorf("DateCreated not preserved: got %v, want %v", unmarshaled.DateCreated, now)
	}
	if unmarshaled.DateModified.Unix() != now.Unix() {
		t.Errorf("DateModified not preserved: got %v, want %v", unmarshaled.DateModified, now)
	}
}

func TestWalkJSON(t *testing.T) {
	input := map[string]interface{}{
		"simple": "value",
		"nested": map[string]interface{}{
			"field": "nested_value",
		},
		"array":  []interface{}{"item1", "item2"},
		"number": 42,
	}

	// Function to uppercase all strings
	uppercaseFn := func(s string) (string, error) {
		return strings.ToUpper(s), nil
	}

	result, err := WalkJSON(input, uppercaseFn)
	if err != nil {
		t.Fatalf("WalkJSON failed: %v", err)
	}

	resultMap := result.(map[string]interface{})

	// Verify transformations
	if resultMap["simple"] != "VALUE" {
		t.Error("simple field not transformed")
	}

	nested := resultMap["nested"].(map[string]interface{})
	if nested["field"] != "NESTED_VALUE" {
		t.Error("nested field not transformed")
	}

	array := resultMap["array"].([]interface{})
	if array[0] != "ITEM1" || array[1] != "ITEM2" {
		t.Error("array elements not transformed")
	}

	// Verify non-strings unchanged
	if resultMap["number"] != 42 {
		t.Error("number changed")
	}
}
