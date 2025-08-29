package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"unicode/utf8"
)

type sparqlValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type sparqlResult struct {
	Bindings []map[string]sparqlValue `json:"bindings"`
}

type sparqlResponse struct {
	Head    map[string][]string `json:"head"`
	Results sparqlResult        `json:"results"`
}

type Repository struct {
	ID    string
	Title string
	Type  string
}

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
	req.Header.Set("Accept", "application/sparql-results+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list repositories. Status: %s, Body: %s", resp.Status, string(body))
	}

	var data sparqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var repos []Repository
	for _, binding := range data.Results.Bindings {
		repos = append(repos, Repository{
			ID:    binding["id"].Value,
			Title: binding["title"].Value,
			Type:  binding["type"].Value,
		})
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

// CreateRepository creates an in-memory RDF4J repository.
func CreateRepository(serverURL, repositoryID, username, password string) error {
	client := &http.Client{}

	repoConfigTurtle := fmt.Sprintf(`
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#>.
@prefix rep: <http://www.openrdf.org/config/repository#>.
@prefix sr: <http://www.openrdf.org/config/repository/sail#>.
@prefix sail: <http://www.openrdf.org/config/sail#>.
@prefix mem: <http://www.openrdf.org/config/sail/memory#>.

[] a rep:Repository ;
   rep:repositoryID "%s" ;
   rdfs:label "Memory Store for %s" ;
   rep:repositoryImpl [
      rep:repositoryType "openrdf:SailRepository" ;
      sr:sailImpl [
         sail:sailType "openrdf:MemoryStore"
      ]
   ].`, repositoryID, repositoryID)

	url := fmt.Sprintf("%s/repositories/%s", serverURL, repositoryID)

	req, err := http.NewRequest("PUT", url, bytes.NewBufferString(repoConfigTurtle))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/turtle")
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to create repository. Status: %d , Body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func CreateLMDBRepository(serverURL, repositoryID, username, password string) error {
	// Turtle configuration for LMDB repo
	config := fmt.Sprintf(`
		@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
		@prefix rep: <http://www.openrdf.org/config/repository#> .
		@prefix sr: <http://www.openrdf.org/config/repository/sail#> .
		@prefix sail: <http://www.openrdf.org/config/sail#> .
		@prefix lmdb: <http://rdf4j.org/config/sail/lmdb#> .

		[] a rep:Repository ;
		   rep:repositoryID "%s" ;
		   rdfs:label "LMDB store repo" ;
		   rep:repositoryImpl [
		       rep:repositoryType "openrdf:SailRepository" ;
		       sr:sailImpl [
		           sail:sailType "rdf4j:LMDBStore" ;
		           lmdb:tripleIndexes "spoc,posc,cosp" ;
		           lmdb:forceSync "false" ;
		           lmdb:path "data/lmdb"
		       ]
		   ] .
	`, repositoryID)

	client := &http.Client{}
	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/repositories/%s", serverURL, repositoryID),
		bytes.NewReader([]byte(config)),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "text/turtle")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to create LMDB repository. Status: %s, Body: %s", resp.Status, string(body))
	}

	return nil
}
