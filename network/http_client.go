package network

import (
	"fmt"
	"io"
	"net/http"
	"os"

	eve "eve.evalgo.org/common"
)

func HttpClientDownloadFile(url, localPath string) error {
	// Create the file
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()
	// Create a custom HTTP client (default redirect behavior: follows up to 10 redirects)
	// client := &http.Client{}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			eve.Logger.Info("Redirecting to:", req.URL)
			return nil // follow redirect
		},
	}
	// Create a new GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	// Optional: add headers if needed (e.g., User-Agent)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0")
	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	// Copy response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	eve.Logger.Info("Download completed successfully.")
	return nil
}
