// Package db provides database client implementations for BaseX, DragonflyDB,
// CouchDB, PoolParty, and other database systems. This package offers
// consistent interfaces for XML databases, key-value stores, document stores,
// and semantic repositories.
package db

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Commands represents a collection of BaseX commands for batch execution.
// It is marshaled to XML format for the BaseX REST API.
//
// Example XML:
//
//	<commands>
//	  <create-db name="mydb"/>
//	  <info-db/>
//	</commands>
type Commands struct {
	XMLName  xml.Name  `xml:"commands"`
	Commands []Command `xml:",any"`
}

// Command represents a single BaseX command with optional name attribute.
// The XMLName field determines the command type (e.g., "create-db", "info-db").
// Common commands include: create-db, info-db, drop-db, open, close, add, delete.
//
// Example usage:
//
//	cmd := Command{XMLName: xml.Name{Local: "create-db"}, Name: "mydb"}
type Command struct {
	XMLName xml.Name
	Name    string `xml:"name,attr,omitempty"`
}

// UploadResult represents the outcome of a file upload operation to BaseX.
// It provides detailed information about the upload status, including success state,
// status code, message, and the remote path where the file was stored.
type UploadResult struct {
	Success    bool
	Message    string
	RemotePath string
	StatusCode int
}

// BaseXCreateDB creates a new database in BaseX with the specified name.
// It sends a batch of commands to the BaseX REST endpoint including database
// creation and info retrieval.
//
// The function requires the following environment variables:
//   - BASEX_URL: The BaseX server URL (e.g., "http://localhost:8984")
//   - BASEX_USERNAME: Authentication username
//   - BASEX_PASSWORD: Authentication password
//
// Parameters:
//   - dbName: The name of the database to create
//
// Returns:
//   - error: Any error encountered during database creation
//
// Example:
//
//	err := BaseXCreateDB("mydb")
//	if err != nil {
//	    log.Fatal(err)
//	}
func BaseXCreateDB(dbName string) error {
	// Build your commands
	cmds := Commands{
		Commands: []Command{
			{XMLName: xml.Name{Local: "create-db"}, Name: dbName},
			{XMLName: xml.Name{Local: "info-db"}},
		},
	}

	// Marshal to XML
	xmlBytes, err := xml.MarshalIndent(cmds, "", " ")
	if err != nil {
		return err
	}

	// Print XML (for verification)
	fmt.Println(string(xmlBytes))

	// Create POST request to BaseX REST endpoint
	url := os.Getenv("BASEX_URL") + "/rest"
	req, err := http.NewRequest("POST", url, bytes.NewReader(xmlBytes))
	if err != nil {
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/xml")
	// Add basic auth if required
	req.SetBasicAuth(os.Getenv("BASEX_USERNAME"), os.Getenv("BASEX_PASSWORD"))

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Response Body:\n%s\n", string(body))
	return nil
}

// BaseXSaveDocument saves an XML document to a BaseX database.
// The document is stored with the specified ID and ".xml" extension.
//
// The function requires the following environment variables:
//   - BASEX_URL: The BaseX server URL
//   - BASEX_USERNAME: Authentication username
//   - BASEX_PASSWORD: Authentication password
//
// Parameters:
//   - dbName: The name of the target database
//   - docID: The document identifier (without .xml extension)
//   - xmlData: The XML document content as bytes
//
// Returns:
//   - error: Any error encountered during document save
//
// Example:
//
//	xmlContent := []byte("<book><title>Go Programming</title></book>")
//	err := BaseXSaveDocument("mydb", "book1", xmlContent)
func BaseXSaveDocument(dbName, docID string, xmlData []byte) error {
	// Target: save into db "mydb" with resource name "book1.xml"
	url := os.Getenv("BASEX_URL") + "/rest/" + dbName + "/" + docID + ".xml"
	fmt.Println(url)

	// Create POST request
	req, err := http.NewRequest("PUT", url, bytes.NewReader(xmlData))
	if err != nil {
		return err
	}

	// Headers
	req.Header.Set("Content-Type", "application/xml")
	req.SetBasicAuth(os.Getenv("BASEX_USERNAME"), os.Getenv("BASEX_PASSWORD"))

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Response Body:\n%s\n", string(body))
	return nil
}

// BaseXQuery executes an XQuery query against a specific document in a BaseX database.
// The query is sent as a POST request to the document endpoint with Content-Type "application/query+xml".
//
// The function requires the following environment variables:
//   - BASEX_URL: The BaseX server URL
//   - BASEX_USERNAME: Authentication username
//   - BASEX_PASSWORD: Authentication password
//
// Parameters:
//   - db: The database name
//   - doc: The document identifier (without .xml extension)
//   - query: The XQuery expression to execute
//
// Returns:
//   - []byte: The query results as bytes
//   - error: Any error encountered during query execution
//
// Example:
//
//	query := "//book[@year > 2020]"
//	results, err := BaseXQuery("mydb", "book1", query)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(string(results))
func BaseXQuery(db, doc, query string) ([]byte, error) {
	// REST endpoint: point to your DB (example: "mydb")
	url := os.Getenv("BASEX_URL") + "/rest/" + db + "/" + doc + ".xml"

	// Create POST request with query as body
	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(query)))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/query+xml")
	req.SetBasicAuth(os.Getenv("BASEX_USERNAME"), os.Getenv("BASEX_PASSWORD"))

	// Send
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Read results
	return io.ReadAll(resp.Body)
}

// BaseXUploadFile uploads a file to a BaseX database using the REST API.
// The content type is automatically determined based on the file extension.
//
// The function requires the following environment variables:
//   - BASEX_URL: The BaseX server URL
//   - BASEX_USERNAME: Authentication username
//   - BASEX_PASSWORD: Authentication password
//
// Parameters:
//   - dbName: The target database name
//   - localFilePath: Path to the local file to upload
//   - remotePath: The path/name to use in the database
//
// Returns:
//   - *UploadResult: Details about the upload operation
//   - error: Any error encountered during file upload
//
// Example:
//
//	result, err := BaseXUploadFile("mydb", "/tmp/data.json", "data.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if result.Success {
//	    fmt.Println("Upload successful:", result.RemotePath)
//	}
func BaseXUploadFile(dbName, localFilePath, remotePath string) (*UploadResult, error) {
	// Read file content
	fileContent, err := os.ReadFile(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create REST URL for file upload
	url := os.Getenv("BASEX_URL") + "/rest/" + dbName + "/" + remotePath

	// Create HTTP request
	req, err := http.NewRequest("PUT", url, bytes.NewReader(fileContent))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication
	req.SetBasicAuth(os.Getenv("BASEX_USERNAME"), os.Getenv("BASEX_PASSWORD"))

	// Set content type based on file extension
	contentType := getContentType(localFilePath)
	req.Header.Set("Content-Type", contentType)

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &UploadResult{
			Success:    true,
			Message:    "File uploaded successfully",
			RemotePath: remotePath,
			StatusCode: resp.StatusCode,
		}, nil
	}

	return &UploadResult{
		Success:    false,
		Message:    fmt.Sprintf("Upload failed: %s - %s", resp.Status, string(body)),
		RemotePath: remotePath,
		StatusCode: resp.StatusCode,
	}, nil
}

// BaseXUploadXMLFile uploads and validates an XML file to a BaseX database.
// The XML content is validated before upload to ensure proper structure.
// This function uses BaseXSaveDocument internally for XML-specific handling.
//
// The function requires the following environment variables:
//   - BASEX_URL: The BaseX server URL
//   - BASEX_USERNAME: Authentication username
//   - BASEX_PASSWORD: Authentication password
//
// Parameters:
//   - dbName: The target database name
//   - localFilePath: Path to the local XML file
//   - remotePath: The path/name to use in the database (should include .xml extension)
//
// Returns:
//   - *UploadResult: Details about the upload operation
//   - error: Any error encountered (including XML validation errors)
//
// Example:
//
//	result, err := BaseXUploadXMLFile("mydb", "/tmp/book.xml", "books/book1.xml")
//	if err != nil {
//	    log.Fatal(err)
//	}
func BaseXUploadXMLFile(dbName, localFilePath, remotePath string) (*UploadResult, error) {
	// Read XML file content
	xmlContent, err := os.ReadFile(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML file: %w", err)
	}

	// Validate that it's proper XML
	var temp interface{}
	if err := xml.Unmarshal(xmlContent, &temp); err != nil {
		return nil, fmt.Errorf("invalid XML file: %w", err)
	}

	// Use the existing BaseXSaveDocument function but return UploadResult
	docID := strings.TrimSuffix(remotePath, ".xml")
	err = BaseXSaveDocument(dbName, docID, xmlContent)
	if err != nil {
		return &UploadResult{
			Success: false,
			Message: fmt.Sprintf("XML upload failed: %v", err),
		}, nil
	}

	return &UploadResult{
		Success:    true,
		Message:    "XML file uploaded successfully",
		RemotePath: remotePath,
	}, nil
}

// BaseXUploadBinaryFile uploads a binary file to BaseX using base64 encoding.
// The file is encoded as base64 and stored using an XQuery command.
// This method is suitable for images, PDFs, archives, and other binary content.
//
// The function requires the following environment variables:
//   - BASEX_URL: The BaseX server URL
//   - BASEX_USERNAME: Authentication username
//   - BASEX_PASSWORD: Authentication password
//
// Parameters:
//   - dbName: The target database name
//   - localFilePath: Path to the local binary file
//   - remotePath: The path/name to use in the database
//
// Returns:
//   - *UploadResult: Details about the upload operation
//   - error: Any error encountered during binary upload
//
// Example:
//
//	result, err := BaseXUploadBinaryFile("mydb", "/tmp/image.jpg", "images/photo.jpg")
func BaseXUploadBinaryFile(dbName, localFilePath, remotePath string) (*UploadResult, error) {
	// Read file content
	fileContent, err := os.ReadFile(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Encode as base64
	base64Content := base64.StdEncoding.EncodeToString(fileContent)

	// Create XQuery to store binary file
	xquery := fmt.Sprintf(`db:add('%s', xs:base64Binary('%s'), '%s')`,
		dbName, base64Content, remotePath)

	// Execute XQuery
	url := os.Getenv("BASEX_URL") + "/rest"
	req, err := http.NewRequest("POST", url, strings.NewReader(xquery))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(os.Getenv("BASEX_USERNAME"), os.Getenv("BASEX_PASSWORD"))
	req.Header.Set("Content-Type", "application/xquery")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &UploadResult{
			Success:    true,
			Message:    "Binary file uploaded successfully",
			RemotePath: remotePath,
			StatusCode: resp.StatusCode,
		}, nil
	}

	return &UploadResult{
		Success:    false,
		Message:    fmt.Sprintf("Binary upload failed: %s - %s", resp.Status, string(body)),
		StatusCode: resp.StatusCode,
	}, nil
}

// BaseXUploadFileAuto automatically detects the file type and uploads using the
// appropriate method. XML files are validated and uploaded via BaseXUploadXMLFile,
// binary files (images, PDFs, archives) use base64 encoding via BaseXUploadBinaryFile,
// and other files use the standard REST upload via BaseXUploadFile.
//
// The function requires the following environment variables:
//   - BASEX_URL: The BaseX server URL
//   - BASEX_USERNAME: Authentication username
//   - BASEX_PASSWORD: Authentication password
//
// Parameters:
//   - dbName: The target database name
//   - localFilePath: Path to the local file
//   - remotePath: The path/name to use in the database
//
// Returns:
//   - *UploadResult: Details about the upload operation
//   - error: Any error encountered during upload
//
// Example:
//
//	result, err := BaseXUploadFileAuto("mydb", "/tmp/document.xml", "docs/doc1.xml")
//	// Automatically uses XML validation and upload
func BaseXUploadFileAuto(dbName, localFilePath, remotePath string) (*UploadResult, error) {
	ext := strings.ToLower(filepath.Ext(localFilePath))

	switch ext {
	case ".xml":
		return BaseXUploadXMLFile(dbName, localFilePath, remotePath)
	case ".jpg", ".jpeg", ".png", ".gif", ".pdf", ".zip", ".tar", ".gz":
		return BaseXUploadBinaryFile(dbName, localFilePath, remotePath)
	default:
		// Use standard REST upload for text files and unknown types
		return BaseXUploadFile(dbName, localFilePath, remotePath)
	}
}

// BaseXUploadToFilesystem uploads a file to the BaseX server's filesystem
// using a multipart form upload to a custom upload endpoint. This is different
// from database storage and places files directly on the server filesystem.
//
// The function requires the following environment variables:
//   - BASEX_URL: The BaseX server URL
//   - BASEX_USERNAME: Authentication username
//   - BASEX_PASSWORD: Authentication password
//
// Parameters:
//   - localFilePath: Path to the local file to upload
//   - remotePath: The target filesystem path on the server (sent via X-Target-Path header)
//
// Returns:
//   - *UploadResult: Details about the upload operation
//   - error: Any error encountered during filesystem upload
//
// Example:
//
//	result, err := BaseXUploadToFilesystem("/tmp/backup.zip", "/server/backups/backup.zip")
func BaseXUploadToFilesystem(localFilePath, remotePath string) (*UploadResult, error) {
	fileContent, err := os.ReadFile(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	fileWriter, err := writer.CreateFormFile("files", filepath.Base(localFilePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	fileWriter.Write(fileContent)

	writer.Close()

	url := fmt.Sprintf("%s/upload-filesystem", strings.TrimSuffix(os.Getenv("BASEX_URL"), "/"))
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(os.Getenv("BASEX_USERNAME"), os.Getenv("BASEX_PASSWORD"))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Target-Path", remotePath)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &UploadResult{
			Success:    true,
			Message:    fmt.Sprintf("File uploaded to filesystem: %s", remotePath),
			RemotePath: remotePath,
			StatusCode: resp.StatusCode,
		}, nil
	}

	return &UploadResult{
		Success:    false,
		Message:    fmt.Sprintf("Upload failed: %s - %s", resp.Status, string(body)),
		RemotePath: remotePath,
		StatusCode: resp.StatusCode,
	}, nil
}

// getContentType determines the MIME content type based on file extension.
// It supports common file formats including XML, JSON, text files, images,
// PDFs, and archives. Unknown file types default to "application/octet-stream".
//
// Parameters:
//   - filePath: The file path (only the extension is used)
//
// Returns:
//   - string: The MIME type for the file
//
// Supported formats:
//   - Documents: .xml, .json, .txt, .csv, .html, .pdf
//   - Images: .jpg, .jpeg, .png, .gif
//   - Archives: .zip, .tar, .gz
func getContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".xml":
		return "application/xml"
	case ".json":
		return "application/json"
	case ".txt":
		return "text/plain"
	case ".csv":
		return "text/csv"
	case ".html", ".htm":
		return "text/html"
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}
