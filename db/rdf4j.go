package db

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"unicode/utf8"
	"encoding/json"
)

// Repository represents a single RDF4J repository
type Repository struct {
	ID   string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// ListRepositories fetches the list of repositories from RDF4J
func ListRepositories(serverURL, username, password string) ([]Repository, error) {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/repositories", serverURL),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list repositories. Status: %s, Body: %s", resp.Status, string(body))
	}

	var repos []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return repos, nil
}

func stripBOM(data []byte) []byte {
	// UTF-8 BOM is 0xEF,0xBB,0xBF
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

func ImportRDF(serverURL, repositoryID, username, password, rdfFilePath, contentType string) ([]byte, error) {

	// Read the RDF/XML file
	rdfData, err := os.ReadFile(rdfFilePath)
	if err != nil {
		return nil, err
	}

	// Strip BOM if present
	rdfData = stripBOM(rdfData)

	// Check if the file is valid UTF-8
	if !utf8.Valid(rdfData) {
		return nil, errors.New("invalid UTF-8 file")
	}

	// Create an HTTP client and request
	client := &http.Client{}
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/repositories/%s/statements", serverURL, repositoryID),
		bytes.NewReader(rdfData),
	)
	if err != nil {
		return nil, err
	}

	// Set the content type header for RDF/XML
	// req.Header.Set("Content-Type", "application/rdf+xml")
	req.Header.Set("Content-Type", contentType)

	// Set basic authentication
	req.SetBasicAuth(username, password)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func ExportRDFXml(serverURL, repositoryID, username, password, outputFilePath, contentType string) error {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/repositories/%s/statements", serverURL, repositoryID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Accept", contentType)
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to export data. Status: %s, Body: %s", resp.Status, string(body))
	}

	rdfData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := ioutil.WriteFile(outputFilePath, rdfData, 0644); err != nil {
		return fmt.Errorf("failed to write RDF data to file: %w", err)
	}

	return nil
}

// DeleteRepository deletes a repository from an RDF4J server.
func DeleteRepository(serverURL, repositoryID, username, password string) error {
	client := &http.Client{}
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/repositories/%s", serverURL, repositoryID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete repository. Status: %s, Body: %s", resp.Status, string(body))
	}

	return nil
}
