package db

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// --- Test Setup / Teardown helpers ---
type testEnv struct {
	server   *httptest.Server
	baseURL  string
	cleanups []func()
}

func setupMockServer(handler http.HandlerFunc) *testEnv {
	server := httptest.NewServer(handler)
	return &testEnv{
		server:  server,
		baseURL: server.URL,
		cleanups: []func(){
			func() { server.Close() },
		},
	}
}

func (te *testEnv) teardown() {
	for _, fn := range te.cleanups {
		fn()
	}
}

// --- stripBOM tests ---
func TestStripBOM(t *testing.T) {
	withBOM := []byte{0xEF, 0xBB, 0xBF, 'h', 'e', 'l', 'l', 'o'}
	withoutBOM := []byte("hello")

	result := stripBOM(withBOM)
	if string(result) != "hello" {
		t.Errorf("expected 'hello', got %q", string(result))
	}

	result = stripBOM(withoutBOM)
	if string(result) != "hello" {
		t.Errorf("expected 'hello', got %q", string(result))
	}
}

// --- ImportRDF tests ---
func TestImportRDF_Success(t *testing.T) {
	env := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		w.WriteHeader(http.StatusOK)
		w.Write(body) // echo back
	})
	defer env.teardown()

	// Create a temp RDF file
	tmpFile, err := ioutil.TempFile("", "test.rdf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	rdfContent := `<rdf>hello</rdf>`
	if _, err := tmpFile.Write([]byte(rdfContent)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	respBody, err := ImportRDF(env.baseURL, "repo1", "user", "pass", tmpFile.Name(), "application/rdf+xml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if string(respBody) != rdfContent {
		t.Errorf("expected %q, got %q", rdfContent, string(respBody))
	}
}

func TestImportRDF_InvalidUTF8(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "invalid.rdf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write invalid UTF-8 bytes
	tmpFile.Write([]byte{0xff, 0xfe, 0xfd})
	tmpFile.Close()

	_, err = ImportRDF("http://localhost", "repo1", "user", "pass", tmpFile.Name(), "application/rdf+xml")
	if err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Errorf("expected invalid UTF-8 error, got %v", err)
	}
}

// --- ExportRDFXml tests ---
func TestExportRDFXml_Success(t *testing.T) {
	expected := `<rdf>world</rdf>`

	env := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expected))
	})
	defer env.teardown()

	tmpFile, err := ioutil.TempFile("", "export.rdf")
	if err != nil {
		t.Fatal(err)
	}
	outputFile := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(outputFile)

	err = ExportRDFXml(env.baseURL, "repo1", "user", "pass", outputFile, "application/rdf+xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ioutil.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

func TestExportRDFXml_Failure(t *testing.T) {
	env := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
	})
	defer env.teardown()

	tmpFile, err := ioutil.TempFile("", "export_fail.rdf")
	if err != nil {
		t.Fatal(err)
	}
	outputFile := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(outputFile)

	err = ExportRDFXml(env.baseURL, "repo1", "user", "pass", outputFile, "application/rdf+xml")
	if err == nil || !strings.Contains(err.Error(), "failed to export data") {
		t.Errorf("expected export error, got %v", err)
	}
}
