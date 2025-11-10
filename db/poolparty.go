package db

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Project represents a PoolParty thesaurus/taxonomy project.
// PoolParty is a semantic technology platform for taxonomy and knowledge graph management.
// Each project can contain concepts, terms, and their relationships.
type Project struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URI         string `json:"uri"`
	Created     string `json:"created"`
	Modified    string `json:"modified"`
	Type        string `json:"type"`
	Status      string `json:"status"`
}

// PoolPartyClient represents a client for interacting with the PoolParty API.
// It handles authentication, SPARQL query execution, template management,
// and project operations. The client supports basic authentication and
// maintains a cache of parsed templates for improved performance.
type PoolPartyClient struct {
	BaseURL       string
	Username      string
	Password      string
	HTTPClient    *http.Client
	TemplateDir   string
	templateCache map[string]*template.Template
}

// SPARQLResult represents a SPARQL query result in JSON format.
// It follows the W3C SPARQL 1.1 Query Results JSON Format specification.
type SPARQLResult struct {
	Head    SPARQLHead     `json:"head"`
	Results SPARQLBindings `json:"results"`
}

// SPARQLHead contains metadata about the SPARQL query result,
// including the variable names used in the query.
type SPARQLHead struct {
	Vars []string `json:"vars"`
}

// SPARQLBindings contains the actual result bindings from a SPARQL query.
// Each binding is a map of variable names to their corresponding values.
type SPARQLBindings struct {
	Bindings []map[string]SPARQLValue `json:"bindings"`
}

// SPARQLValue represents a single value in a SPARQL query result.
// It includes the value type (uri, literal, bnode), the value itself,
// and optional language tag for literals.
type SPARQLValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Lang  string `json:"xml:lang,omitempty"`
}

// RDF represents an RDF graph with multiple resource descriptions.
// It is used for parsing RDF/XML responses from PoolParty.
type RDF struct {
	XMLName      xml.Name      `xml:"RDF"`
	Descriptions []Description `xml:"Description"`
}

// Description represents an RDF resource description with various properties.
// It maps to rdf:Description elements in RDF/XML format and includes
// SKOS preferred labels, URIs, restrictions, and custom properties.
type Description struct {
	XMLName   xml.Name  `xml:"Description"`
	About     string    `xml:"about,attr"` // rdf:about attribute
	PrefLabel PrefLabel `xml:"prefLabel"`
	URI4IRI   Resource  `xml:"URI4IRI"`
	ICURI4IRI Resource  `xml:"IC_URI4IRI"`

	Restriction Restriction `xml:"Role-is-restricted-to-Protection-Class"`
	ID          string      `xml:"ID"` // literal value inside <czo:ID>
	UserSkills  *Resource   `xml:"User-has-skills-for-Product"`
}

// PrefLabel represents a SKOS preferred label with language tag.
// Example: <skos:prefLabel xml:lang="en">Computer Science</skos:prefLabel>
type PrefLabel struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}

// Resource represents an RDF resource reference using rdf:resource attribute.
// Example: <czo:URI4IRI rdf:resource="http://example.org/concept/123"/>
type Resource struct {
	Resource string `xml:"resource,attr"`
}

// Restriction represents an RDF restriction with a resource reference.
// Used for OWL restrictions and constraints in the knowledge graph.
type Restriction struct {
	Resource string `xml:"resource,attr"` // rdf:resource
}

// Sparql represents the root element of a SPARQL XML query result.
// It follows the W3C SPARQL Query Results XML Format specification.
type Sparql struct {
	XMLName xml.Name `xml:"http://www.w3.org/2005/sparql-results# sparql"`
	Head    Head     `xml:"head"`
	Results Results  `xml:"results"`
}

// Head contains the variable declarations for a SPARQL XML result.
type Head struct {
	Variables []Variable `xml:"variable"`
}

// Variable represents a SPARQL query variable declaration in XML format.
type Variable struct {
	Name string `xml:"name,attr"`
}

// Results contains all result rows from a SPARQL XML query.
type Results struct {
	Results []Result `xml:"result"`
}

// Result represents a single result row in SPARQL XML format,
// containing multiple variable bindings.
type Result struct {
	Bindings []Binding `xml:"binding"`
}

// Binding represents a variable binding in a SPARQL XML result.
// The value can be either a URI or a literal.
type Binding struct {
	Name    string  `xml:"name,attr"`
	Uri     *string `xml:"uri,omitempty"`
	Literal *string `xml:"literal,omitempty"`
}

// NewPoolPartyClient creates a new PoolParty API client with authentication credentials
// and template directory configuration. The client maintains a template cache and
// uses a 60-second HTTP timeout for all requests.
//
// Parameters:
//   - baseURL: The PoolParty server base URL (e.g., "https://poolparty.example.com")
//   - username: Authentication username
//   - password: Authentication password
//   - templateDir: Directory containing SPARQL query template files
//
// Returns:
//   - *PoolPartyClient: Configured client ready for API operations
//
// Example:
//
//	client := NewPoolPartyClient("https://poolparty.example.com", "admin", "password", "./templates")
func NewPoolPartyClient(baseURL, username, password, templateDir string) *PoolPartyClient {
	return &PoolPartyClient{
		BaseURL:       baseURL,
		Username:      username,
		Password:      password,
		TemplateDir:   templateDir,
		templateCache: make(map[string]*template.Template),
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Follow up to 10 redirects, preserving authentication
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				// Preserve basic auth on redirects
				if len(via) > 0 {
					req.SetBasicAuth(username, password)
				}
				return nil
			},
		},
	}
}

// LoadTemplate loads a SPARQL query template from the template directory and caches it.
// Subsequent calls with the same template name return the cached version for better performance.
// Templates use Go's text/template syntax and can accept parameters for dynamic query generation.
//
// Parameters:
//   - templateName: Name of the template file (e.g., "query.sparql")
//
// Returns:
//   - *template.Template: Parsed template ready for execution
//   - error: Any error encountered during template loading or parsing
//
// Example:
//
//	tmpl, err := client.LoadTemplate("find_concepts.sparql")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *PoolPartyClient) LoadTemplate(templateName string) (*template.Template, error) {
	// Check cache first
	if tmpl, ok := c.templateCache[templateName]; ok {
		return tmpl, nil
	}

	// Build full path
	templatePath := filepath.Join(c.TemplateDir, templateName)

	// Read template file
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("error reading template %s: %w", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New(templateName).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("error parsing template %s: %w", templateName, err)
	}

	// Cache it
	c.templateCache[templateName] = tmpl

	return tmpl, nil
}

// ExecuteSPARQLFromTemplate loads a SPARQL query template, executes it with the given parameters,
// and runs the resulting query against the specified PoolParty project.
// This is the recommended way to execute parameterized SPARQL queries.
//
// Parameters:
//   - projectID: The PoolParty project/thesaurus ID
//   - templateName: Name of the template file (e.g., "query.sparql")
//   - contentType: Desired response format ("application/json", "application/rdf+xml", etc.)
//   - params: Parameters to pass to the template (can be any Go type)
//
// Returns:
//   - []byte: Query results in the requested format
//   - error: Any error encountered during template execution or query
//
// Example:
//
//	params := map[string]string{"concept": "http://example.org/concept/123"}
//	results, err := client.ExecuteSPARQLFromTemplate("myproject", "get_related.sparql", "application/json", params)
func (c *PoolPartyClient) ExecuteSPARQLFromTemplate(projectID, templateName, contentType string, params interface{}) ([]byte, error) {
	// Load template
	tmpl, err := c.LoadTemplate(templateName)
	if err != nil {
		return nil, err
	}

	// Execute template
	var queryBuf bytes.Buffer
	err = tmpl.Execute(&queryBuf, params)
	if err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	query := queryBuf.String()

	// Execute the generated query
	return c.ExecuteSPARQL(projectID, query, contentType)
}

// ExecuteSPARQL executes a raw SPARQL query against a PoolParty project's SPARQL endpoint.
// The query is sent as form-encoded POST data with the specified Accept header for format negotiation.
//
// Parameters:
//   - projectID: The PoolParty project/thesaurus ID
//   - query: The SPARQL query string
//   - contentType: Desired response format ("application/json", "application/sparql-results+xml", "application/rdf+xml", etc.)
//
// Returns:
//   - []byte: Query results in the requested format
//   - error: Any error encountered during query execution
//
// Example:
//
//	query := `SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10`
//	results, err := client.ExecuteSPARQL("myproject", query, "application/json")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *PoolPartyClient) ExecuteSPARQL(projectID, query, contentType string) ([]byte, error) {
	// PoolParty SPARQL endpoint
	endpoint := fmt.Sprintf("%s/PoolParty/sparql/%s", c.BaseURL, projectID)
	// Create form data with query and content-type parameter
	data := url.Values{}
	data.Set("query", query)
	fmt.Println(endpoint, contentType)
	// fmt.Println(query)
	// Create request
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set authentication
	req.SetBasicAuth(c.Username, c.Password)

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept", contentType)

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// RunSparQLFromFile is a convenience function that creates a PoolParty client and
// executes a SPARQL query from a template file in a single call. This is useful
// for one-off queries where you don't need to maintain a persistent client.
//
// Parameters:
//   - baseURL: The PoolParty server base URL
//   - username: Authentication username
//   - password: Authentication password
//   - projectID: The PoolParty project/thesaurus ID
//   - templateDir: Directory containing template files
//   - tmplFileName: Name of the template file
//   - contentType: Desired response format
//   - params: Parameters to pass to the template
//
// Returns:
//   - []byte: Query results in the requested format
//   - error: Any error encountered during execution
//
// Example:
//
//	params := map[string]string{"limit": "100"}
//	results, err := RunSparQLFromFile("https://poolparty.example.com", "admin", "pass",
//	    "myproject", "./templates", "concepts.sparql", "application/json", params)
func RunSparQLFromFile(baseURL, username, password, projectID, templateDir, tmplFileName, contentType string, params interface{}) ([]byte, error) {
	// Create client
	client := NewPoolPartyClient(baseURL, username, password, templateDir)
	fmt.Println("\n\nExecuting SPARQL queries from templates...")
	return client.ExecuteSPARQLFromTemplate(projectID, tmplFileName, contentType, params)
}

// ListProjects retrieves all projects (thesauri) from the PoolParty server.
// This method tries multiple endpoint and content-type combinations to handle
// different PoolParty versions and configurations. It automatically discovers
// the working endpoint by trying various API paths and accept headers.
//
// Returns:
//   - []Project: List of projects found on the server
//   - error: Error if all endpoint attempts fail
//
// Example:
//
//	projects, err := client.ListProjects()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, proj := range projects {
//	    fmt.Printf("Project: %s (%s)\n", proj.Title, proj.ID)
//	}
func (c *PoolPartyClient) ListProjects() ([]Project, error) {
	// Try different endpoint and header combinations
	endpoints := []string{
		"/PoolParty/api/projects",
		"/api/projects",
		"/PoolParty/api/thesauri",
		"/api/thesauri",
		"/PoolParty/sparql",
	}

	acceptHeaders := []string{
		"application/json",
		"application/ld+json",
		"application/rdf+xml",
		"text/turtle",
		"*/*",
	}

	for _, endpoint := range endpoints {
		for _, accept := range acceptHeaders {
			url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				continue
			}

			req.SetBasicAuth(c.Username, c.Password)
			req.Header.Set("Accept", accept)

			resp, err := c.HTTPClient.Do(req)
			if err != nil {
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}

			if resp.StatusCode == http.StatusOK {
				// Try to parse as JSON
				var projects []Project
				err = json.Unmarshal(body, &projects)
				if err == nil && len(projects) > 0 {
					fmt.Printf("✅ Success with: %s (Accept: %s)\n", url, accept)
					return projects, nil
				}

				// Maybe it's a different structure
				var result map[string]interface{}
				err = json.Unmarshal(body, &result)
				if err == nil {
					fmt.Printf("✅ Got response from: %s (Accept: %s)\n", url, accept)
					fmt.Printf("Response structure: %+v\n", result)
					return nil, fmt.Errorf("successful request but unexpected response structure")
				}
			}
		}
	}

	return nil, fmt.Errorf("all endpoint attempts failed")
}

// GetProjectDetails retrieves detailed information about a specific PoolParty project
// by its ID. This includes metadata such as title, description, URI, creation date,
// modification date, type, and status.
//
// Parameters:
//   - projectID: The project/thesaurus identifier
//
// Returns:
//   - *Project: Project details
//   - error: Any error encountered during retrieval
//
// Example:
//
//	project, err := client.GetProjectDetails("myproject")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Project: %s\nDescription: %s\n", project.Title, project.Description)
func (c *PoolPartyClient) GetProjectDetails(projectID string) (*Project, error) {
	url := fmt.Sprintf("%s/PoolParty/api/thesauri/%s", c.BaseURL, projectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var project Project
	err = json.Unmarshal(body, &project)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &project, nil
}

// PrintProjects prints a list of PoolParty projects in a human-readable format.
// Each project is displayed with its ID, title, description, URI, type, status,
// creation date, and modification date. This is useful for debugging and CLI output.
//
// Parameters:
//   - projects: List of projects to print
//
// Example:
//
//	projects, _ := client.ListProjects()
//	PrintProjects(projects)
func PrintProjects(projects []Project) {
	fmt.Printf("\n========================================\n")
	fmt.Printf("Total Projects: %d\n", len(projects))
	fmt.Printf("========================================\n\n")

	for i, project := range projects {
		fmt.Printf("Project #%d\n", i+1)
		fmt.Printf("  ID:          %s\n", project.ID)
		fmt.Printf("  Title:       %s\n", project.Title)
		fmt.Printf("  Description: %s\n", project.Description)
		fmt.Printf("  URI:         %s\n", project.URI)
		fmt.Printf("  Type:        %s\n", project.Type)
		fmt.Printf("  Status:      %s\n", project.Status)
		fmt.Printf("  Created:     %s\n", project.Created)
		fmt.Printf("  Modified:    %s\n", project.Modified)
		fmt.Printf("\n")
	}
}

// PoolPartyProjects is a convenience function that creates a PoolParty client,
// fetches all projects, and prints them with troubleshooting tips if the operation fails.
// This is useful for quick CLI tools and debugging connection issues.
//
// Parameters:
//   - baseURL: The PoolParty server base URL
//   - username: Authentication username
//   - password: Authentication password
//   - templateDir: Directory for SPARQL query templates (can be empty string)
//
// Example:
//
//	PoolPartyProjects("https://poolparty.example.com", "admin", "password", "./templates")
func PoolPartyProjects(baseURL, username, password, templateDir string) {
	// Create client
	client := NewPoolPartyClient(baseURL, username, password, templateDir)

	fmt.Println("Attempting to fetch projects...")

	// Try to list projects with auto-discovery
	projects, err := client.ListProjects()
	if err != nil {
		log.Printf("Error fetching projects: %v", err)

		fmt.Println("\n========================================")
		fmt.Println("TROUBLESHOOTING TIPS:")
		fmt.Println("========================================")
		fmt.Println("1. Check your PoolParty version")
		fmt.Println("2. Verify the base URL is correct")
		fmt.Println("3. Ensure your credentials have API access")
		fmt.Println("4. Look at the successful responses above")
		fmt.Println("5. Check PoolParty documentation for your version")
		return
	}

	// Print projects
	PrintProjects(projects)
}
