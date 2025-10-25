package db

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProject tests Project struct JSON marshaling
func TestProject(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		project := Project{
			ID:          "test-project-123",
			Title:       "Test Taxonomy",
			Description: "A test taxonomy for unit testing",
			URI:         "http://example.org/taxonomy/test",
			Created:     "2024-01-01T10:00:00Z",
			Modified:    "2024-01-15T14:30:00Z",
			Type:        "thesaurus",
			Status:      "active",
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(project)
		require.NoError(t, err)
		assert.Contains(t, string(jsonData), "test-project-123")
		assert.Contains(t, string(jsonData), "Test Taxonomy")

		// Unmarshal back
		var decoded Project
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		assert.Equal(t, project.ID, decoded.ID)
		assert.Equal(t, project.Title, decoded.Title)
		assert.Equal(t, project.Description, decoded.Description)
	})

	t.Run("empty project", func(t *testing.T) {
		project := Project{}
		jsonData, err := json.Marshal(project)
		require.NoError(t, err)

		var decoded Project
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "", decoded.ID)
		assert.Equal(t, "", decoded.Title)
	})
}

// TestSPARQLResult tests SPARQL JSON result parsing
func TestSPARQLResult(t *testing.T) {
	t.Run("parse valid SPARQL JSON result", func(t *testing.T) {
		jsonData := `{
			"head": {
				"vars": ["subject", "predicate", "object"]
			},
			"results": {
				"bindings": [
					{
						"subject": {"type": "uri", "value": "http://example.org/concept/1"},
						"predicate": {"type": "uri", "value": "http://www.w3.org/2004/02/skos/core#prefLabel"},
						"object": {"type": "literal", "value": "Computer Science", "xml:lang": "en"}
					}
				]
			}
		}`

		var result SPARQLResult
		err := json.Unmarshal([]byte(jsonData), &result)
		require.NoError(t, err)

		assert.Equal(t, 3, len(result.Head.Vars))
		assert.Contains(t, result.Head.Vars, "subject")
		assert.Contains(t, result.Head.Vars, "predicate")
		assert.Contains(t, result.Head.Vars, "object")

		assert.Equal(t, 1, len(result.Results.Bindings))
		binding := result.Results.Bindings[0]

		assert.Equal(t, "uri", binding["subject"].Type)
		assert.Equal(t, "http://example.org/concept/1", binding["subject"].Value)

		assert.Equal(t, "literal", binding["object"].Type)
		assert.Equal(t, "Computer Science", binding["object"].Value)
		assert.Equal(t, "en", binding["object"].Lang)
	})

	t.Run("empty SPARQL result", func(t *testing.T) {
		jsonData := `{
			"head": {"vars": []},
			"results": {"bindings": []}
		}`

		var result SPARQLResult
		err := json.Unmarshal([]byte(jsonData), &result)
		require.NoError(t, err)

		assert.Equal(t, 0, len(result.Head.Vars))
		assert.Equal(t, 0, len(result.Results.Bindings))
	})
}

// TestSPARQLValue tests SPARQL value struct
func TestSPARQLValue(t *testing.T) {
	tests := []struct {
		name     string
		value    SPARQLValue
		jsonStr  string
	}{
		{
			name:    "URI value",
			value:   SPARQLValue{Type: "uri", Value: "http://example.org/concept/123"},
			jsonStr: `{"type":"uri","value":"http://example.org/concept/123"}`,
		},
		{
			name:    "Literal without language",
			value:   SPARQLValue{Type: "literal", Value: "Test Value"},
			jsonStr: `{"type":"literal","value":"Test Value"}`,
		},
		{
			name:    "Literal with language",
			value:   SPARQLValue{Type: "literal", Value: "Hello", Lang: "en"},
			jsonStr: `{"type":"literal","value":"Hello","xml:lang":"en"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.value)
			require.NoError(t, err)
			assert.JSONEq(t, tt.jsonStr, string(jsonData))
		})
	}
}

// TestRDF tests RDF XML parsing
func TestRDF(t *testing.T) {
	t.Run("parse RDF XML", func(t *testing.T) {
		xmlData := `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:skos="http://www.w3.org/2004/02/skos/core#">
  <rdf:Description rdf:about="http://example.org/concept/1">
    <skos:prefLabel xml:lang="en">Computer Science</skos:prefLabel>
  </rdf:Description>
</rdf:RDF>`

		var rdf RDF
		err := xml.Unmarshal([]byte(xmlData), &rdf)
		require.NoError(t, err)

		assert.Equal(t, 1, len(rdf.Descriptions))
		desc := rdf.Descriptions[0]
		assert.Equal(t, "http://example.org/concept/1", desc.About)
		assert.Equal(t, "Computer Science", desc.PrefLabel.Value)
		assert.Equal(t, "en", desc.PrefLabel.Lang)
	})
}

// TestPrefLabel tests SKOS prefLabel parsing
func TestPrefLabel(t *testing.T) {
	xmlData := `<prefLabel xml:lang="de">Informatik</prefLabel>`

	var label PrefLabel
	err := xml.Unmarshal([]byte(xmlData), &label)
	require.NoError(t, err)

	assert.Equal(t, "de", label.Lang)
	assert.Equal(t, "Informatik", label.Value)
}

// TestResource tests RDF resource references
func TestResource(t *testing.T) {
	xmlData := `<URI4IRI rdf:resource="http://example.org/resource/123" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"/>`

	var resource Resource
	err := xml.Unmarshal([]byte(xmlData), &resource)
	require.NoError(t, err)

	assert.Equal(t, "http://example.org/resource/123", resource.Resource)
}

// TestSparql_XML tests SPARQL XML result parsing
func TestSparql_XML(t *testing.T) {
	t.Run("parse SPARQL XML result", func(t *testing.T) {
		xmlData := `<?xml version="1.0"?>
<sparql xmlns="http://www.w3.org/2005/sparql-results#">
  <head>
    <variable name="x"/>
    <variable name="y"/>
  </head>
  <results>
    <result>
      <binding name="x">
        <uri>http://example.org/1</uri>
      </binding>
      <binding name="y">
        <literal>Test</literal>
      </binding>
    </result>
  </results>
</sparql>`

		var result Sparql
		err := xml.Unmarshal([]byte(xmlData), &result)
		require.NoError(t, err)

		assert.Equal(t, 2, len(result.Head.Variables))
		assert.Equal(t, "x", result.Head.Variables[0].Name)
		assert.Equal(t, "y", result.Head.Variables[1].Name)

		assert.Equal(t, 1, len(result.Results.Results))
		assert.Equal(t, 2, len(result.Results.Results[0].Bindings))
	})
}

// TestNewPoolPartyClient tests client construction
func TestNewPoolPartyClient(t *testing.T) {
	client := NewPoolPartyClient(
		"https://poolparty.example.com",
		"admin",
		"password",
		"/tmp/templates",
	)

	assert.NotNil(t, client)
	assert.Equal(t, "https://poolparty.example.com", client.BaseURL)
	assert.Equal(t, "admin", client.Username)
	assert.Equal(t, "password", client.Password)
	assert.Equal(t, "/tmp/templates", client.TemplateDir)
	assert.NotNil(t, client.HTTPClient)
	assert.NotNil(t, client.templateCache)
	assert.Equal(t, 0, len(client.templateCache))
}

// TestLoadTemplate tests template loading and caching
func TestLoadTemplate(t *testing.T) {
	t.Run("load and cache template", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "test.sparql")
		templateContent := "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT {{.Limit}}"
		err := ioutil.WriteFile(templateFile, []byte(templateContent), 0644)
		require.NoError(t, err)

		client := NewPoolPartyClient("http://localhost", "user", "pass", tmpDir)

		// Load template first time
		tmpl, err := client.LoadTemplate("test.sparql")
		require.NoError(t, err)
		assert.NotNil(t, tmpl)

		// Verify template works
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, map[string]int{"Limit": 100})
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "LIMIT 100")

		// Load template second time (should come from cache)
		tmpl2, err := client.LoadTemplate("test.sparql")
		require.NoError(t, err)
		assert.Equal(t, tmpl, tmpl2) // Should be same instance
	})

	t.Run("template file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		client := NewPoolPartyClient("http://localhost", "user", "pass", tmpDir)

		_, err := client.LoadTemplate("nonexistent.sparql")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading template")
	})

	t.Run("invalid template syntax", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "invalid.sparql")
		invalidContent := "SELECT {{.Invalid"
		err := ioutil.WriteFile(templateFile, []byte(invalidContent), 0644)
		require.NoError(t, err)

		client := NewPoolPartyClient("http://localhost", "user", "pass", tmpDir)

		_, err = client.LoadTemplate("invalid.sparql")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error parsing template")
	})
}

// TestExecuteSPARQL tests SPARQL query execution
func TestExecuteSPARQL(t *testing.T) {
	t.Run("successful SPARQL query", func(t *testing.T) {
		expectedResult := `{"head":{"vars":["s"]},"results":{"bindings":[]}}`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "/PoolParty/sparql/testproject")
			assert.Equal(t, "application/x-www-form-urlencoded; charset=UTF-8", r.Header.Get("Content-Type"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			username, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "admin", username)
			assert.Equal(t, "password", password)

			body, _ := ioutil.ReadAll(r.Body)
			assert.Contains(t, string(body), "query=")

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedResult))
		}))
		defer server.Close()

		client := NewPoolPartyClient(server.URL, "admin", "password", "/tmp")

		query := "SELECT ?s WHERE { ?s ?p ?o } LIMIT 10"
		result, err := client.ExecuteSPARQL("testproject", query, "application/json")
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, string(result))
	})

	t.Run("SPARQL query with error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid SPARQL syntax"))
		}))
		defer server.Close()

		client := NewPoolPartyClient(server.URL, "admin", "password", "/tmp")

		query := "INVALID QUERY"
		result, err := client.ExecuteSPARQL("testproject", query, "application/json")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("SPARQL query returns RDF/XML", func(t *testing.T) {
		rdfResult := `<?xml version="1.0"?><rdf:RDF></rdf:RDF>`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/rdf+xml", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(rdfResult))
		}))
		defer server.Close()

		client := NewPoolPartyClient(server.URL, "admin", "password", "/tmp")

		query := "CONSTRUCT { ?s ?p ?o } WHERE { ?s ?p ?o } LIMIT 10"
		result, err := client.ExecuteSPARQL("testproject", query, "application/rdf+xml")
		assert.NoError(t, err)
		assert.Contains(t, string(result), "rdf:RDF")
	})
}

// TestExecuteSPARQLFromTemplate tests templated query execution
func TestExecuteSPARQLFromTemplate(t *testing.T) {
	t.Run("execute query from template", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "concepts.sparql")
		templateContent := "SELECT ?concept WHERE { ?concept a skos:Concept } LIMIT {{.Limit}}"
		err := ioutil.WriteFile(templateFile, []byte(templateContent), 0644)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseForm()
			require.NoError(t, err)

			query := r.FormValue("query")
			assert.Contains(t, query, "LIMIT 50")

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results":[]}`))
		}))
		defer server.Close()

		client := NewPoolPartyClient(server.URL, "admin", "password", tmpDir)

		params := map[string]int{"Limit": 50}
		result, err := client.ExecuteSPARQLFromTemplate("testproject", "concepts.sparql", "application/json", params)
		assert.NoError(t, err)
		assert.Contains(t, string(result), "results")
	})

	t.Run("template file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		client := NewPoolPartyClient("http://localhost", "admin", "password", tmpDir)

		// Try to load a non-existent template
		result, err := client.ExecuteSPARQLFromTemplate("testproject", "nonexistent.sparql", "application/json", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error reading template")
	})
}

// TestListProjects tests project listing with endpoint discovery
func TestListProjects(t *testing.T) {
	t.Run("successful project listing", func(t *testing.T) {
		projects := []Project{
			{ID: "proj1", Title: "Project 1", Type: "thesaurus", Status: "active"},
			{ID: "proj2", Title: "Project 2", Type: "taxonomy", Status: "active"},
		}
		projectsJSON, _ := json.Marshal(projects)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/PoolParty/api/projects" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(projectsJSON)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		client := NewPoolPartyClient(server.URL, "admin", "password", "/tmp")

		result, err := client.ListProjects()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(result))
		assert.Equal(t, "proj1", result[0].ID)
		assert.Equal(t, "Project 1", result[0].Title)
	})

	t.Run("all endpoints fail", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewPoolPartyClient(server.URL, "admin", "password", "/tmp")

		result, err := client.ListProjects()
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "all endpoint attempts failed")
	})
}

// TestGetProjectDetails tests fetching specific project details
func TestGetProjectDetails(t *testing.T) {
	t.Run("successful project details retrieval", func(t *testing.T) {
		project := Project{
			ID:          "myproject",
			Title:       "My Project",
			Description: "A test project",
			URI:         "http://example.org/project",
			Type:        "thesaurus",
			Status:      "active",
			Created:     "2024-01-01T00:00:00Z",
			Modified:    "2024-01-15T00:00:00Z",
		}
		projectJSON, _ := json.Marshal(project)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Contains(t, r.URL.Path, "/PoolParty/api/thesauri/myproject")
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			w.WriteHeader(http.StatusOK)
			w.Write(projectJSON)
		}))
		defer server.Close()

		client := NewPoolPartyClient(server.URL, "admin", "password", "/tmp")

		result, err := client.GetProjectDetails("myproject")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "myproject", result.ID)
		assert.Equal(t, "My Project", result.Title)
		assert.Equal(t, "A test project", result.Description)
	})

	t.Run("project not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Project not found"))
		}))
		defer server.Close()

		client := NewPoolPartyClient(server.URL, "admin", "password", "/tmp")

		result, err := client.GetProjectDetails("nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "404")
	})
}

// TestPrintProjects tests project printing
func TestPrintProjects(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	projects := []Project{
		{ID: "proj1", Title: "Project 1", Description: "First project", Type: "thesaurus", Status: "active"},
		{ID: "proj2", Title: "Project 2", Description: "Second project", Type: "taxonomy", Status: "inactive"},
	}

	PrintProjects(projects)

	w.Close()
	output, _ := ioutil.ReadAll(r)
	os.Stdout = oldStdout

	outputStr := string(output)
	assert.Contains(t, outputStr, "Total Projects: 2")
	assert.Contains(t, outputStr, "proj1")
	assert.Contains(t, outputStr, "Project 1")
	assert.Contains(t, outputStr, "First project")
	assert.Contains(t, outputStr, "proj2")
	assert.Contains(t, outputStr, "Project 2")
}

// TestRunSparQLFromFile tests convenience function
func TestRunSparQLFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "query.sparql")
	templateContent := "SELECT * WHERE { ?s ?p ?o } LIMIT {{.Limit}}"
	err := ioutil.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		require.NoError(t, err)

		query := r.FormValue("query")
		assert.Contains(t, query, "LIMIT 100")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	params := map[string]int{"Limit": 100}
	result, err := RunSparQLFromFile(
		server.URL,
		"admin",
		"password",
		"testproject",
		tmpDir,
		"query.sparql",
		"application/json",
		params,
	)

	assert.NoError(t, err)
	assert.Contains(t, string(result), "results")
}

// TestBinding tests SPARQL XML binding parsing
func TestBinding(t *testing.T) {
	t.Run("URI binding", func(t *testing.T) {
		xmlData := `<binding name="subject">
			<uri>http://example.org/concept/1</uri>
		</binding>`

		var binding Binding
		err := xml.Unmarshal([]byte(xmlData), &binding)
		require.NoError(t, err)

		assert.Equal(t, "subject", binding.Name)
		assert.NotNil(t, binding.Uri)
		assert.Equal(t, "http://example.org/concept/1", *binding.Uri)
		assert.Nil(t, binding.Literal)
	})

	t.Run("Literal binding", func(t *testing.T) {
		xmlData := `<binding name="label">
			<literal>Test Label</literal>
		</binding>`

		var binding Binding
		err := xml.Unmarshal([]byte(xmlData), &binding)
		require.NoError(t, err)

		assert.Equal(t, "label", binding.Name)
		assert.NotNil(t, binding.Literal)
		assert.Equal(t, "Test Label", *binding.Literal)
		assert.Nil(t, binding.Uri)
	})
}

// TestPoolPartyClient_TemplateCache tests template caching behavior
func TestPoolPartyClient_TemplateCache(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewPoolPartyClient("http://localhost", "user", "pass", tmpDir)

	// Create multiple template files
	templates := []string{"query1.sparql", "query2.sparql", "query3.sparql"}
	for _, tmplName := range templates {
		content := "SELECT ?s WHERE { ?s ?p ?o }"
		err := ioutil.WriteFile(filepath.Join(tmpDir, tmplName), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Load all templates
	for _, tmplName := range templates {
		tmpl, err := client.LoadTemplate(tmplName)
		require.NoError(t, err)
		assert.NotNil(t, tmpl)
	}

	// Verify cache size
	assert.Equal(t, 3, len(client.templateCache))

	// Load again and verify same instances returned
	for _, tmplName := range templates {
		cachedTmpl := client.templateCache[tmplName]
		tmpl, err := client.LoadTemplate(tmplName)
		require.NoError(t, err)
		assert.Equal(t, cachedTmpl, tmpl)
	}
}

// BenchmarkExecuteSPARQL benchmarks SPARQL query execution
func BenchmarkExecuteSPARQL(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	client := NewPoolPartyClient(server.URL, "admin", "password", "/tmp")
	query := "SELECT ?s WHERE { ?s ?p ?o } LIMIT 10"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.ExecuteSPARQL("testproject", query, "application/json")
	}
}

// BenchmarkLoadTemplate benchmarks template loading with cache
func BenchmarkLoadTemplate(b *testing.B) {
	tmpDir := b.TempDir()
	templateFile := filepath.Join(tmpDir, "test.sparql")
	err := ioutil.WriteFile(templateFile, []byte("SELECT ?s WHERE { ?s ?p ?o }"), 0644)
	if err != nil {
		b.Fatal(err)
	}

	client := NewPoolPartyClient("http://localhost", "user", "pass", tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.LoadTemplate("test.sparql")
	}
}
