package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShellExecute_Success tests successful command execution
func TestShellExecute_Success(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		expectedOutput string
	}{
		{
			name:           "EchoCommand",
			command:        "echo 'Hello World'",
			expectedOutput: "Hello World\n",
		},
		{
			name:           "PWDCommand",
			command:        "pwd",
			expectedOutput: "", // Output varies, just check no error
		},
		{
			name:           "DateCommand",
			command:        "date +%Y",
			expectedOutput: "", // Output varies, just check no error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := ShellExecute(tt.command)
			require.NoError(t, err)
			assert.NotEmpty(t, output)

			if tt.expectedOutput != "" {
				assert.Equal(t, tt.expectedOutput, output)
			}
		})
	}
}

// TestShellExecute_Failure tests command execution failures
func TestShellExecute_Failure(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		expectedError string
	}{
		{
			name:          "NonExistentCommand",
			command:       "nonexistentcommand123",
			expectedError: "command failed",
		},
		{
			name:          "InvalidSyntax",
			command:       "ls --invalid-flag-xyz",
			expectedError: "command failed",
		},
		{
			name:          "FailingCommand",
			command:       "false", // always returns exit code 1
			expectedError: "command failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := ShellExecute(tt.command)
			assert.Error(t, err)
			assert.Empty(t, output)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// TestShellExecute_OutputCapture tests that stdout is properly captured
func TestShellExecute_OutputCapture(t *testing.T) {
	output, err := ShellExecute("echo -n 'test output'")
	require.NoError(t, err)
	assert.Equal(t, "test output", output)
}

// TestShellExecute_StderrCapture tests that stderr is captured in error
func TestShellExecute_StderrCapture(t *testing.T) {
	output, err := ShellExecute("echo 'error message' >&2 && false")
	assert.Error(t, err)
	assert.Empty(t, output)
	assert.Contains(t, err.Error(), "stderr: error message")
}

// TestShellExecute_MultilineOutput tests multiline output handling
func TestShellExecute_MultilineOutput(t *testing.T) {
	output, err := ShellExecute("echo 'line1'; echo 'line2'; echo 'line3'")
	require.NoError(t, err)
	assert.Contains(t, output, "line1")
	assert.Contains(t, output, "line2")
	assert.Contains(t, output, "line3")
}

// TestShellExecute_PipeCommands tests command piping
func TestShellExecute_PipeCommands(t *testing.T) {
	output, err := ShellExecute("echo 'hello world' | tr '[:lower:]' '[:upper:]'")
	require.NoError(t, err)
	assert.Contains(t, output, "HELLO WORLD")
}

// TestShellExecute_EmptyCommand tests empty command handling
func TestShellExecute_EmptyCommand(t *testing.T) {
	output, err := ShellExecute("")
	require.NoError(t, err)
	assert.Empty(t, output)
}

// TestShellSudoExecute_Format tests sudo command construction
func TestShellSudoExecute_Format(t *testing.T) {
	// We can't actually test sudo without proper setup
	// but we can test that it calls ShellExecute correctly
	// The sudo command may succeed or fail depending on system setup
	_, err := ShellSudoExecute("wrongpassword", "echo test")
	// If there's an error, it should be from ShellExecute
	if err != nil {
		assert.Contains(t, err.Error(), "command failed")
	}
	// If no error, sudo succeeded (unlikely but possible in some test environments)
}

// BenchmarkShellExecute benchmarks command execution
func BenchmarkShellExecute(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ShellExecute("echo test")
	}
}
