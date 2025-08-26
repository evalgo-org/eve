package db

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"unicode/utf8"
)

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

func ExportRDFXml(serverURL, repositoryID, username, password, outputFilePath string) {
	// Create an HTTP client and request
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/repositories/%s/statements", serverURL, repositoryID),
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to create HTTP request: %v", err)
	}

	// Set the accept header to request RDF/XML format
	req.Header.Set("Accept", "application/rdf+xml")

	// Set basic authentication
	req.SetBasicAuth(username, password)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("Failed to export data. Status: %s, Body: %s", resp.Status, body)
	}

	// Read the response body (RDF/XML data)
	rdfData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	// Save the RDF/XML data to a file
	err = ioutil.WriteFile(outputFilePath, rdfData, 0644)
	if err != nil {
		log.Fatalf("Failed to write RDF data to file: %v", err)
	}

	fmt.Printf("RDF data exported successfully to %s\n", outputFilePath)
}
