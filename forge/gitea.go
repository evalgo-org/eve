// Package forge provides utilities for interacting with source code forges and version control systems.
// It includes functions for retrieving repository archives from platforms like Gitea,
// enabling integration with source code repositories for build, deployment, or analysis purposes.
//
// Features:
//   - Repository archive retrieval from Gitea instances
//   - Authentication support using personal access tokens
//   - Archive format selection (currently supports tar.gz)
package forge

import (
	"fmt"
	"io"
	"os"

	"code.gitea.io/sdk/gitea"
)

// GiteaGetRepo retrieves a repository archive from a Gitea instance and saves it as a local file.
// This function:
//  1. Creates a Gitea client with the provided URL and authentication token
//  2. Requests a tar.gz archive of the specified repository and branch
//  3. Saves the archive to a local file named "{repo}-{branch}.tar.gz"
//
// Parameters:
//   - url: Base URL of the Gitea instance (e.g., "https://gitea.example.com")
//   - token: Personal access token for authentication
//   - owner: Owner/organization name of the repository
//   - repo: Name of the repository
//   - branch: Branch, tag, or commit hash to retrieve
//
// Returns:
//   - string: Path to the created archive file
//   - error: Error if any step fails, nil on success
//
// The resulting archive file will be saved in the current working directory with the name:
//
//	"{repo}-{branch}.tar.gz"
//
// Example:
//
//	filename, err := GiteaGetRepo("https://gitea.example.com", "my-token", "my-org", "my-repo", "main")
//	if err != nil {
//	    log.Fatal(err)
//	}
func GiteaGetRepo(url, token, owner, repo, branch string) (string, error) {
	client, err := gitea.NewClient(url, gitea.SetToken(token))
	if err != nil {
		return "", fmt.Errorf("failed to create Gitea client: %w", err)
	}

	reader, resp, err := client.GetArchiveReader(owner, repo, branch, gitea.TarGZArchive)
	if err != nil {
		return "", fmt.Errorf("failed to get archive reader: %w", err)
	}
	defer resp.Body.Close()

	filename := repo + "-" + branch + ".tar.gz"
	out, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, reader); err != nil {
		return "", fmt.Errorf("failed to write archive: %w", err)
	}

	return filename, nil
}
