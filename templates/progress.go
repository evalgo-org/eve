package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

// ProcessState represents the possible states of a process
type ProcessState string

const (
	StateStarted    ProcessState = "started"
	StateRunning    ProcessState = "running"
	StateSuccessful ProcessState = "successful"
	StateFailed     ProcessState = "failed"
)

// HistoryEntry represents a single state change in the process history
type HistoryEntry struct {
	State     ProcessState `json:"state"`
	Timestamp time.Time    `json:"timestamp"`
}

// Process represents a single process with its metadata and history
type Process struct {
	ID        string         `json:"_id"`
	Rev       string         `json:"_rev"`
	ProcessID string         `json:"process_id"`
	State     ProcessState   `json:"state"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	History   []HistoryEntry `json:"history"`
}

// ProcessList represents the root structure containing all processes
type ProcessList struct {
	Count     int       `json:"count"`
	Processes []Process `json:"processes"`
}

// Helper methods for template formatting

// FormatTimestamp returns a formatted timestamp string for display
func (h HistoryEntry) FormatTimestamp() string {
	return h.Timestamp.Format("2006-01-02 15:04:05")
}

// FormatCreatedAt returns a formatted creation timestamp
func (p Process) FormatCreatedAt() string {
	return p.CreatedAt.Format("2006-01-02 15:04:05")
}

// FormatUpdatedAt returns a formatted update timestamp
func (p Process) FormatUpdatedAt() string {
	return p.UpdatedAt.Format("2006-01-02 15:04:05")
}

// Duration returns the total duration of the process
func (p Process) Duration() time.Duration {
	if len(p.History) == 0 {
		return 0
	}

	firstEntry := p.History[0]
	lastEntry := p.History[len(p.History)-1]

	return lastEntry.Timestamp.Sub(firstEntry.Timestamp)
}

// FormatDuration returns a human-readable duration string
func (p Process) FormatDuration() string {
	duration := p.Duration()

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// IsCompleted returns true if the process is in a final state
func (p Process) IsCompleted() bool {
	return p.State == StateSuccessful || p.State == StateFailed
}

// IsRunning returns true if the process is currently running
func (p Process) IsRunning() bool {
	return p.State == StateRunning
}

// GetProgressPercentage returns a rough progress percentage based on state
func (p Process) GetProgressPercentage() int {
	switch p.State {
	case StateStarted:
		return 25
	case StateRunning:
		return 75
	case StateSuccessful:
		return 100
	case StateFailed:
		return 100
	default:
		return 0
	}
}

// GetStateIcon returns an emoji icon for the current state
func (p Process) GetStateIcon() string {
	switch p.State {
	case StateStarted:
		return "🚀"
	case StateRunning:
		return "⚡"
	case StateSuccessful:
		return "✅"
	case StateFailed:
		return "❌"
	default:
		return "❓"
	}
}

// JSON marshaling/unmarshaling helpers

// ToJSON converts the ProcessList to JSON string
func (pl *ProcessList) ToJSON() ([]byte, error) {
	return json.MarshalIndent(pl, "", "  ")
}

// FromJSON creates a ProcessList from JSON data
func FromJSON(data []byte) (*ProcessList, error) {
	var pl ProcessList
	err := json.Unmarshal(data, &pl)
	return &pl, err
}

// Constructor functions for creating processes

// NewProcess creates a new process with initial state
func NewProcess(processID string) *Process {
	now := time.Now()
	return &Process{
		ID:        fmt.Sprintf("process_%s", processID),
		ProcessID: processID,
		State:     StateStarted,
		CreatedAt: now,
		UpdatedAt: now,
		History: []HistoryEntry{
			{
				State:     StateStarted,
				Timestamp: now,
			},
		},
	}
}

// UpdateState adds a new state to the process history
func (p *Process) UpdateState(newState ProcessState) {
	now := time.Now()
	p.State = newState
	p.UpdatedAt = now

	p.History = append(p.History, HistoryEntry{
		State:     newState,
		Timestamp: now,
	})
}

// Template function map for additional formatting in templates
var templateFuncs = template.FuncMap{
	"formatTime": func(t time.Time) string {
		return t.Format("2006-01-02 15:04:05")
	},
	"formatDuration": func(start, end time.Time) string {
		duration := end.Sub(start)
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		seconds := int(duration.Seconds()) % 60

		if hours > 0 {
			return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
		} else if minutes > 0 {
			return fmt.Sprintf("%dm %ds", minutes, seconds)
		}
		return fmt.Sprintf("%ds", seconds)
	},
	"stateIcon": func(state ProcessState) string {
		switch state {
		case StateStarted:
			return "🚀"
		case StateRunning:
			return "⚡"
		case StateSuccessful:
			return "✅"
		case StateFailed:
			return "❌"
		default:
			return "❓"
		}
	},
}

// Example usage function
func main() {
	// Your JSON data
	jsonData := `{
		"count": 2,
		"processes": [
			{
				"_id": "process_0123456789",
				"_rev": "4-c71f1347463fe0a951c38d032fb3a832",
				"process_id": "0123456789",
				"state": "successful",
				"created_at": "2025-08-26T20:29:44.01016197+02:00",
				"updated_at": "2025-08-26T20:37:15.232752777+02:00",
				"history": [
					{
						"state": "started",
						"timestamp": "2025-08-26T20:29:44.01016197+02:00"
					},
					{
						"state": "running",
						"timestamp": "2025-08-26T20:36:48.804930215+02:00"
					},
					{
						"state": "running",
						"timestamp": "2025-08-26T20:36:55.976230255+02:00"
					},
					{
						"state": "successful",
						"timestamp": "2025-08-26T20:37:15.232752777+02:00"
					}
				]
			},
			{
				"_id": "process_1234567890",
				"_rev": "3-049c68cedf3a8046d26b219f28119157",
				"process_id": "1234567890",
				"state": "failed",
				"created_at": "2025-08-26T20:08:54.165061549+02:00",
				"updated_at": "2025-08-26T20:29:20.10776054+02:00",
				"history": [
					{
						"state": "started",
						"timestamp": "2025-08-26T20:08:54.165061549+02:00"
					},
					{
						"state": "running",
						"timestamp": "2025-08-26T20:23:00.097861464+02:00"
					},
					{
						"state": "failed",
						"timestamp": "2025-08-26T20:29:20.10776054+02:00"
					}
				]
			}
		]
	}`

	// Parse JSON into struct
	processList, err := FromJSON([]byte(jsonData))
	if err != nil {
		log.Fatal("Error parsing JSON:", err)
	}

	// Print some info
	log.Printf("Loaded %d processes", processList.Count)
	for _, process := range processList.Processes {
		log.Printf("Process %s: %s (Duration: %s)",
			process.ProcessID,
			process.State,
			process.FormatDuration())
	}

	// Set up HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Try to find the template file (could be flow.gohtml or process-progress.html)
		var tmpl *template.Template
		var err error

		// First try flow.gohtml (based on your error message)
		if tmpl, err = template.New("flow.gohtml").Funcs(templateFuncs).ParseFiles("flow.gohtml"); err != nil {
			// Fallback to process-progress.html
			// if tmpl, err = template.New("process-progress.html").Funcs(templateFuncs).ParseFiles("process-progress.html"); err != nil {
			// 	http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
			// 	log.Printf("Template parsing error: %v", err)
			// 	return
			// }
		}

		fmt.Println(processList.Count)

		// Execute template with data (dereference the pointer)
		if err := tmpl.Execute(w, *processList); err != nil {
			http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Template execution error: %v", err)
			return
		}
	})

	// API endpoint to return JSON data
	http.HandleFunc("/api/processes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		jsonBytes, err := processList.ToJSON()
		if err != nil {
			http.Error(w, "JSON encoding error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(jsonBytes)
	})

	// Start server
	log.Println("Server starting on :8080")
	log.Println("View processes at: http://localhost:8080")
	log.Println("API endpoint at: http://localhost:8080/api/processes")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
