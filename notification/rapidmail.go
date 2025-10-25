// Package notification provides utilities for creating and sending email notifications
// through the RapidMail API. It includes functions for packaging HTML content into ZIP archives
// and sending email campaigns to recipients.
//
// Features:
//   - Create ZIP archives from HTML files for email content
//   - Send email campaigns through RapidMail API
//   - Support for scheduled email delivery
//   - Base64 encoding of email content
//   - Recipient list management
package notification

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	eve "eve.evalgo.org/common"
)

// createZipFromHTML creates a ZIP archive containing an HTML file.
// This function is used to package HTML email content for RapidMail.
//
// Parameters:
//   - htmlPath: Path to the HTML file to include in the ZIP
//   - zipPath: Path where the ZIP file should be created
//
// Returns:
//   - error: If any step in the process fails (file operations, ZIP creation)
//
// The function:
//  1. Opens the HTML file
//  2. Creates a new ZIP file
//  3. Adds the HTML file to the ZIP archive
//  4. Closes all files and writers
func createZipFromHTML(htmlPath, zipPath string) error {
	htmlFile, err := os.Open(htmlPath)
	if err != nil {
		return err
	}
	defer htmlFile.Close()

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	htmlFileName := filepath.Base(htmlPath)
	writer, err := zipWriter.Create(htmlFileName)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, htmlFile)
	return err
}

// RapidMailCreateContentFromFile prepares HTML email content for RapidMail.
// This function creates a ZIP archive from an HTML file and encodes it in base64,
// which is the format required by the RapidMail API.
//
// Parameters:
//   - htmlFilePath: Path to the HTML file containing the email content
//
// Returns:
//   - string: Base64-encoded ZIP content
//   - error: If any step in the process fails
//
// The function:
//  1. Creates a ZIP file containing the HTML file
//  2. Reads the ZIP file content
//  3. Encodes the content in base64
//  4. Returns the encoded string
func RapidMailCreateContentFromFile(htmlFilePath string) (string, error) {
	zipFileName := "newsletter.zip" // temporary ZIP filename

	// Step 1: Create ZIP file containing the HTML file
	err := createZipFromHTML(htmlFilePath, zipFileName)
	if err != nil {
		return "", err
	}

	// Read ZIP file
	zipData, err := ioutil.ReadFile(zipFileName)
	if err != nil {
		return "", err
	}

	// Base64 encode the ZIP file content
	return base64.StdEncoding.EncodeToString(zipData), nil
}

// RapidMailSend sends an email campaign through the RapidMail API.
// This function creates and schedules an email campaign with the provided content
// and recipient list.
//
// Parameters:
//   - apiUser: RapidMail API username
//   - apiPass: RapidMail API password
//   - subject: Email subject line
//   - recipients: List of recipient maps (each should contain "email" and optionally other fields)
//   - htmlFilePath: Path to the HTML file containing the email content
//
// Returns:
//   - error: If any step in the process fails
//
// The function:
//  1. Prepares the email content by creating a ZIP and encoding it
//  2. Creates a payload with campaign details and content
//  3. Makes an authenticated POST request to the RapidMail API
//  4. Logs the API response
//
// Example recipient format:
//
//	[]map[string]interface{}{
//	    {"email": "recipient1@example.com", "firstname": "John", "lastname": "Doe"},
//	    {"email": "recipient2@example.com", "firstname": "Jane", "lastname": "Smith"},
//	}
func RapidMailSend(apiUser, apiPass, subject string, recipients []map[string]interface{}, htmlFilePath string) error {
	// Prepare the email content
	content, err := RapidMailCreateContentFromFile(htmlFilePath)
	if err != nil {
		return err
	}

	// Create the content map expected by RapidMail
	zippedContent := map[string]string{
		"content": content,
		"type":    "application/zip",
	}

	// Prepare the API payload
	apiURL := "https://apiv3.emailsys.net/mailings"
	payload := map[string]interface{}{
		"status":         "scheduled", // "draft" = just create it, "scheduled" = run it
		"destinations":   recipients,  // List of recipient maps
		"from_name":      "Francisc Simon",
		"from_email":     "francisc@simon.services",
		"subject":        subject,
		"send_at":        "2025-08-09 21:25:00",      // Scheduled send time
		"check_ecg":      "no",                       // Skip ECG check
		"check_robinson": "no",                       // Skip Robinson list check
		"host":           "tf22c2f8a.emailsys1a.net", // RapidMail host
		"file":           zippedContent,              // Base64-encoded ZIP content
	}

	// Marshal the payload to JSON
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Create and send the HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(apiUser, apiPass)

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read and log the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	eve.Logger.Info(string(respBody))
	return nil
}
