# Semantic Result Structure Migration Guide

## Overview

This guide helps you migrate your EVE services from the old type-guessing approach to the new **semantic Result structure** based on Schema.org patterns.

## Why This Change?

### Old Approach Problems
- **Type guessing**: Code tried to guess result formats based on action types
- **Service-specific parsing**: Different parsing logic for each service
- **No schema**: No contract for what services should return
- **Hard-coded paths**: Looking for nested fields like `additionalProperty.result`

### New Semantic Approach Benefits
- **Schema-driven**: Services declare their result structure using Schema.org vocabulary
- **Type-safe**: Result schema defines what fields are available
- **Extensible**: Easy to add new result types without modifying executor
- **Discoverable**: Consumers can inspect the schema to understand results

## Changes to SemanticResult

### Before
```go
type SemanticResult struct {
    Type   string `json:"@type"`
    Output string `json:"text,omitempty"`
    Value  int    `json:"value,omitempty"`  // ❌ Only integers
}
```

### After
```go
type SemanticResult struct {
    Type         string        `json:"@type"`                   // "Result", "Dataset", "DigitalDocument"
    ActionStatus string        `json:"actionStatus,omitempty"`
    Output       string        `json:"text,omitempty"`          // Raw output
    Value        interface{}   `json:"value,omitempty"`         // ✅ Any structured data
    Format       string        `json:"encodingFormat,omitempty"` // MIME type
    Schema       *ResultSchema `json:"about,omitempty"`         // ✅ Describes structure
}

type ResultSchema struct {
    Type       string              `json:"@type"`
    Properties []PropertyValueSpec `json:"variableMeasured,omitempty"`
}
```

## Migration Steps for Services

### 1. Credential Services (like Infisicalservice)

**Before** (storing in Properties):
```go
secrets := []interface{}{
    map[string]string{"name": "KEY1", "value": "val1"},
    map[string]string{"name": "KEY2", "value": "val2"},
}
action.Properties["result"] = secrets  // ❌ Wrong location
```

**After** (using semantic Result):
```go
secrets := []interface{}{
    map[string]string{"name": "KEY1", "value": "val1"},
    map[string]string{"name": "KEY2", "value": "val2"},
}

action.Result = &semantic.SemanticResult{
    Type:   "Dataset",                    // Schema.org Dataset type
    Format: "application/json",
    Value:  secrets,                      // Structured credentials
    Schema: &semantic.ResultSchema{
        Type: "PropertyValueList",        // Declares it's a list of properties
        Properties: []semantic.PropertyValueSpec{
            {Type: "PropertyValue", Name: "name", ValueType: "Text"},
            {Type: "PropertyValue", Name: "value", ValueType: "Text"},
        },
    },
}
```

### 2. Query Services (like SPARQLservice)

**After**:
```go
action.Result = &semantic.SemanticResult{
    Type:   "Dataset",
    Format: "application/sparql-results+json",
    Output: string(resultsJSON),  // Raw SPARQL JSON
    Value:  parsedBindings,        // Structured query results
    Schema: &semantic.ResultSchema{
        Type: "Dataset",
        Properties: extractedVariables,  // SPARQL variables
    },
}
```

### 3. Storage Services (like S3service, WorkflowStorageService)

**After**:
```go
action.Result = &semantic.SemanticResult{
    Type:   "DigitalDocument",
    Format: originalMimeType,
    Value: map[string]interface{}{
        "contentUrl":      storedPath,         // File location
        "encodingFormat":  originalFormat,
        "contentSize":     fileSize,
    },
}
```

### 4. Transform Services (like BaseXservice with XQuery)

**After**:
```go
action.Result = &semantic.SemanticResult{
    Type:   "Dataset",
    Format: "application/xml",
    Output: string(transformedXML),  // Raw XML output
    Value: map[string]interface{}{
        "documentCount": count,
        "transformedAt": time.Now(),
    },
}
```

## Executor Changes

### Old Approach (in when/executor.go)
```go
// ❌ Type guessing and nested path checking
if actionType == "RetrieveAction" {
    var response struct {
        AdditionalProperty struct {
            Result []PropertyValue `json:"result"`
        } `json:"additionalProperty"`
    }
    json.Unmarshal(...)  // Hope the structure matches!
}
```

### New Semantic Approach
```go
// ✅ Use schema to understand structure
if result.Schema != nil && result.Schema.Type == "PropertyValueList" {
    extractCredentials(result.Value, credentials)
}

if result.Type == "DigitalDocument" {
    extractFileReferences(result.Value, actionID, credentials)
}
```

## Result Type Guidelines

| Service Type | Result @type | Format | Use Case |
|--------------|--------------|--------|----------|
| Credentials  | Dataset | application/json | Secret management |
| Query Results | Dataset | application/sparql-results+json | SPARQL, SQL queries |
| File Storage | DigitalDocument | varies | File uploads/downloads |
| Transformations | Dataset | application/xml, etc | XQuery, XSLT |
| API Calls | Result | application/json | Generic HTTP responses |

## Testing Your Migration

1. **Update service to return semantic Result**
2. **Test service directly**:
```bash
curl -X POST http://yourservice:8080/v1/api/semantic/action \
  -H "Content-Type: application/json" \
  -d @test-action.json | jq '.result'
```

3. **Verify Result structure**:
```json
{
  "result": {
    "@type": "Dataset",
    "value": [...],
    "encodingFormat": "application/json",
    "about": {
      "@type": "PropertyValueList",
      "variableMeasured": [...]
    }
  }
}
```

4. **Test credential substitution** in workflow:
   - Credentials from `PropertyValueList` → available as `${KEY_NAME}`
   - File paths from `DigitalDocument` → available as `${action-id.contentUrl}`

## Example: Complete Migration

### Infisicalservice Migration

**File**: `infisicalservice/cmd/infisicalservice/semantic_api.go`

```go
// OLD (lines 59-72)
secrets, err := fetchSecretsFromInfisical(...)
if err != nil {
    return semantic.ReturnActionError(c, action, "Failed to retrieve", err)
}
action.Properties["result"] = secrets  // ❌
semantic.SetSuccessOnAction(action)
return c.JSON(http.StatusOK, action)

// NEW
secrets, err := fetchSecretsFromInfisical(...)
if err != nil {
    return semantic.ReturnActionError(c, action, "Failed to retrieve", err)
}

action.Result = &semantic.SemanticResult{
    Type:   "Dataset",
    Format: "application/json",
    Value:  secrets,
    Schema: &semantic.ResultSchema{
        Type: "PropertyValueList",
        Properties: []semantic.PropertyValueSpec{
            {Type: "PropertyValue", Name: "name", ValueType: "Text", Description: "Secret key name"},
            {Type: "PropertyValue", Name: "value", ValueType: "Text", Description: "Secret value"},
        },
    },
}

semantic.SetSuccessOnAction(action)
return c.JSON(http.StatusOK, action)
```

## Checklist for Service Migration

- [ ] Update EVE dependency to version with new SemanticResult
- [ ] Change `action.Properties["result"]` to `action.Result`
- [ ] Set appropriate `@type` (Dataset, DigitalDocument, Result)
- [ ] Populate `Value` with structured data
- [ ] Set `Format` (MIME type)
- [ ] Define `Schema` with ResultSchema
- [ ] Test service returns correct Result structure
- [ ] Verify credentials/files are extracted in workflows

## References

- **EVE Types**: `/home/opunix/eve/semantic/types.go`
- **Executor Logic**: `/home/opunix/when/executor.go` (lines 882-904)
- **Example Service**: `/home/opunix/infisicalservice/cmd/infisicalservice/semantic_api.go`
- **Schema.org Dataset**: https://schema.org/Dataset
- **Schema.org PropertyValue**: https://schema.org/PropertyValue
