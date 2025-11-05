package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"eve.evalgo.org/semantic"
)

// CommandExecutor executes shell commands
type CommandExecutor struct {
	Shell string
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{
		Shell: "/bin/sh",
	}
}

// Name returns the executor's identifier
func (e *CommandExecutor) Name() string {
	return "command"
}

// CanHandle determines if this executor can process the action
func (e *CommandExecutor) CanHandle(action *semantic.SemanticScheduledAction) bool {
	if action == nil || action.Object == nil {
		return false
	}

	// Check if it's a command-based action
	if action.Object.ContentUrl != "" {
		return strings.HasPrefix(action.Object.ContentUrl, "exec://") ||
			strings.HasPrefix(action.Object.ContentUrl, "command://") ||
			strings.HasPrefix(action.Object.ContentUrl, "shell://")
	}

	return false
}

// Execute runs the command and returns the result
func (e *CommandExecutor) Execute(ctx context.Context, action *semantic.SemanticScheduledAction) (*Result, error) {
	result := &Result{
		StartTime: time.Now(),
		Status:    StatusRunning,
		Metadata:  make(map[string]interface{}),
	}

	if action == nil || action.Object == nil {
		result.Status = StatusFailed
		result.Error = &ExecutionError{
			Message: "action or action.Object is nil",
			Code:    "INVALID_ACTION",
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, result.Error
	}

	// Extract command from URL
	command := strings.TrimPrefix(action.Object.ContentUrl, "exec://")
	command = strings.TrimPrefix(command, "command://")
	command = strings.TrimPrefix(command, "shell://")

	if command == "" {
		result.Status = StatusFailed
		result.Error = &ExecutionError{
			Message: "empty command",
			Code:    "INVALID_COMMAND",
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, result.Error
	}

	result.Metadata["command"] = command
	result.Metadata["shell"] = e.Shell

	// Create command
	cmd := exec.CommandContext(ctx, e.Shell, "-c", command)

	// Execute command
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	result.Metadata["output_length"] = len(output)

	if err != nil {
		result.Status = StatusFailed
		result.Error = &ExecutionError{
			Message: fmt.Sprintf("command execution failed: %v", err),
			Code:    "COMMAND_ERROR",
			Details: map[string]interface{}{
				"command": command,
				"output":  string(output),
			},
		}

		// Check if it's an exit error to get the exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Metadata["exit_code"] = exitErr.ExitCode()
		}

		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, result.Error
	}

	result.Status = StatusCompleted
	result.Metadata["exit_code"] = 0
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}
