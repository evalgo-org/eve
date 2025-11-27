package coordinator

import (
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// LogrusHook is a logrus hook that forwards log entries to when-v3.
// Use this to automatically forward all log messages from your service
// to the centralized log aggregation system in when-v3.
type LogrusHook struct {
	coordinator *Coordinator
	levels      []logrus.Level
	minLevel    logrus.Level
}

// NewLogrusHook creates a new logrus hook for forwarding logs to when-v3.
// The minLevel parameter specifies the minimum log level to forward (default: Info).
func NewLogrusHook(coordinator *Coordinator, minLevel logrus.Level) *LogrusHook {
	levels := make([]logrus.Level, 0)
	for _, level := range logrus.AllLevels {
		if level <= minLevel {
			levels = append(levels, level)
		}
	}

	return &LogrusHook{
		coordinator: coordinator,
		levels:      levels,
		minLevel:    minLevel,
	}
}

// Levels returns the log levels this hook fires for.
func (h *LogrusHook) Levels() []logrus.Level {
	return h.levels
}

// Fire is called when a log entry is made.
func (h *LogrusHook) Fire(entry *logrus.Entry) error {
	// Don't forward if coordinator is not connected
	if !h.coordinator.IsConnected() {
		return nil
	}

	// Convert logrus level to our level string
	level := logrusLevelToString(entry.Level)

	// Extract known fields
	logEntry := LogEntry{
		Timestamp: entry.Time,
		Level:     level,
		Message:   entry.Message,
		Fields:    make(map[string]interface{}),
	}

	// Extract workflow/action context from fields if present
	for k, v := range entry.Data {
		switch k {
		case "trace_id", "traceID", "traceId":
			if s, ok := v.(string); ok {
				logEntry.TraceID = s
			}
		case "span_id", "spanID", "spanId":
			if s, ok := v.(string); ok {
				logEntry.SpanID = s
			}
		case "workflow_id", "workflowID", "workflowId":
			if s, ok := v.(string); ok {
				logEntry.WorkflowID = s
			}
		case "action_id", "actionID", "actionId":
			if s, ok := v.(string); ok {
				logEntry.ActionID = s
			}
		case "correlation_id", "correlationID", "correlationId":
			if s, ok := v.(string); ok {
				logEntry.CorrelationID = s
			}
		default:
			// Store other fields as additional context
			logEntry.Fields[k] = v
		}
	}

	// Try to get source file and line
	if entry.HasCaller() && entry.Caller != nil {
		logEntry.SourceFile = entry.Caller.File
		logEntry.SourceLine = entry.Caller.Line
	} else {
		// Manually get caller info if not available
		if _, file, line, ok := runtime.Caller(7); ok {
			// Skip internal logrus/hook frames
			if !strings.Contains(file, "logrus") {
				logEntry.SourceFile = file
				logEntry.SourceLine = line
			}
		}
	}

	// Send the log entry asynchronously
	go h.coordinator.SendLog(logEntry)

	return nil
}

// logrusLevelToString converts a logrus level to our string format.
func logrusLevelToString(level logrus.Level) string {
	switch level {
	case logrus.TraceLevel, logrus.DebugLevel:
		return "debug"
	case logrus.InfoLevel:
		return "info"
	case logrus.WarnLevel:
		return "warn"
	case logrus.ErrorLevel:
		return "error"
	case logrus.FatalLevel, logrus.PanicLevel:
		return "fatal"
	default:
		return "info"
	}
}

// LogForwarder provides batched log forwarding for high-volume logging.
// It collects logs and sends them in batches to reduce WebSocket overhead.
type LogForwarder struct {
	coordinator   *Coordinator
	buffer        []LogEntry
	bufferSize    int
	flushInterval time.Duration
	flushChan     chan struct{}
	stopChan      chan struct{}
	doneChan      chan struct{}
}

// NewLogForwarder creates a new batched log forwarder.
// bufferSize is the maximum number of logs to buffer before flushing.
// flushInterval is how often to flush even if buffer isn't full.
func NewLogForwarder(coordinator *Coordinator, bufferSize int, flushInterval time.Duration) *LogForwarder {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}

	lf := &LogForwarder{
		coordinator:   coordinator,
		buffer:        make([]LogEntry, 0, bufferSize),
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		flushChan:     make(chan struct{}, 1),
		stopChan:      make(chan struct{}),
		doneChan:      make(chan struct{}),
	}

	go lf.run()
	return lf
}

// Log adds a log entry to the buffer.
func (lf *LogForwarder) Log(entry LogEntry) {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	lf.buffer = append(lf.buffer, entry)

	if len(lf.buffer) >= lf.bufferSize {
		select {
		case lf.flushChan <- struct{}{}:
		default:
		}
	}
}

// Flush immediately sends all buffered logs.
func (lf *LogForwarder) Flush() {
	select {
	case lf.flushChan <- struct{}{}:
	default:
	}
}

// Stop stops the log forwarder and flushes remaining logs.
func (lf *LogForwarder) Stop() {
	close(lf.stopChan)
	<-lf.doneChan
}

func (lf *LogForwarder) run() {
	defer close(lf.doneChan)

	ticker := time.NewTicker(lf.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-lf.stopChan:
			lf.doFlush()
			return
		case <-lf.flushChan:
			lf.doFlush()
		case <-ticker.C:
			lf.doFlush()
		}
	}
}

func (lf *LogForwarder) doFlush() {
	if len(lf.buffer) == 0 {
		return
	}

	// Copy buffer and clear
	logs := make([]LogEntry, len(lf.buffer))
	copy(logs, lf.buffer)
	lf.buffer = lf.buffer[:0]

	// Send batch
	lf.coordinator.SendLogBatch(logs)
}
