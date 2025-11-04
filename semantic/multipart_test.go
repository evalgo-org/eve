package semantic

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseMultipartSemanticRequest_CreateAction(t *testing.T) {
	// Create a multipart form with CreateAction and config file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add JSON-LD action
	actionJSON := `{
		"@context": "https://schema.org",
		"@type": "CreateAction",
		"identifier": "create-test-repo",
		"name": "Create Test Repository",
		"result": {
			"@type": "SoftwareSourceCode",
			"identifier": "test-repo",
			"codeRepository": "http://localhost:7200/repositories/test-repo",
			"additionalProperty": {
				"serverUrl": "http://localhost:7200",
				"username": "admin",
				"password": "admin"
			}
		}
	}`
	writer.WriteField("action", actionJSON)

	// Add config file
	configWriter, err := writer.CreateFormFile("config", "repo-config.ttl")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	configWriter.Write([]byte("@prefix rep: <http://www.openrdf.org/config/repository#> ."))

	writer.Close()

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/api/action", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Parse multipart request
	semanticReq, err := ParseMultipartSemanticRequest(req)
	if err != nil {
		t.Fatalf("Failed to parse multipart request: %v", err)
	}

	// Verify action type
	action, ok := semanticReq.Action.(*CreateAction)
	if !ok {
		t.Fatalf("Expected CreateAction, got %T", semanticReq.Action)
	}

	if action.Identifier != "create-test-repo" {
		t.Errorf("Expected identifier 'create-test-repo', got '%s'", action.Identifier)
	}

	// Verify file exists
	if !semanticReq.HasFile("config") {
		t.Error("Expected config file to exist")
	}

	configFile, err := semanticReq.GetFile("config")
	if err != nil {
		t.Errorf("Failed to get config file: %v", err)
	}

	if configFile.Filename != "repo-config.ttl" {
		t.Errorf("Expected filename 'repo-config.ttl', got '%s'", configFile.Filename)
	}

	// Read file content
	content, err := ReadFileContent(configFile)
	if err != nil {
		t.Errorf("Failed to read file content: %v", err)
	}

	if !strings.Contains(string(content), "@prefix rep:") {
		t.Error("File content doesn't match expected Turtle content")
	}
}

func TestParseMultipartSemanticRequest_UploadAction(t *testing.T) {
	// Create a multipart form with UploadAction and data files
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add JSON-LD action
	actionJSON := `{
		"@context": "https://schema.org",
		"@type": "UploadAction",
		"identifier": "import-graph",
		"name": "Import Graph Data",
		"object": {
			"@type": "Dataset",
			"identifier": "http://example.org/graph/test"
		},
		"target": {
			"@type": "DataCatalog",
			"identifier": "test-repo",
			"url": "http://localhost:7200",
			"additionalProperty": {
				"serverUrl": "http://localhost:7200",
				"username": "admin",
				"password": "admin"
			}
		}
	}`
	writer.WriteField("action", actionJSON)

	// Add multiple data files
	dataWriter1, _ := writer.CreateFormFile("data", "data1.ttl")
	dataWriter1.Write([]byte("@prefix ex: <http://example.org/> ."))

	dataWriter2, _ := writer.CreateFormFile("data", "data2.rdf")
	dataWriter2.Write([]byte("<rdf:RDF></rdf:RDF>"))

	writer.Close()

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/api/action", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Parse multipart request
	semanticReq, err := ParseMultipartSemanticRequest(req)
	if err != nil {
		t.Fatalf("Failed to parse multipart request: %v", err)
	}

	// Verify action type
	action, ok := semanticReq.Action.(*UploadAction)
	if !ok {
		t.Fatalf("Expected UploadAction, got %T", semanticReq.Action)
	}

	if action.Identifier != "import-graph" {
		t.Errorf("Expected identifier 'import-graph', got '%s'", action.Identifier)
	}

	// Verify multiple files
	dataFiles, err := semanticReq.GetFiles("data")
	if err != nil {
		t.Errorf("Failed to get data files: %v", err)
	}

	if len(dataFiles) != 2 {
		t.Errorf("Expected 2 data files, got %d", len(dataFiles))
	}
}

func TestParseMultipartSemanticRequest_MissingAction(t *testing.T) {
	// Create a multipart form without action field
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	configWriter, _ := writer.CreateFormFile("config", "repo-config.ttl")
	configWriter.Write([]byte("test content"))

	writer.Close()

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/api/action", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Parse should fail
	_, err := ParseMultipartSemanticRequest(req)
	if err == nil {
		t.Error("Expected error for missing action field, got nil")
	}
}

func TestCreateActionWithConfigFile(t *testing.T) {
	repo := NewGraphDBRepository("http://localhost:7200", "test-repo", "admin", "admin")
	action := NewCreateAction("create-repo", "Create Repository", repo)

	// Add config file metadata
	action = CreateActionWithConfigFile(action, "repo-config.ttl", "text/turtle", 1024)

	// Verify metadata was added
	resultRepo, ok := action.Result.(*GraphDBRepository)
	if !ok {
		t.Fatalf("Expected GraphDBRepository, got %T", action.Result)
	}

	configFile, ok := resultRepo.Properties["configFile"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected configFile in properties")
	}

	if configFile["fileName"] != "repo-config.ttl" {
		t.Errorf("Expected fileName 'repo-config.ttl', got '%v'", configFile["fileName"])
	}

	if configFile["fileSize"] != int64(1024) {
		t.Errorf("Expected fileSize 1024, got %v", configFile["fileSize"])
	}
}

func TestUploadActionWithDataFiles(t *testing.T) {
	graph := NewGraphDBGraph("http://example.org/graph/test", "http://localhost:7200", "test-repo")
	catalog := &DataCatalog{
		Type:       "DataCatalog",
		Identifier: "test-repo",
		URL:        "http://localhost:7200",
	}
	action := NewUploadAction("import-data", "Import Data", graph, catalog)

	// Add data file metadata
	fileNames := []string{"data1.ttl", "data2.rdf", "data3.n3"}
	action = UploadActionWithDataFiles(action, fileNames)

	// Verify metadata was added
	resultGraph, ok := action.Object.(*GraphDBGraph)
	if !ok {
		t.Fatalf("Expected GraphDBGraph, got %T", action.Object)
	}

	dataFiles, ok := resultGraph.Properties["dataFiles"].([]string)
	if !ok {
		t.Fatal("Expected dataFiles in properties")
	}

	if len(dataFiles) != 3 {
		t.Errorf("Expected 3 data files, got %d", len(dataFiles))
	}

	if dataFiles[0] != "data1.ttl" {
		t.Errorf("Expected first file 'data1.ttl', got '%s'", dataFiles[0])
	}
}
