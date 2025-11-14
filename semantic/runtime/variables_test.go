package runtime

import (
	"fmt"
	"testing"
)

func TestSubstituteVariables(t *testing.T) {
	// Create an action with variable references
	action := &RuntimeAction{
		Type:       "SearchAction",
		Identifier: "test-action",
		AllFields: map[string]interface{}{
			"query": map[string]interface{}{
				"queryInput": "${render-query.result.text}",
			},
			"target": map[string]interface{}{
				"additionalProperty": map[string]interface{}{
					"concept_scheme": "${CONCEPT_SCHEME}",
					"mixed":          "prefix-${PARAM}-suffix",
				},
			},
			"result": map[string]interface{}{
				"contentUrl": "/tmp/output_${HASH}.xml",
			},
			"noVariables": "plain text",
		},
	}

	// Create a simple map resolver
	resolver := &MapVariableResolver{
		Variables: map[string]string{
			"render-query.result.text": "SELECT * WHERE { ?s ?p ?o }",
			"CONCEPT_SCHEME":           "https://data.zeiss.com/IQS/4",
			"HASH":                     "abc123",
			"PARAM":                    "value",
		},
	}

	// Substitute
	substituted, err := SubstituteVariables(action, resolver)
	if err != nil {
		t.Fatalf("SubstituteVariables failed: %v", err)
	}

	// Verify substitutions
	query, err := substituted.GetField("query.queryInput")
	if err != nil {
		t.Fatalf("Failed to get query.queryInput: %v", err)
	}
	if query != "SELECT * WHERE { ?s ?p ?o }" {
		t.Errorf("query.queryInput not substituted correctly: got %v", query)
	}

	conceptScheme, err := substituted.GetField("target.additionalProperty.concept_scheme")
	if err != nil {
		t.Fatalf("Failed to get concept_scheme: %v", err)
	}
	if conceptScheme != "https://data.zeiss.com/IQS/4" {
		t.Errorf("concept_scheme not substituted correctly: got %v", conceptScheme)
	}

	// Verify mixed substitution
	mixed, err := substituted.GetField("target.additionalProperty.mixed")
	if err != nil {
		t.Fatalf("Failed to get mixed: %v", err)
	}
	if mixed != "prefix-value-suffix" {
		t.Errorf("mixed not substituted correctly: got %v", mixed)
	}

	// Verify contentUrl substitution
	contentUrl, err := substituted.GetField("result.contentUrl")
	if err != nil {
		t.Fatalf("Failed to get result.contentUrl: %v", err)
	}
	if contentUrl != "/tmp/output_abc123.xml" {
		t.Errorf("result.contentUrl not substituted correctly: got %v", contentUrl)
	}

	// Verify fields without variables are unchanged
	noVars, err := substituted.GetField("noVariables")
	if err != nil {
		t.Fatalf("Failed to get noVariables: %v", err)
	}
	if noVars != "plain text" {
		t.Errorf("noVariables changed: got %v", noVars)
	}

	// Verify original action is unchanged (deep copy worked)
	original, _ := action.GetField("query.queryInput")
	if original != "${render-query.result.text}" {
		t.Error("Original action was modified")
	}
}

func TestSubstituteString(t *testing.T) {
	resolver := &MapVariableResolver{
		Variables: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		},
	}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "single variable",
			input: "${VAR1}",
			want:  "value1",
		},
		{
			name:  "multiple variables",
			input: "${VAR1} and ${VAR2}",
			want:  "value1 and value2",
		},
		{
			name:  "variable in middle",
			input: "prefix-${VAR1}-suffix",
			want:  "prefix-value1-suffix",
		},
		{
			name:  "no variables",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "duplicate variable",
			input: "${VAR1} ${VAR1}",
			want:  "value1 value1",
		},
		{
			name:    "missing variable",
			input:   "${MISSING}",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := substituteString(tt.input, resolver)
			if (err != nil) != tt.wantErr {
				t.Errorf("substituteString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("substituteString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActionResultResolver(t *testing.T) {
	// Create a completed action to reference
	completedAction := &RuntimeAction{
		Identifier:   "render-query",
		ActionStatus: "CompletedActionStatus",
		AllFields: map[string]interface{}{
			"result": map[string]interface{}{
				"text":       "SELECT * WHERE { ?s ?p ?o }",
				"contentUrl": "/tmp/query.txt",
			},
		},
	}

	// Create resolver
	resolver := &ActionResultResolver{
		GetAction: func(actionID string) (*RuntimeAction, error) {
			if actionID == "render-query" {
				return completedAction, nil
			}
			return nil, fmt.Errorf("action not found: %s", actionID)
		},
	}

	tests := []struct {
		name      string
		reference string
		want      string
		wantErr   bool
	}{
		{
			name:      "resolve result.text",
			reference: "render-query.result.text",
			want:      "SELECT * WHERE { ?s ?p ?o }",
		},
		{
			name:      "resolve result.contentUrl",
			reference: "render-query.result.contentUrl",
			want:      "/tmp/query.txt",
		},
		{
			name:      "action not found",
			reference: "missing-action.result.text",
			wantErr:   true,
		},
		{
			name:      "field not found",
			reference: "render-query.result.missing",
			wantErr:   true,
		},
		{
			name:      "invalid format",
			reference: "PARAMETER",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Resolve(tt.reference)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActionResultResolver_NotCompleted(t *testing.T) {
	// Create an action that hasn't completed yet
	pendingAction := &RuntimeAction{
		Identifier:   "pending-action",
		ActionStatus: "ActiveActionStatus",
		AllFields: map[string]interface{}{
			"result": map[string]interface{}{
				"text": "some value",
			},
		},
	}

	resolver := &ActionResultResolver{
		GetAction: func(actionID string) (*RuntimeAction, error) {
			return pendingAction, nil
		},
	}

	// Should fail because action hasn't completed
	_, err := resolver.Resolve("pending-action.result.text")
	if err == nil {
		t.Error("Expected error for non-completed action, got nil")
	}
}

func TestChainVariableResolver(t *testing.T) {
	// First resolver - handles parameters
	paramResolver := &MapVariableResolver{
		Variables: map[string]string{
			"PARAM1": "value1",
			"PARAM2": "value2",
		},
	}

	// Second resolver - handles action references
	completedAction := &RuntimeAction{
		Identifier:   "action1",
		ActionStatus: "CompletedActionStatus",
		AllFields: map[string]interface{}{
			"result": map[string]interface{}{
				"text": "action result",
			},
		},
	}

	actionResolver := &ActionResultResolver{
		GetAction: func(actionID string) (*RuntimeAction, error) {
			if actionID == "action1" {
				return completedAction, nil
			}
			return nil, fmt.Errorf("not found")
		},
	}

	// Chain them
	chain := &ChainVariableResolver{
		Resolvers: []VariableResolver{paramResolver, actionResolver},
	}

	// Test parameter resolution (first resolver)
	result, err := chain.Resolve("PARAM1")
	if err != nil {
		t.Errorf("Failed to resolve PARAM1: %v", err)
	}
	if result != "value1" {
		t.Errorf("Expected value1, got %s", result)
	}

	// Test action resolution (second resolver)
	result, err = chain.Resolve("action1.result.text")
	if err != nil {
		t.Errorf("Failed to resolve action1.result.text: %v", err)
	}
	if result != "action result" {
		t.Errorf("Expected 'action result', got %s", result)
	}

	// Test not found in any resolver
	_, err = chain.Resolve("MISSING")
	if err == nil {
		t.Error("Expected error for missing variable")
	}
}

func TestExtractVariableReferences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single variable",
			input: "${VAR1}",
			want:  []string{"VAR1"},
		},
		{
			name:  "multiple variables",
			input: "${VAR1} and ${VAR2}",
			want:  []string{"VAR1", "VAR2"},
		},
		{
			name:  "nested reference",
			input: "${action.result.field}",
			want:  []string{"action.result.field"},
		},
		{
			name:  "no variables",
			input: "plain text",
			want:  nil,
		},
		{
			name:  "duplicate variable",
			input: "${VAR} ${VAR}",
			want:  []string{"VAR", "VAR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVariableReferences(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractVariableReferences() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractVariableReferences()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestHasVariableReferences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "has variable",
			input: "${VAR}",
			want:  true,
		},
		{
			name:  "has variable in middle",
			input: "text ${VAR} more text",
			want:  true,
		},
		{
			name:  "no variable",
			input: "plain text",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasVariableReferences(tt.input); got != tt.want {
				t.Errorf("HasVariableReferences() = %v, want %v", got, tt.want)
			}
		})
	}
}
