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

func RapidMailCreateContentFromFile(htmlFilePath string) (string, error) {
	zipFileName := "newsletter.zip" // temporary ZIP filename
	// Step 1: Create ZIP file containing the HTML file
	err := createZipFromHTML(htmlFilePath, zipFileName)
	if err != nil {
		return "", err
	}
	zipFilePath := "newsletter.zip" // Path to your ZIP file
	// Read ZIP file
	zipData, err := ioutil.ReadFile(zipFilePath)
	if err != nil {
		return "", err
	}
	// Base64 encode the ZIP file content
	return base64.StdEncoding.EncodeToString(zipData), nil
}

func RapidMailSend(apiUser, apiPass, subject string, recipients []map[string]interface{}, htmlFilePath string) error {
	content, err := RapidMailCreateContentFromFile(htmlFilePath)
	if err != nil {
		return err
	}
	zippedContent := map[string]string{"content": content, "type": "application/zip"}
	apiURL := "https://apiv3.emailsys.net/mailings"
	payload := map[string]interface{}{
		"status":         "scheduled", // draft = just create it, scheduled = run it
		"destinations":   recipients,
		"from_name":      "Francisc Simon",
		"from_email":     "francisc@simon.services",
		"subject":        subject,
		"send_at":        "2025-08-09 21:25:00",
		"check_ecg":      "no",
		"check_robinson": "no",
		"host":           "tf22c2f8a.emailsys1a.net",
		"file":           zippedContent,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(apiUser, apiPass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	eve.Logger.Info(string(respBody))
	return nil
}
