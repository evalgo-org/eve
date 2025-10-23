package db

import (
	"fmt"
	"os"
	// "context"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	// "net/url"
	"errors"
	"path/filepath"
	"strings"

	eve "eve.evalgo.org/common"
)

type ContextID struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type GraphDBBinding struct {
	Readable  map[string]string `json:"readable"`
	Id        map[string]string `json:"id"`
	Title     map[string]string `json:"title"`
	Uri       map[string]string `json:"uri"`
	Writable  map[string]string `json:"writable"`
	ContextID ContextID         `json:"contextID"`
}

type GraphDBResults struct {
	Bindings []GraphDBBinding `json:"bindings"`
}

type GraphDBResponse struct {
	Head    []interface{}  `json:"head>vars"`
	Results GraphDBResults `json:"results"`
}

func GraphDBRepositories(url string, user string, pass string) (*GraphDBResponse,error) {
	tgt_url := url + "/repositories"
	req, err := http.NewRequest("GET", tgt_url, nil)
	if err != nil {
		return nil, err
	}
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/json")
	// req.Header.Add("Authorization", "Bearer "+token)
	// req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		response := GraphDBResponse{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, err
		}
		// eve.Logger.Info(string(body))
		return &response, nil
	}
	return nil, fmt.Errorf("could not return repositories because of status code: %d", res.StatusCode)
}

func GraphDBRepositoryConf(url string, user string, pass string, repo string) string {
	tgt_url := url + "/rest/repositories/" + repo + "/download-ttl"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "text/turtle")
	// req.Header.Add("Authorization", "Bearer "+token)
	// req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		out, err := os.Create(repo + ".ttl")
		if err != nil {
			eve.Logger.Info(err)
		}
		defer out.Close()
		io.Copy(out, res.Body)
		return repo + ".ttl"
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return ""
}

func GraphDBRepositoryBrf(url string, user string, pass string, repo string) string {
	tgt_url := url + "/repositories/" + repo + "/statements"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/x-binary-rdf")
	// req.Header.Add("Authorization", "Bearer "+token)
	// req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		out, err := os.Create(repo + ".brf")
		if err != nil {
			eve.Logger.Info(err)
		}
		defer out.Close()
		io.Copy(out, res.Body)
		return repo + ".brf"
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return ""
}

func GraphDBRestoreConf(url string, user string, pass string, restoreFile string) error {
	var (
		buf = new(bytes.Buffer)
		w   = multipart.NewWriter(buf)
	)
	part, err := w.CreateFormFile("config", filepath.Base(restoreFile))
	if err != nil {
		return err
	}
	fData, err := ioutil.ReadFile(restoreFile)
	if err != nil {
		return err
	}
	_, err = part.Write(fData)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	tgt_url := url + "/rest/repositories"
	req, _ := http.NewRequest("POST", tgt_url, buf)
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", w.FormDataContentType())
	res, _ := http.DefaultClient.Do(req)
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode == http.StatusCreated {
		eve.Logger.Info(string(body))
		return nil
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return errors.New("could not run GraphDBRestoreConf")
}

func GraphDBRestoreBrf(url string, user string, pass string, restoreFile string) error {
	fData, err := ioutil.ReadFile(restoreFile)
	if err != nil {
		return err
	}
	repo := strings.TrimSuffix(filepath.Base(restoreFile), filepath.Ext(restoreFile))
	tgt_url := url + "/repositories/" + repo + "/statements"
	req, _ := http.NewRequest("POST", tgt_url, bytes.NewBuffer(fData))
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-binary-rdf")
	res, _ := http.DefaultClient.Do(req)
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		eve.Logger.Info(string(body))
		return nil
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return errors.New("could not run GraphDBRestoreBrf")
}

func GraphDBImportGraphRdf(url, user, pass, repo, graph, restoreFile string) error {
	fData, err := ioutil.ReadFile(restoreFile)
	if err != nil {
		return err
	}
	tgt_url := url + "/repositories/" + repo + "/rdf-graphs/service"
	req, _ := http.NewRequest("PUT", tgt_url, bytes.NewBuffer(fData))
	values := req.URL.Query()
	values.Add("graph", graph)
	req.URL.RawQuery = values.Encode()
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/rdf+xml")
	res, _ := http.DefaultClient.Do(req)
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		eve.Logger.Info(string(body))
		return nil
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return errors.New("could not run GraphDBImportRdf")
}

func GraphDBDeleteRepository(URL, user, pass, repo string) error {
	tgt_url := URL + "/rest/repositories/" + repo
	eve.Logger.Info(tgt_url)

	req, err := http.NewRequest("DELETE", tgt_url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}

	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNoContent {
		eve.Logger.Info("Repository deleted successfully:", string(body))
		return nil
	}

	eve.Logger.Error("Failed to delete repository:", res.StatusCode, http.StatusText(res.StatusCode), string(body))
	return fmt.Errorf("could not delete repository: %s (%d)", http.StatusText(res.StatusCode), res.StatusCode)
}

func GraphDBDeleteGraph(URL, user, pass, repo, graph string) error {
	tgt_url := URL + "/repositories/" + repo + "/statements"
	eve.Logger.Info(tgt_url)
	fData := []byte("DROP GRAPH <" + graph + ">")
	req, _ := http.NewRequest("POST", tgt_url, bytes.NewBuffer(fData))
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Content-Type", "application/sparql-update")
	res, _ := http.DefaultClient.Do(req)
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		eve.Logger.Info(string(body))
		return nil
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return errors.New("could not run GraphDBDeleteGraph")
}

func GraphDBListGraphs(url, user, pass, repo string) (*GraphDBResponse, error) {
	tgt_url := url + "/repositories/" + repo + "/rdf-graphs"
	req, err := http.NewRequest("GET", tgt_url, nil)
	if err != nil {
		return nil, err
	}
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	fmt.Println(res.StatusCode)
	if res.StatusCode == http.StatusOK {
		response := GraphDBResponse{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, err
		}
		return &response, nil
	}
	return nil, errors.New("could not run GraphDBListGraphs on " + repo )
}

func GraphDBExportGraphRdf(url, user, pass, repo, graph, exportFile string) error {
	tgt_url := url + "/repositories/" + repo + "/rdf-graphs/service"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	values := req.URL.Query()
	values.Add("graph", graph)
	req.URL.RawQuery = values.Encode()
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/rdf+xml")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Info("Failed to create file:", err)
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		outFile, err := os.Create(exportFile)
		if err != nil {
			eve.Logger.Info("Failed to create file:", err)
			return err
		}
		defer outFile.Close()
		_, err = io.Copy(outFile, res.Body)
		if err != nil {
			eve.Logger.Info("Error writing response to file:", err)
			return err
		}
		return nil
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return errors.New("could not run GraphDBExportGraphRdf")
}
