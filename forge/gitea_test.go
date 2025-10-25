package forge

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestJobInfo tests the JobInfo struct
func TestJobInfo(t *testing.T) {
	job := JobInfo{
		ID:       123,
		Name:     "build-job",
		Status:   "success",
		Stage:    "build",
		Ref:      "main",
		Pipeline: 456,
	}

	assert.Equal(t, 123, job.ID)
	assert.Equal(t, "build-job", job.Name)
	assert.Equal(t, "success", job.Status)
	assert.Equal(t, "build", job.Stage)
	assert.Equal(t, "main", job.Ref)
	assert.Equal(t, 456, job.Pipeline)
}

// TestJobInfo_EmptyValues tests JobInfo with empty values
func TestJobInfo_EmptyValues(t *testing.T) {
	job := JobInfo{}

	assert.Equal(t, 0, job.ID)
	assert.Empty(t, job.Name)
	assert.Empty(t, job.Status)
	assert.Empty(t, job.Stage)
	assert.Empty(t, job.Ref)
	assert.Equal(t, 0, job.Pipeline)
}

// TestJobDetails tests the JobDetails struct
func TestJobDetails(t *testing.T) {
	createdAt := time.Now().Add(-10 * time.Minute)
	startedAt := time.Now().Add(-8 * time.Minute)
	finishedAt := time.Now().Add(-2 * time.Minute)

	details := JobDetails{
		ID:             789,
		Name:           "test-job",
		Status:         "failed",
		Stage:          "test",
		Ref:            "feature-branch",
		PipelineID:     101,
		CreatedAt:      createdAt,
		StartedAt:      &startedAt,
		FinishedAt:     &finishedAt,
		Duration:       360.5,
		QueuedDuration: 120.0,
		WebURL:         "https://gitlab.example.com/jobs/789",
		FailureReason:  "script_failure",
		ErrorMessage:   "Test failed",
		TraceLog:       "Error: assertion failed",
	}

	assert.Equal(t, 789, details.ID)
	assert.Equal(t, "test-job", details.Name)
	assert.Equal(t, "failed", details.Status)
	assert.Equal(t, "test", details.Stage)
	assert.Equal(t, "feature-branch", details.Ref)
	assert.Equal(t, 101, details.PipelineID)
	assert.Equal(t, createdAt, details.CreatedAt)
	assert.NotNil(t, details.StartedAt)
	assert.NotNil(t, details.FinishedAt)
	assert.Equal(t, 360.5, details.Duration)
	assert.Equal(t, 120.0, details.QueuedDuration)
	assert.Equal(t, "https://gitlab.example.com/jobs/789", details.WebURL)
	assert.Equal(t, "script_failure", details.FailureReason)
	assert.Equal(t, "Test failed", details.ErrorMessage)
	assert.Equal(t, "Error: assertion failed", details.TraceLog)
}

// TestJobDetails_NilTimestamps tests JobDetails with nil timestamps
func TestJobDetails_NilTimestamps(t *testing.T) {
	details := JobDetails{
		ID:         123,
		Name:       "pending-job",
		Status:     "pending",
		CreatedAt:  time.Now(),
		StartedAt:  nil,
		FinishedAt: nil,
	}

	assert.Nil(t, details.StartedAt)
	assert.Nil(t, details.FinishedAt)
	assert.Equal(t, "pending", details.Status)
}

// TestJobDetails_SuccessfulJob tests a successful job details
func TestJobDetails_SuccessfulJob(t *testing.T) {
	createdAt := time.Now().Add(-30 * time.Minute)
	startedAt := time.Now().Add(-25 * time.Minute)
	finishedAt := time.Now().Add(-5 * time.Minute)

	details := JobDetails{
		ID:             999,
		Name:           "deploy-job",
		Status:         "success",
		Stage:          "deploy",
		Ref:            "main",
		PipelineID:     555,
		CreatedAt:      createdAt,
		StartedAt:      &startedAt,
		FinishedAt:     &finishedAt,
		Duration:       1200.0,
		QueuedDuration: 300.0,
		WebURL:         "https://gitlab.example.com/jobs/999",
		FailureReason:  "",
		ErrorMessage:   "",
		TraceLog:       "Deployment successful",
	}

	assert.Equal(t, "success", details.Status)
	assert.Empty(t, details.FailureReason)
	assert.Empty(t, details.ErrorMessage)
	assert.Contains(t, details.TraceLog, "successful")
}

// TestJobInfo_MultipleJobs tests multiple job infos
func TestJobInfo_MultipleJobs(t *testing.T) {
	jobs := []JobInfo{
		{ID: 1, Name: "build", Status: "success", Stage: "build", Ref: "main", Pipeline: 100},
		{ID: 2, Name: "test", Status: "failed", Stage: "test", Ref: "main", Pipeline: 100},
		{ID: 3, Name: "deploy", Status: "pending", Stage: "deploy", Ref: "main", Pipeline: 100},
	}

	assert.Len(t, jobs, 3)
	assert.Equal(t, "success", jobs[0].Status)
	assert.Equal(t, "failed", jobs[1].Status)
	assert.Equal(t, "pending", jobs[2].Status)
}

// TestJobDetails_TimeDurations tests duration calculations
func TestJobDetails_TimeDurations(t *testing.T) {
	details := JobDetails{
		Duration:       300.5,
		QueuedDuration: 60.25,
	}

	assert.Equal(t, 300.5, details.Duration)
	assert.Equal(t, 60.25, details.QueuedDuration)
	assert.Greater(t, details.Duration, details.QueuedDuration)
}
