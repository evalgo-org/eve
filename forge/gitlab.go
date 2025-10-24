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

// GitlabCreateTag creates a new tag on the specified repository with a tag message
func GitlabCreateTag(url, token, projectID, tagName, ref, message string) (*gitlab.Tag, error) {
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(url+"/api/v4"))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	createTagOptions := &gitlab.CreateTagOptions{
		TagName: &tagName,
		Ref:     &ref,
		Message: &message,
	}

	tag, _, err := client.Tags.CreateTag(projectID, createTagOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag '%s': %w", tagName, err)
	}

	eve.Logger.Info(fmt.Sprintf("Successfully created tag '%s' on project '%s'", tagName, projectID))
	return tag, nil
}

// JobInfo represents simplified job information for display
type JobInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Stage    string `json:"stage"`
	Ref      string `json:"ref"`
	Pipeline int    `json:"pipeline_id"`
}

// GitlabListJobsForTag lists all running jobs and their states for the given tag
func GitlabListJobsForTag(url, token, projectID, tagName string) ([]JobInfo, error) {
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(url+"/api/v4"))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// First, get pipelines for the specific tag
	pipelineOptions := &gitlab.ListProjectPipelinesOptions{
		Ref: &tagName,
	}

	pipelines, _, err := client.Pipelines.ListProjectPipelines(projectID, pipelineOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines for tag '%s': %w", tagName, err)
	}

	if len(pipelines) == 0 {
		eve.Logger.Info(fmt.Sprintf("No pipelines found for tag '%s'", tagName))
		return []JobInfo{}, nil
	}

	var allJobs []JobInfo

	// Get jobs for each pipeline
	for _, pipeline := range pipelines {
		jobOptions := &gitlab.ListJobsOptions{}

		jobs, _, err := client.Jobs.ListPipelineJobs(projectID, pipeline.ID, jobOptions)
		if err != nil {
			eve.Logger.Error(fmt.Sprintf("Failed to get jobs for pipeline %d: %v", pipeline.ID, err))
			continue
		}

		for _, job := range jobs {
			jobInfo := JobInfo{
				ID:       job.ID,
				Name:     job.Name,
				Status:   job.Status,
				Stage:    job.Stage,
				Ref:      job.Ref,
				Pipeline: pipeline.ID,
			}
			allJobs = append(allJobs, jobInfo)
		}
	}

	// Log summary
	eve.Logger.Info(fmt.Sprintf("Found %d jobs for tag '%s' across %d pipelines", len(allJobs), tagName, len(pipelines)))

	// Log job details
	for _, job := range allJobs {
		eve.Logger.Info(fmt.Sprintf("Job ID: %d, Name: %s, Status: %s, Stage: %s, Pipeline: %d",
			job.ID, job.Name, job.Status, job.Stage, job.Pipeline))
	}

	return allJobs, nil
}

// GitlabListRunningJobsForTag lists only the currently running jobs for the given tag
func GitlabListRunningJobsForTag(url, token, projectID, tagName string) ([]JobInfo, error) {
	allJobs, err := GitlabListJobsForTag(url, token, projectID, tagName)
	if err != nil {
		return nil, err
	}

	var runningJobs []JobInfo
	for _, job := range allJobs {
		if job.Status == "running" || job.Status == "pending" {
			runningJobs = append(runningJobs, job)
		}
	}

	eve.Logger.Info(fmt.Sprintf("Found %d running/pending jobs for tag '%s'", len(runningJobs), tagName))
	return runningJobs, nil
}

// JobDetails represents detailed job information including error details
type JobDetails struct {
	ID             int        `json:"id"`
	Name           string     `json:"name"`
	Status         string     `json:"status"`
	Stage          string     `json:"stage"`
	Ref            string     `json:"ref"`
	PipelineID     int        `json:"pipeline_id"`
	CreatedAt      time.Time  `json:"created_at"`
	StartedAt      *time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	Duration       float64    `json:"duration"`
	QueuedDuration float64    `json:"queued_duration"`
	WebURL         string     `json:"web_url"`
	FailureReason  string     `json:"failure_reason"`
	ErrorMessage   string     `json:"error_message"`
	TraceLog       string     `json:"trace_log"`
}

// GitlabGetJobDetails gets detailed information about a specific job, including error details
func GitlabGetJobDetails(url, token, projectID string, jobID int) (*JobDetails, error) {
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(url+"/api/v4"))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Get job details
	job, _, err := client.Jobs.GetJob(projectID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job %d details: %w", jobID, err)
	}

	jobDetails := &JobDetails{
		ID:             job.ID,
		Name:           job.Name,
		Status:         job.Status,
		Stage:          job.Stage,
		Ref:            job.Ref,
		PipelineID:     job.Pipeline.ID,
		CreatedAt:      *job.CreatedAt,
		StartedAt:      job.StartedAt,
		FinishedAt:     job.FinishedAt,
		Duration:       job.Duration,
		QueuedDuration: job.QueuedDuration,
		WebURL:         job.WebURL,
		FailureReason:  job.FailureReason,
	}

	// Get job trace (log) if the job has failed or completed
	if job.Status == "failed" || job.Status == "success" || job.Status == "canceled" {
		trace, _, err := client.Jobs.GetTraceFile(projectID, jobID)
		if err != nil {
			eve.Logger.Warn(fmt.Sprintf("Could not retrieve trace for job %d: %v", jobID, err))
		} else {
			// Read all content from the *bytes.Reader
			traceBytes, err := io.ReadAll(trace)
			if err != nil {
				eve.Logger.Warn(fmt.Sprintf("Could not read trace content for job %d: %v", jobID, err))
			} else {
				jobDetails.TraceLog = string(traceBytes)

				// Extract error message from trace if job failed
				if job.Status == "failed" {
					jobDetails.ErrorMessage = extractErrorFromTrace(string(traceBytes))
				}
			}
		}
	}

	return jobDetails, nil
}

// GitlabDisplayJobState displays the detailed state of a job, with special formatting for errors
func GitlabDisplayJobState(url, token, projectID string, jobID int) error {
	jobDetails, err := GitlabGetJobDetails(url, token, projectID, jobID)
	if err != nil {
		return err
	}

	// Display basic job information
	eve.Logger.Info("=== Job Details ===")
	eve.Logger.Info(fmt.Sprintf("Job ID: %d", jobDetails.ID))
	eve.Logger.Info(fmt.Sprintf("Name: %s", jobDetails.Name))
	eve.Logger.Info(fmt.Sprintf("Status: %s", jobDetails.Status))
	eve.Logger.Info(fmt.Sprintf("Stage: %s", jobDetails.Stage))
	eve.Logger.Info(fmt.Sprintf("Ref: %s", jobDetails.Ref))
	eve.Logger.Info(fmt.Sprintf("Pipeline ID: %d", jobDetails.PipelineID))
	eve.Logger.Info(fmt.Sprintf("Created At: %s", jobDetails.CreatedAt.Format(time.RFC3339)))

	if jobDetails.StartedAt != nil {
		eve.Logger.Info(fmt.Sprintf("Started At: %s", jobDetails.StartedAt.Format(time.RFC3339)))
	}

	if jobDetails.FinishedAt != nil {
		eve.Logger.Info(fmt.Sprintf("Finished At: %s", jobDetails.FinishedAt.Format(time.RFC3339)))
		eve.Logger.Info(fmt.Sprintf("Duration: %.2f seconds", jobDetails.Duration))
	}

	eve.Logger.Info(fmt.Sprintf("Queued Duration: %.2f seconds", jobDetails.QueuedDuration))
	eve.Logger.Info(fmt.Sprintf("Web URL: %s", jobDetails.WebURL))

	// Display error information if job failed
	if jobDetails.Status == "failed" {
		eve.Logger.Error("=== ERROR DETAILS ===")

		if jobDetails.FailureReason != "" {
			eve.Logger.Error(fmt.Sprintf("Failure Reason: %s", jobDetails.FailureReason))
		}

		if jobDetails.ErrorMessage != "" {
			eve.Logger.Error(fmt.Sprintf("Error Message: %s", jobDetails.ErrorMessage))
		}

		if jobDetails.TraceLog != "" {
			eve.Logger.Error("=== JOB TRACE LOG ===")
			// Display last 50 lines of trace log for failed jobs
			lines := strings.Split(jobDetails.TraceLog, "\n")
			startLine := 0
			if len(lines) > 50 {
				startLine = len(lines) - 50
			}

			for i := startLine; i < len(lines); i++ {
				if strings.TrimSpace(lines[i]) != "" {
					eve.Logger.Error(lines[i])
				}
			}
		}
	}

	return nil
}

// extractErrorFromTrace extracts relevant error messages from the job trace log
func extractErrorFromTrace(trace string) string {
	lines := strings.Split(trace, "\n")
	var errorLines []string

	// Look for common error patterns
	errorKeywords := []string{"ERROR", "FAILED", "FATAL", "Exception", "error:", "Error:", "FAILURE"}

	for _, line := range lines {
		for _, keyword := range errorKeywords {
			if strings.Contains(line, keyword) {
				errorLines = append(errorLines, strings.TrimSpace(line))
				break
			}
		}
	}

	// Return the last few error lines (most relevant)
	if len(errorLines) > 0 {
		start := 0
		if len(errorLines) > 5 {
			start = len(errorLines) - 5
		}
		return strings.Join(errorLines[start:], "\n")
	}

	return "No specific error message found in trace log"
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
