package db

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"encoding/json"
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
	mockRepos := []Repository{
		{ID: "repo1", Title: "Repository 1", Type: "memory"},
		{ID: "repo2", Title: "Repository 2", Type: "native"},
	}

	env := setup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/repositories" {
			t.Errorf("expected path /repositories, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockRepos)
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
