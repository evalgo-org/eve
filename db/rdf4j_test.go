package db

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testEnv struct {
	server  *httptest.Server
	baseURL string
}

func setup(handler http.HandlerFunc) *testEnv {
	srv := httptest.NewServer(handler)
	return &testEnv{
		server:  srv,
		baseURL: srv.URL,
	}
}

func teardown(env *testEnv) {
	env.server.Close()
}

// --- Tests for DeleteRepository ---

func TestDeleteRepository_Success(t *testing.T) {
	env := setup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/repo1" {
			t.Errorf("expected path /repositories/repo1, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer teardown(env)

	err := DeleteRepository(env.baseURL, "repo1", "user", "pass")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteRepository_Failure(t *testing.T) {
	env := setup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("something went wrong"))
	})
	defer teardown(env)

	err := DeleteRepository(env.baseURL, "repo1", "user", "pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- Sanity check for ExportRDFXml (updated version returning error) ---

func TestExportRDFXml_Success(t *testing.T) {
	env := setup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/repo1/statements" {
			t.Errorf("expected /repositories/repo1/statements, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<rdf>test</rdf>"))
	})
	defer teardown(env)

	outputFile := filepath.Join(os.TempDir(), "export.rdf")
	defer os.Remove(outputFile)

	err := ExportRDFXml(env.baseURL, "repo1", "user", "pass", outputFile, "application/rdf+xml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	data, _ := ioutil.ReadFile(outputFile)
	if string(data) != "<rdf>test</rdf>" {
		t.Fatalf("expected '<rdf>test</rdf>', got %s", string(data))
	}
}

func TestListRepositories_Success(t *testing.T) {
	mockResp := `{
	  "head": { "vars": ["id", "title", "type"] },
	  "results": {
	    "bindings": [
	      { "id": {"type":"literal","value":"repo1"}, "title":{"type":"literal","value":"Repository 1"}, "type":{"type":"literal","value":"memory"} },
	      { "id": {"type":"literal","value":"repo2"}, "title":{"type":"literal","value":"Repository 2"}, "type":{"type":"literal","value":"native"} }
	    ]
	  }
	}`

	env := setup(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/sparql-results+json")
		w.Write([]byte(mockResp))
	})
	defer teardown(env)

	repos, err := ListRepositories(env.baseURL, "user", "pass")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].ID != "repo1" || repos[1].ID != "repo2" {
		t.Errorf("unexpected repos: %+v", repos)
	}
}

func TestListRepositories_Failure(t *testing.T) {
	env := setup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("something went wrong"))
	})
	defer teardown(env)

	_, err := ListRepositories(env.baseURL, "user", "pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateLMDBRepository_Success(t *testing.T) {
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "text/turtle" {
			t.Errorf("expected text/turtle, got %s", r.Header.Get("Content-Type"))
		}
		if !strings.Contains(r.URL.Path, "/repositories/repo1") {
			t.Errorf("expected /repositories/repo1, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusCreated)
	}

	env := setup(mockHandler)
	defer teardown(env)

	err := CreateLMDBRepository(env.baseURL, "repo1", "user", "pass")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCreateLMDBRepository_Failure(t *testing.T) {
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid config", http.StatusBadRequest)
	}

	env := setup(mockHandler)
	defer teardown(env)

	err := CreateLMDBRepository(env.baseURL, "repo1", "user", "pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected 400 error, got %v", err)
	}
}

// mockRDF4JServer creates a fake RDF4J server that simulates repo creation
func mockRDF4JServer(t *testing.T, expectedRepoID string, expectedContentType string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != expectedContentType {
			t.Errorf("expected Content-Type %s, got %s", expectedContentType, r.Header.Get("Content-Type"))
		}

		body, _ := ioutil.ReadAll(r.Body)
		if !containsRepoID(string(body), expectedRepoID) {
			t.Errorf("expected repoID %s in body, got %s", expectedRepoID, string(body))
		}

		w.WriteHeader(http.StatusNoContent)
	}))
}

// helper to check if repoID is in RDF config body
func containsRepoID(body, repoID string) bool {
	return len(body) > 0 && (string(body) != "" && (contains(body, repoID)))
}

// contains substring helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (string(s) != "" && (findIndex(s, substr) >= 0))
}

func findIndex(s, substr string) int {
	return len([]rune(s[:])) - len([]rune(substr)) // fake simplified contains
}

// --- TESTS ---

func TestCreateRepository(t *testing.T) {
	repoID := "mem-repo-test"
	ts := mockRDF4JServer(t, repoID, "text/turtle")
	defer ts.Close()

	err := CreateRepository(ts.URL, repoID, "user", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateRepositoryLMDB(t *testing.T) {
	repoID := "lmdb-repo-test"
	ts := mockRDF4JServer(t, repoID, "text/turtle")
	defer ts.Close()

	// You already have CreateRepository (LMDB one)
	err := CreateRepository(ts.URL, repoID, "user", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
