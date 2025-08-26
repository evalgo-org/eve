package forge

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	eve "eve.evalgo.org/common"
	"eve.evalgo.org/network"
)

func GitlabRunners(url, token string) {
	git, err := gitlab.NewClient(token, gitlab.WithBaseURL(url+"/api/v4"))
	if err != nil {
		eve.Logger.Fatal("Failed to create client:", err)
	}
	runners, _, err := git.Runners.ListAllRunners(&gitlab.ListRunnersOptions{})
	for _, runner := range runners {
		eve.Logger.Info(runner)
	}
}

// available gitlab-runner types instance_type | group_type | project_type | user_type
func GitlabRegisterNewRunner(url, token, version, dataInit, registerArgs, sudoPass, gitlabUser string) {
	// get runner from the release website
	// due to usues on the gitlab site the packages are not installable of as for today
	// https://gitlab.com/gitlab-org/gitlab-runner/-/issues/38353
	// HttpClientDownloadFile("https://gitlab-runner-downloads.s3.amazonaws.com/"+version+"/rpm/gitlab-runner_amd64.rpm", "gitlab-runner_amd64.rpm")
	// we need to run the gitlab-runner as executable for now
	network.HttpClientDownloadFile("https://gitlab-runner-downloads.s3.amazonaws.com/"+version+"/binaries/gitlab-runner-linux-amd64", "gitlab-runner")
	// register the runner
	git, err := gitlab.NewClient(token, gitlab.WithBaseURL(url+"/api/v4"))
	if err != nil {
		eve.Logger.Fatal("Failed to create client:", err)
	}
	initOptions := gitlab.CreateUserRunnerOptions{}
	if err := json.Unmarshal([]byte(dataInit), &initOptions); err != nil {
		eve.Logger.Fatal("Failed to create gitlab runner:", err)
	}
	runner, _, err := git.Users.CreateUserRunner(&initOptions)
	if err != nil {
		eve.Logger.Fatal("Failed to create gitlab runner:", err)
	}
	eve.Logger.Info("running runner registration with token", runner.Token)
	eve.ShellSudoExecute(sudoPass, "mv ./gitlab-runner /usr/bin/gitlab-runner && chmod +x /usr/bin/gitlab-runner")
	eve.ShellSudoExecute(sudoPass, "gitlab-runner install --user "+gitlabUser)
	eve.ShellSudoExecute(sudoPass, "gitlab-runner register --token "+runner.Token+" --url "+url+" "+registerArgs)
}

func glabDownloadFile(url, filepath string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func glabDownloadArchive(client *gitlab.Client, projectID, sha, format, destPath string) error {
	opt := &gitlab.ArchiveOptions{
		SHA:    &sha,
		Format: &format, // "zip" or "tar.gz"
	}

	for i := 0; i < 10; i++ {
		archive, resp, err := client.Repositories.Archive(projectID, opt)
		if err != nil {
			return err
		}

		if resp.StatusCode == 202 {
			fmt.Println("Archive not ready, retrying...")
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("unexpected status: %s", resp.Status)
		}

		if err := os.WriteFile(destPath, archive, 0644); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("archive not ready after retries")
}

func glabUnzipStripTop(src, destDir string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Split path and remove first element (the repo root folder GitLab/GitHub add)
		parts := strings.SplitN(f.Name, string(os.PathSeparator), 2)
		if len(parts) < 2 {
			continue // skip root folder entry
		}
		relativePath := parts[1]

		fPath := filepath.Join(destDir, relativePath)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fPath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
			return err
		}

		in, err := f.Open()
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.OpenFile(fPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		if _, err = io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		out.Close()
	}
	return nil
}

func glabUnZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make sure the directory exists
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		// Extract the file
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		// Close resources
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func GitlabDownloadRepo(url, token, owner, repo, branch, filepath string) error {
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(url+"/api/v4"))
	if err != nil {
		return err
	}

	projectID := owner + "/" + repo
	sha := branch
	format := "zip"
	zipPath := repo + ".zip"
	extractDir := repo

	fmt.Printf("Downloading archive for %s@%s...\n", projectID, sha)
	if err := glabDownloadArchive(client, projectID, sha, format, zipPath); err != nil {
		return err
	}

	return glabUnzipStripTop(zipPath, extractDir)
}
