package db

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContextID tests the ContextID structure
func TestContextID(t *testing.T) {
	t.Run("valid context ID", func(t *testing.T) {
		ctx := ContextID{
			Type:  "uri",
			Value: "http://example.org/graph/1",
		}

		assert.Equal(t, "uri", ctx.Type)
		assert.Equal(t, "http://example.org/graph/1", ctx.Value)
	})

	t.Run("literal context ID", func(t *testing.T) {
		ctx := ContextID{
			Type:  "literal",
			Value: "some literal value",
		}

		assert.Equal(t, "literal", ctx.Type)
	})

	t.Run("blank node context ID", func(t *testing.T) {
		ctx := ContextID{
			Type:  "bnode",
			Value: "_:b123",
		}

		assert.Equal(t, "bnode", ctx.Type)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		ctx := ContextID{
			Type:  "uri",
			Value: "http://example.org/graph/1",
		}

		data, err := json.Marshal(ctx)
		require.NoError(t, err)

		var decoded ContextID
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, ctx.Type, decoded.Type)
		assert.Equal(t, ctx.Value, decoded.Value)
	})
}

// TestGraphDBBinding tests the GraphDBBinding structure
func TestGraphDBBinding(t *testing.T) {
	t.Run("repository binding", func(t *testing.T) {
		binding := GraphDBBinding{
			Readable: map[string]string{"type": "literal", "value": "true"},
			Id:       map[string]string{"type": "literal", "value": "test-repo"},
			Title:    map[string]string{"type": "literal", "value": "Test Repository"},
			Uri:      map[string]string{"type": "uri", "value": "http://example.org/repo/test"},
			Writable: map[string]string{"type": "literal", "value": "true"},
			ContextID: ContextID{
				Type:  "uri",
				Value: "http://example.org/context/default",
			},
		}

		assert.Equal(t, "test-repo", binding.Id["value"])
		assert.Equal(t, "Test Repository", binding.Title["value"])
		assert.Equal(t, "true", binding.Readable["value"])
		assert.Equal(t, "true", binding.Writable["value"])
	})

	t.Run("JSON serialization", func(t *testing.T) {
		binding := GraphDBBinding{
			Id:    map[string]string{"type": "literal", "value": "repo1"},
			Title: map[string]string{"type": "literal", "value": "Repository 1"},
			ContextID: ContextID{
				Type:  "uri",
				Value: "http://example.org/ctx",
			},
		}

		data, err := json.Marshal(binding)
		require.NoError(t, err)
		assert.Contains(t, string(data), "repo1")
		assert.Contains(t, string(data), "Repository 1")

		var decoded GraphDBBinding
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "repo1", decoded.Id["value"])
	})
}

// TestGraphDBResults tests the GraphDBResults structure
func TestGraphDBResults(t *testing.T) {
	t.Run("multiple bindings", func(t *testing.T) {
		results := GraphDBResults{
			Bindings: []GraphDBBinding{
				{
					Id:    map[string]string{"value": "repo1"},
					Title: map[string]string{"value": "Repository 1"},
				},
				{
					Id:    map[string]string{"value": "repo2"},
					Title: map[string]string{"value": "Repository 2"},
				},
			},
		}

		assert.Len(t, results.Bindings, 2)
		assert.Equal(t, "repo1", results.Bindings[0].Id["value"])
		assert.Equal(t, "repo2", results.Bindings[1].Id["value"])
	})

	t.Run("empty bindings", func(t *testing.T) {
		results := GraphDBResults{
			Bindings: []GraphDBBinding{},
		}

		assert.Empty(t, results.Bindings)
		assert.NotNil(t, results.Bindings)
	})
}

// TestGraphDBResponse tests the complete response structure
func TestGraphDBResponse(t *testing.T) {
	t.Run("full response", func(t *testing.T) {
		response := GraphDBResponse{
			Head: []interface{}{"id", "title", "uri"},
			Results: GraphDBResults{
				Bindings: []GraphDBBinding{
					{
						Id:    map[string]string{"value": "test-repo"},
						Title: map[string]string{"value": "Test Repository"},
					},
				},
			},
		}

		assert.Len(t, response.Head, 3)
		assert.Len(t, response.Results.Bindings, 1)
		assert.Equal(t, "test-repo", response.Results.Bindings[0].Id["value"])
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"results": {
				"bindings": [
					{
						"id": {"type": "literal", "value": "repo1"},
						"title": {"type": "literal", "value": "Repository 1"},
						"contextID": {"type": "uri", "value": "http://example.org/ctx"}
					}
				]
			}
		}`

		var response GraphDBResponse
		err := json.Unmarshal([]byte(jsonData), &response)
		require.NoError(t, err)

		assert.Len(t, response.Results.Bindings, 1)
		assert.Equal(t, "repo1", response.Results.Bindings[0].Id["value"])
		assert.Equal(t, "http://example.org/ctx", response.Results.Bindings[0].ContextID.Value)
	})
}

// TestGraphDBRepositories tests repository listing with mock HTTP server
func TestGraphDBRepositories(t *testing.T) {
	t.Run("successful repository list", func(t *testing.T) {
		// Create mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repositories", r.URL.Path)
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			response := GraphDBResponse{
				Results: GraphDBResults{
					Bindings: []GraphDBBinding{
						{
							Id:    map[string]string{"value": "repo1"},
							Title: map[string]string{"value": "Test Repository 1"},
						},
						{
							Id:    map[string]string{"value": "repo2"},
							Title: map[string]string{"value": "Test Repository 2"},
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		// Use the test server
		HttpClient = server.Client()

		repos, err := GraphDBRepositories(server.URL, "", "")
		require.NoError(t, err)
		require.NotNil(t, repos)

		assert.Len(t, repos.Results.Bindings, 2)
		assert.Equal(t, "repo1", repos.Results.Bindings[0].Id["value"])
		assert.Equal(t, "repo2", repos.Results.Bindings[1].Id["value"])
	})

	t.Run("with authentication", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "admin", user)
			assert.Equal(t, "password", pass)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(GraphDBResponse{
				Results: GraphDBResults{Bindings: []GraphDBBinding{}},
			})
		}))
		defer server.Close()

		HttpClient = server.Client()

		_, err := GraphDBRepositories(server.URL, "admin", "password")
		require.NoError(t, err)
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		HttpClient = server.Client()

		repos, err := GraphDBRepositories(server.URL, "", "")
		assert.Error(t, err)
		assert.Nil(t, repos)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		HttpClient = server.Client()

		repos, err := GraphDBRepositories(server.URL, "", "")
		assert.Error(t, err)
		assert.Nil(t, repos)
		assert.Contains(t, err.Error(), "401")
	})
}

// TestGraphDBListGraphs tests graph listing functionality
func TestGraphDBListGraphs(t *testing.T) {
	t.Run("successful graph list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repositories/test-repo/rdf-graphs", r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			response := GraphDBResponse{
				Results: GraphDBResults{
					Bindings: []GraphDBBinding{
						{
							ContextID: ContextID{
								Type:  "uri",
								Value: "http://example.org/graph/1",
							},
						},
						{
							ContextID: ContextID{
								Type:  "uri",
								Value: "http://example.org/graph/2",
							},
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		HttpClient = server.Client()

		graphs, err := GraphDBListGraphs(server.URL, "", "", "test-repo")
		require.NoError(t, err)
		require.NotNil(t, graphs)

		assert.Len(t, graphs.Results.Bindings, 2)
		assert.Equal(t, "http://example.org/graph/1", graphs.Results.Bindings[0].ContextID.Value)
	})

	t.Run("repository not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		HttpClient = server.Client()

		graphs, err := GraphDBListGraphs(server.URL, "", "", "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, graphs)
		assert.Contains(t, err.Error(), "could not run GraphDBListGraphs")
	})
}

// TestGraphDBDeleteRepository tests repository deletion
func TestGraphDBDeleteRepository(t *testing.T) {
	t.Run("successful deletion", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rest/repositories/test-repo", r.URL.Path)
			assert.Equal(t, "DELETE", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Repository deleted"))
		}))
		defer server.Close()

		HttpClient = server.Client()

		err := GraphDBDeleteRepository(server.URL, "", "", "test-repo")
		assert.NoError(t, err)
	})

	t.Run("deletion with authentication", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "admin", user)
			assert.Equal(t, "secret", pass)

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		HttpClient = server.Client()

		err := GraphDBDeleteRepository(server.URL, "admin", "secret", "test-repo")
		assert.NoError(t, err)
	})

	t.Run("repository not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Repository not found"))
		}))
		defer server.Close()

		HttpClient = server.Client()

		err := GraphDBDeleteRepository(server.URL, "", "", "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not delete repository")
	})
}

// TestGraphDBRepositoryConf tests configuration download
func TestGraphDBRepositoryConf(t *testing.T) {
	t.Run("successful config download", func(t *testing.T) {
		tempDir := t.TempDir()
		originalWd, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalWd)

		turtleContent := `@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix rep: <http://www.openrdf.org/config/repository#> .

<#test-repo> a rep:Repository ;
    rep:repositoryID "test-repo" ;
    rdfs:label "Test Repository" .`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rest/repositories/test-repo/download-ttl", r.URL.Path)
			assert.Equal(t, "text/turtle", r.Header.Get("Accept"))

			w.Header().Set("Content-Type", "text/turtle")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(turtleContent))
		}))
		defer server.Close()

		HttpClient = server.Client()

		filename, err := GraphDBRepositoryConf(server.URL, "", "", "test-repo")
		require.NoError(t, err)
		assert.Equal(t, "test-repo.ttl", filename)

		// Verify file was created
		content, err := os.ReadFile(filename)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test-repo")
		assert.Contains(t, string(content), "Test Repository")
	})
}

// TestGraphDBRepositoryBrf tests BRF export
func TestGraphDBRepositoryBrf(t *testing.T) {
	t.Run("successful BRF export", func(t *testing.T) {
		tempDir := t.TempDir()
		originalWd, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalWd)

		brfData := []byte{0x00, 0x01, 0x02, 0x03} // Mock binary data

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repositories/test-repo/statements", r.URL.Path)
			assert.Equal(t, "application/x-binary-rdf", r.Header.Get("Accept"))

			w.Header().Set("Content-Type", "application/x-binary-rdf")
			w.WriteHeader(http.StatusOK)
			w.Write(brfData)
		}))
		defer server.Close()

		HttpClient = server.Client()

		filename, err := GraphDBRepositoryBrf(server.URL, "", "", "test-repo")
		require.NoError(t, err)
		assert.Equal(t, "test-repo.brf", filename)

		// Verify file was created
		content, err := os.ReadFile(filename)
		require.NoError(t, err)
		assert.Equal(t, brfData, content)
	})
}

// TestGraphDBExportGraphRdf tests RDF graph export
func TestGraphDBExportGraphRdf(t *testing.T) {
	t.Run("successful graph export", func(t *testing.T) {
		tempDir := t.TempDir()
		exportFile := filepath.Join(tempDir, "export.rdf")

		rdfContent := `<?xml version="1.0" encoding="UTF-8"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about="http://example.org/resource/1">
    <rdfs:label>Test Resource</rdfs:label>
  </rdf:Description>
</rdf:RDF>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repositories/test-repo/rdf-graphs/service", r.URL.Path)
			assert.Contains(t, r.URL.Query().Get("graph"), "http://example.org/graph/1")
			assert.Equal(t, "application/rdf+xml", r.Header.Get("Accept"))

			w.Header().Set("Content-Type", "application/rdf+xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(rdfContent))
		}))
		defer server.Close()

		HttpClient = server.Client()

		err := GraphDBExportGraphRdf(server.URL, "", "", "test-repo", "http://example.org/graph/1", exportFile)
		require.NoError(t, err)

		// Verify file was created
		content, err := os.ReadFile(exportFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Test Resource")
	})

	t.Run("export with authentication", func(t *testing.T) {
		tempDir := t.TempDir()
		exportFile := filepath.Join(tempDir, "export.rdf")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "user1", user)
			assert.Equal(t, "pass1", pass)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<rdf:RDF/>"))
		}))
		defer server.Close()

		HttpClient = server.Client()

		err := GraphDBExportGraphRdf(server.URL, "user1", "pass1", "repo", "http://graph", exportFile)
		assert.NoError(t, err)
	})
}

// TestGraphDBImportGraphRdf tests RDF graph import
func TestGraphDBImportGraphRdf(t *testing.T) {
	t.Run("successful graph import", func(t *testing.T) {
		tempDir := t.TempDir()
		importFile := filepath.Join(tempDir, "import.rdf")

		rdfContent := `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about="http://example.org/resource">
    <rdfs:label>Imported Resource</rdfs:label>
  </rdf:Description>
</rdf:RDF>`

		err := os.WriteFile(importFile, []byte(rdfContent), 0644)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			assert.Equal(t, "/repositories/test-repo/rdf-graphs/service", r.URL.Path)
			assert.Contains(t, r.URL.Query().Get("graph"), "http://example.org/graph/import")
			assert.Equal(t, "application/rdf+xml", r.Header.Get("Content-Type"))

			body, _ := io.ReadAll(r.Body)
			assert.Contains(t, string(body), "Imported Resource")

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		HttpClient = server.Client()

		err = GraphDBImportGraphRdf(server.URL, "", "", "test-repo", "http://example.org/graph/import", importFile)
		assert.NoError(t, err)
	})

	t.Run("file not found", func(t *testing.T) {
		err := GraphDBImportGraphRdf("http://localhost:7200", "", "", "repo", "graph", "/nonexistent/file.rdf")
		assert.Error(t, err)
	})
}

// TestGraphDBRestoreConf tests configuration restoration
func TestGraphDBRestoreConf(t *testing.T) {
	t.Run("successful config restore", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "restore.ttl")

		turtleContent := `@prefix rep: <http://www.openrdf.org/config/repository#> .
<#restored-repo> a rep:Repository ;
    rep:repositoryID "restored-repo" .`

		err := os.WriteFile(configFile, []byte(turtleContent), 0644)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/rest/repositories", r.URL.Path)
			assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("Repository created successfully"))
		}))
		defer server.Close()

		HttpClient = server.Client()

		err = GraphDBRestoreConf(server.URL, "", "", configFile)
		assert.NoError(t, err)
	})

	t.Run("file not found", func(t *testing.T) {
		err := GraphDBRestoreConf("http://localhost:7200", "", "", "/nonexistent/config.ttl")
		assert.Error(t, err)
	})
}

// TestGraphDBRestoreBrf tests BRF restoration
func TestGraphDBRestoreBrf(t *testing.T) {
	t.Run("successful BRF restore", func(t *testing.T) {
		tempDir := t.TempDir()
		brfFile := filepath.Join(tempDir, "test-repo.brf")

		brfData := []byte{0x00, 0x01, 0x02, 0x03}
		err := os.WriteFile(brfFile, brfData, 0644)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/repositories/test-repo/statements", r.URL.Path)
			assert.Equal(t, "application/x-binary-rdf", r.Header.Get("Content-Type"))

			body, _ := io.ReadAll(r.Body)
			assert.Equal(t, brfData, body)

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		HttpClient = server.Client()

		err = GraphDBRestoreBrf(server.URL, "", "", brfFile)
		assert.NoError(t, err)
	})

	t.Run("repository name extraction", func(t *testing.T) {
		tempDir := t.TempDir()
		brfFile := filepath.Join(tempDir, "my-custom-repo.brf")

		err := os.WriteFile(brfFile, []byte{0x00}, 0644)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Repository name should be extracted from filename
			assert.Equal(t, "/repositories/my-custom-repo/statements", r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		HttpClient = server.Client()

		err = GraphDBRestoreBrf(server.URL, "", "", brfFile)
		assert.NoError(t, err)
	})
}

// TestGraphDBDeleteGraph tests graph deletion via SPARQL
func TestGraphDBDeleteGraph(t *testing.T) {
	t.Run("successful graph deletion", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/repositories/test-repo/statements", r.URL.Path)
			assert.Equal(t, "application/sparql-update", r.Header.Get("Content-Type"))

			body, _ := io.ReadAll(r.Body)
			sparql := string(body)
			assert.Contains(t, sparql, "DROP GRAPH")
			assert.Contains(t, sparql, "http://example.org/graph/delete")

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		HttpClient = server.Client()

		err := GraphDBDeleteGraph(server.URL, "", "", "test-repo", "http://example.org/graph/delete")
		assert.NoError(t, err)
	})

	t.Run("SPARQL format validation", func(t *testing.T) {
		var receivedSPARQL string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			receivedSPARQL = string(body)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		HttpClient = server.Client()

		graphURI := "http://example.org/test-graph"
		GraphDBDeleteGraph(server.URL, "", "", "repo", graphURI)

		expected := "DROP GRAPH <" + graphURI + ">"
		assert.Equal(t, expected, receivedSPARQL)
	})
}

// TestGraphDBURLConstruction tests URL building in various functions
func TestGraphDBURLConstruction(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		expectedPath string
		testFunc     func(string) error
	}{
		{
			name:         "list graphs URL",
			baseURL:      "http://localhost:7200",
			expectedPath: "/repositories/test/rdf-graphs",
			testFunc: func(url string) error {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/repositories/test/rdf-graphs", r.URL.Path)
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(GraphDBResponse{})
				}))
				defer server.Close()
				HttpClient = server.Client()
				_, err := GraphDBListGraphs(server.URL, "", "", "test")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc(tt.baseURL)
			// We're primarily testing URL construction, errors are secondary
			_ = err
		})
	}
}

// TestHTTPClientCustomization tests the global HttpClient variable
func TestHTTPClientCustomization(t *testing.T) {
	t.Run("default client", func(t *testing.T) {
		// Save original client
		originalClient := HttpClient
		defer func() { HttpClient = originalClient }()

		// Reset to default
		HttpClient = http.DefaultClient
		assert.NotNil(t, HttpClient)
	})

	t.Run("custom client", func(t *testing.T) {
		originalClient := HttpClient
		defer func() { HttpClient = originalClient }()

		customClient := &http.Client{}
		HttpClient = customClient

		assert.Equal(t, customClient, HttpClient)
	})
}

// BenchmarkGraphDBJSONSerialization benchmarks JSON operations
func BenchmarkGraphDBJSONSerialization(b *testing.B) {
	response := GraphDBResponse{
		Results: GraphDBResults{
			Bindings: []GraphDBBinding{
				{
					Id:    map[string]string{"type": "literal", "value": "repo1"},
					Title: map[string]string{"type": "literal", "value": "Repository 1"},
					ContextID: ContextID{
						Type:  "uri",
						Value: "http://example.org/context/1",
					},
				},
			},
		},
	}

	b.Run("marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(response)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		data, _ := json.Marshal(response)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var r GraphDBResponse
			_ = json.Unmarshal(data, &r)
		}
	})
}

// TestGraphDBAuthentication tests authentication header handling
func TestGraphDBAuthentication(t *testing.T) {
	t.Run("with credentials", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			assert.True(t, ok, "Basic auth should be present")
			assert.Equal(t, "testuser", user)
			assert.Equal(t, "testpass", pass)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(GraphDBResponse{})
		}))
		defer server.Close()

		HttpClient = server.Client()
		_, _ = GraphDBRepositories(server.URL, "testuser", "testpass")
	})

	t.Run("without credentials", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _, ok := r.BasicAuth()
			assert.False(t, ok, "Basic auth should not be present")

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(GraphDBResponse{})
		}))
		defer server.Close()

		HttpClient = server.Client()
		_, _ = GraphDBRepositories(server.URL, "", "")
	})
}

// TestGraphDBHeadersAndContentTypes tests HTTP headers
func TestGraphDBHeadersAndContentTypes(t *testing.T) {
	tests := []struct {
		name                string
		function            func(string) error
		expectedContentType string
		expectedAccept      string
	}{
		{
			name: "repositories list",
			function: func(url string) error {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "application/json", r.Header.Get("Accept"))
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(GraphDBResponse{})
				}))
				defer server.Close()
				HttpClient = server.Client()
				_, err := GraphDBRepositories(server.URL, "", "")
				return err
			},
			expectedAccept: "application/json",
		},
		{
			name: "repository config",
			function: func(url string) error {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "text/turtle", r.Header.Get("Accept"))
					w.WriteHeader(http.StatusOK)
				}))
				defer server.Close()

				tempDir := os.TempDir()
				originalWd, _ := os.Getwd()
				os.Chdir(tempDir)
				defer os.Chdir(originalWd)

				HttpClient = server.Client()
				GraphDBRepositoryConf(server.URL, "", "", "test")
				return nil
			},
			expectedAccept: "text/turtle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.function("http://localhost:7200")
		})
	}
}

// TestErrorMessageFormats tests error messages from various functions
func TestErrorMessageFormats(t *testing.T) {
	t.Run("repository list error message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		HttpClient = server.Client()

		_, err := GraphDBRepositories(server.URL, "", "")
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "status code")
		assert.Contains(t, err.Error(), "403")
	})

	t.Run("delete repository error message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		HttpClient = server.Client()

		err := GraphDBDeleteRepository(server.URL, "", "", "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not delete repository")
	})
}
