package semantic

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// MultipartSemanticRequest represents a semantic action with optional file attachments
// This allows CreateAction, UploadAction, etc. to include configuration or data files
type MultipartSemanticRequest struct {
	Action interface{}                        // The semantic action (TransferAction, CreateAction, etc.)
	Files  map[string][]*multipart.FileHeader // Uploaded files keyed by form field name
}

// ParseMultipartSemanticRequest parses a multipart/form-data request containing:
// - "action" field: JSON-LD semantic action
// - File fields: Optional file attachments (config files, data files, etc.)
//
// Example form fields:
//   - action: {"@type":"CreateAction", "identifier":"create-repo", ...}
//   - config: repo-config.ttl (Turtle configuration file)
//   - data: graph-data.rdf (RDF data file)
//
// Returns a MultipartSemanticRequest with parsed action and files
func ParseMultipartSemanticRequest(r *http.Request) (*MultipartSemanticRequest, error) {
	// Parse multipart form with 32MB memory limit
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	form := r.MultipartForm
	if form == nil {
		return nil, fmt.Errorf("no multipart form data found")
	}

	// Extract JSON-LD action from "action" field
	// It can be either a form value or a file upload
	var actionJSON string

	// Try form value first
	if actionFields, exists := form.Value["action"]; exists && len(actionFields) > 0 {
		actionJSON = actionFields[0]
	} else if actionFiles, exists := form.File["action"]; exists && len(actionFiles) > 0 {
		// Read from uploaded file
		actionFile, err := actionFiles[0].Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open action file: %w", err)
		}
		defer actionFile.Close()

		actionBytes, err := io.ReadAll(actionFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read action file: %w", err)
		}
		actionJSON = string(actionBytes)
	} else {
		return nil, fmt.Errorf("missing 'action' field in multipart form")
	}

	// Parse the JSON-LD action
	var actionData map[string]interface{}
	if err := json.Unmarshal([]byte(actionJSON), &actionData); err != nil {
		return nil, fmt.Errorf("invalid JSON in action field: %w", err)
	}

	// Determine action type and parse accordingly
	actionType, ok := actionData["@type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid @type in action")
	}

	var parsedAction interface{}

	switch actionType {
	case "TransferAction", "CreateAction", "DeleteAction", "UpdateAction", "UploadAction",
		"ActivateAction", "DeactivateAction", "DownloadAction", "ConnectAction", "AssignAction",
		"SearchAction", "RetrieveAction":
		// All these action types now use SemanticAction
		var action SemanticAction
		if err := json.Unmarshal([]byte(actionJSON), &action); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", actionType, err)
		}
		parsedAction = &action

	case "ItemList":
		// For workflows, we'll parse as generic map for now
		parsedAction = actionData

	default:
		return nil, fmt.Errorf("unsupported action type: %s", actionType)
	}

	// Extract uploaded files
	files := make(map[string][]*multipart.FileHeader)
	for key, fileHeaders := range form.File {
		files[key] = fileHeaders
	}

	return &MultipartSemanticRequest{
		Action: parsedAction,
		Files:  files,
	}, nil
}

// GetFile retrieves a specific file from the multipart request by field name
// Returns the first file if multiple files are uploaded with the same field name
func (m *MultipartSemanticRequest) GetFile(fieldName string) (*multipart.FileHeader, error) {
	fileHeaders, exists := m.Files[fieldName]
	if !exists || len(fileHeaders) == 0 {
		return nil, fmt.Errorf("no file found with field name: %s", fieldName)
	}
	return fileHeaders[0], nil
}

// GetFiles retrieves all files for a specific field name
func (m *MultipartSemanticRequest) GetFiles(fieldName string) ([]*multipart.FileHeader, error) {
	fileHeaders, exists := m.Files[fieldName]
	if !exists || len(fileHeaders) == 0 {
		return nil, fmt.Errorf("no files found with field name: %s", fieldName)
	}
	return fileHeaders, nil
}

// HasFile checks if a file with the given field name exists
func (m *MultipartSemanticRequest) HasFile(fieldName string) bool {
	fileHeaders, exists := m.Files[fieldName]
	return exists && len(fileHeaders) > 0
}

// ReadFileContent reads the content of a file into a byte slice
func ReadFileContent(fileHeader *multipart.FileHeader) ([]byte, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return content, nil
}

// SaveUploadedFile saves an uploaded file to a specific path
func SaveUploadedFile(fileHeader *multipart.FileHeader, destPath string) error {
	srcFile, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Note: Actual file writing should be done by the caller with proper error handling
	// This is a helper function that returns the open file for the caller to handle
	return fmt.Errorf("not implemented - use ReadFileContent instead")
}

// SemanticActionWithConfigFile is a helper to add config file metadata to a SemanticAction
// This adds metadata about the config file to the action's properties
func SemanticActionWithConfigFile(action *SemanticAction, fileName, fileType string, fileSize int64) *SemanticAction {
	if action == nil || action.Properties == nil {
		return action
	}

	// Add file metadata to the action properties
	action.Properties["configFile"] = map[string]interface{}{
		"fileName":       fileName,
		"fileType":       fileType,
		"fileSize":       fileSize,
		"encodingFormat": "text/turtle",
	}

	return action
}

// SemanticActionWithDataFiles is a helper to add data file metadata to a SemanticAction
// This tracks which files are being uploaded with the action
func SemanticActionWithDataFiles(action *SemanticAction, fileNames []string) *SemanticAction {
	if action == nil || action.Properties == nil {
		return action
	}

	// Add file metadata to the action properties
	action.Properties["dataFiles"] = fileNames

	return action
}
