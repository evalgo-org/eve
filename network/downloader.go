// Package network provides utilities for network operations, particularly
// for downloading files with progress tracking.
//
// Features:
//   - File downloading with progress reporting
//   - Support for authenticated downloads using tokens
//   - Human-readable progress display
//   - Safe file writing with temporary files
package network

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	eve "eve.evalgo.org/common"
	"github.com/dustin/go-humanize"
)

// WriteCounter counts the number of bytes written to it and provides
// progress reporting functionality. It implements the io.Writer interface
// and is used to track download progress.
//
// Fields:
//   - Total: The total number of bytes written so far
type WriteCounter struct {
	Total uint64
}

// Write implements the io.Writer interface for WriteCounter.
// It updates the byte count and returns the number of bytes written.
//
// Parameters:
//   - p: The byte slice being written
//
// Returns:
//   - int: The number of bytes written (same as len(p))
//   - error: Always nil, as this is just a counter
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

// PrintProgress displays the current download progress.
// It shows the total bytes downloaded in a human-readable format
// and updates the display in place.
func (wc WriteCounter) PrintProgress() {
	// Clear the line by overwriting with spaces
	eve.Logger.Info("\r", strings.Repeat(" ", 50))
	// Print the current progress
	eve.Logger.Info("\rDownloading...", humanize.Bytes(wc.Total), "complete")
}

// DownloadFile downloads a file from a URL to a local path with progress tracking.
// This function:
//  1. Creates a temporary file for safe downloading
//  2. Makes an authenticated HTTP GET request
//  3. Downloads the file while tracking progress
//  4. Renames the temporary file to the final destination on success
//
// Parameters:
//   - token: Authentication token for the request (can be empty if no auth needed)
//   - url: The URL to download from
//   - filepath: The local path to save the file to
//
// Returns:
//   - error: If any step in the download process fails
//
// The function uses a temporary file (.tmp extension) during download
// and only renames it to the final destination after successful completion.
// This prevents corrupted files if the download is interrupted.
func DownloadFile(token string, url string, filepath string) error {
	// Create the temporary output file
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()

	// Create HTTP client and request
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Add authorization header if token is provided
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create progress counter and download the file
	counter := &WriteCounter{}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}

	// Print final progress (100%)
	eve.Logger.Info("\rDownload complete: ", humanize.Bytes(counter.Total))

	// Rename temporary file to final destination
	err = os.Rename(filepath+".tmp", filepath)
	if err != nil {
		return err
	}

	return nil
}
