package network

import (
	"io"
	"net/http"
	"os"

	eve "eve.evalgo.org/common"
)

func HttpClientDownloadFile(url, localPath string) {
	// Create the file
	out, err := os.Create(localPath)
	if err != nil {
		eve.Logger.Fatal("Failed to create file:", err)
	}
	defer out.Close()
	// Create a custom HTTP client (default redirect behavior: follows up to 10 redirects)
	// client := &http.Client{}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			eve.Logger.Fatal("Redirecting to:", req.URL)
			return nil // follow redirect
		},
	}
	// Create a new GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		eve.Logger.Fatal("Failed to create HTTP request:", err)
	}
	// Optional: add headers if needed (e.g., User-Agent)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0")
	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		eve.Logger.Fatal("Failed to perform request:", err)
	}
	defer resp.Body.Close()
	// Check response status
	if resp.StatusCode != http.StatusOK {
		eve.Logger.Fatal("Bad status:", resp.Status)
	}
	// Copy response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		eve.Logger.Fatal("Failed to write file:", err)
	}
	eve.Logger.Info("Download completed successfully.")
}
