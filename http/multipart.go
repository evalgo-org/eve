package http

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// buildMultipartBody builds a multipart/form-data request body
func buildMultipartBody(req *Request) (io.Reader, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add form fields
	for key, value := range req.FormData {
		if err := writer.WriteField(key, value); err != nil {
			return nil, "", fmt.Errorf("failed to write form field %s: %w", key, err)
		}
	}

	// Add files
	for fieldName, filePath := range req.Files {
		if err := addFileToMultipart(writer, fieldName, filePath); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &body, writer.FormDataContentType(), nil
}

// addFileToMultipart adds a single file to the multipart writer
func addFileToMultipart(writer *multipart.Writer, fieldName, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	// Create form file part
	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file for %s: %w", filePath, err)
	}

	// Copy file content
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file %s: %w", filePath, err)
	}

	return nil
}
