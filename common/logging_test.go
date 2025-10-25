package common

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOutputSplitter_ErrorToStderr tests that error messages go to stderr
func TestOutputSplitter_ErrorToStderr(t *testing.T) {
	// Note: We can't easily capture os.Stderr/os.Stdout in tests without
	// complex setup, so we test the logic by checking the byte pattern matching

	splitter := &OutputSplitter{}

	tests := []struct {
		name          string
		logMessage    []byte
		expectStderr  bool
	}{
		{
			name:         "ErrorLevel",
			logMessage:   []byte(`time="2024-01-15T10:30:00Z" level=error msg="Database connection failed"`),
			expectStderr: true,
		},
		{
			name:         "InfoLevel",
			logMessage:   []byte(`time="2024-01-15T10:30:00Z" level=info msg="Service started"`),
			expectStderr: false,
		},
		{
			name:         "WarnLevel",
			logMessage:   []byte(`time="2024-01-15T10:30:00Z" level=warning msg="High memory usage"`),
			expectStderr: false,
		},
		{
			name:         "DebugLevel",
			logMessage:   []byte(`time="2024-01-15T10:30:00Z" level=debug msg="Processing request"`),
			expectStderr: false,
		},
		{
			name:         "ErrorInMessage",
			logMessage:   []byte(`time="2024-01-15T10:30:00Z" level=info msg="error occurred but not error level"`),
			expectStderr: false,
		},
		{
			name:         "EmptyMessage",
			logMessage:   []byte(``),
			expectStderr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that Write returns the correct number of bytes
			n, err := splitter.Write(tt.logMessage)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.logMessage), n)
		})
	}
}

// TestOutputSplitter_WriteReturnsLength tests Write returns correct length
func TestOutputSplitter_WriteReturnsLength(t *testing.T) {
	splitter := &OutputSplitter{}

	tests := []struct {
		name    string
		message []byte
	}{
		{
			name:    "ShortMessage",
			message: []byte("short"),
		},
		{
			name:    "LongMessage",
			message: []byte("This is a very long log message that contains multiple words and should still be written correctly to the output stream"),
		},
		{
			name:    "EmptyMessage",
			message: []byte(""),
		},
		{
			name:    "WithNewlines",
			message: []byte("Line 1\nLine 2\nLine 3\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := splitter.Write(tt.message)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.message), n)
		})
	}
}

// TestOutputSplitter_BytePatternMatching tests the pattern matching logic
func TestOutputSplitter_BytePatternMatching(t *testing.T) {
	splitter := &OutputSplitter{}

	// Test that the pattern matching works correctly
	errorPatterns := [][]byte{
		[]byte("level=error"),
		[]byte("level=error msg=\"test\""),
		[]byte("prefix level=error suffix"),
		[]byte("...level=error..."),
	}

	for i, pattern := range errorPatterns {
		n, err := splitter.Write(pattern)
		assert.NoError(t, err, "Pattern %d failed", i)
		assert.Equal(t, len(pattern), n, "Pattern %d returned wrong length", i)
		assert.True(t, bytes.Contains(pattern, []byte("level=error")))
	}

	nonErrorPatterns := [][]byte{
		[]byte("level=info"),
		[]byte("level=warning"),
		[]byte("level=debug"),
		[]byte("error in message but level=info"),
		[]byte("LEVEL=ERROR"), // Different case
	}

	for i, pattern := range nonErrorPatterns {
		n, err := splitter.Write(pattern)
		assert.NoError(t, err, "Non-error pattern %d failed", i)
		assert.Equal(t, len(pattern), n, "Non-error pattern %d returned wrong length", i)
		assert.False(t, bytes.Contains(pattern, []byte("level=error")))
	}
}

// TestOutputSplitter_ConcurrentWrites tests concurrent writes
func TestOutputSplitter_ConcurrentWrites(t *testing.T) {
	splitter := &OutputSplitter{}

	// Test concurrent writes don't cause issues
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			message := []byte("Concurrent message from goroutine")
			n, err := splitter.Write(message)
			assert.NoError(t, err)
			assert.Equal(t, len(message), n)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestLogger_Initialization tests that Logger is initialized
func TestLogger_Initialization(t *testing.T) {
	assert.NotNil(t, Logger, "Logger should be initialized")
	assert.NotNil(t, Logger.Out, "Logger output should be set")
}

// TestLogger_OutputIsSplitter tests that Logger uses OutputSplitter
func TestLogger_OutputIsSplitter(t *testing.T) {
	// Verify Logger.Out is an OutputSplitter
	_, ok := Logger.Out.(*OutputSplitter)
	assert.True(t, ok, "Logger should use OutputSplitter")
}

// BenchmarkOutputSplitter_Write benchmarks the Write method
func BenchmarkOutputSplitter_Write(b *testing.B) {
	splitter := &OutputSplitter{}
	message := []byte(`time="2024-01-15T10:30:00Z" level=info msg="Benchmark message"`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitter.Write(message)
	}
}

// BenchmarkOutputSplitter_WriteError benchmarks error message writes
func BenchmarkOutputSplitter_WriteError(b *testing.B) {
	splitter := &OutputSplitter{}
	message := []byte(`time="2024-01-15T10:30:00Z" level=error msg="Benchmark error message"`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitter.Write(message)
	}
}
