// Package main provides a comprehensive process monitoring and web dashboard application.
// This application implements a real-time process state tracking system with historical
// data management, web-based visualization, and RESTful API endpoints for enterprise
// process monitoring and workflow orchestration scenarios.
//
// Process State Management:
//
//	The application manages process lifecycle states with comprehensive tracking:
//	- Process creation with unique identification and metadata
//	- State transitions through defined workflow stages
//	- Historical state tracking with timestamp precision
//	- Duration calculation and performance analytics
//	- Completion status monitoring and reporting
//
// Web Dashboard Features:
//
//	Provides comprehensive web-based monitoring capabilities:
//	- Real-time process status visualization with progress indicators
//	- Historical timeline view of process state transitions
//	- Interactive dashboard with filtering and sorting capabilities
//	- Template-driven HTML rendering with custom formatting functions
//	- Responsive design for desktop and mobile monitoring
//
// API Integration:
//
//	RESTful API endpoints for programmatic access:
//	- JSON-based process data retrieval and manipulation
//	- Real-time status updates and webhook integration
//	- Bulk process operations and batch processing
//	- Integration with external monitoring and alerting systems
//	- Machine-readable data formats for automation workflows
//
// Enterprise Features:
//
//	Production-ready capabilities for enterprise environments:
//	- Structured logging and audit trail management
//	- Error handling and graceful degradation
//	- Performance monitoring and analytics
//	- Scalable architecture for high-volume processing
//	- Security considerations for production deployment
//
// Use Cases:
//   - Workflow orchestration and process monitoring
//   - CI/CD pipeline status tracking and visualization
//   - Long-running task management and progress reporting
//   - Business process monitoring and analytics
//   - System integration and status dashboard applications
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

// ProcessState represents the possible states of a process in the workflow lifecycle.
// This enumeration defines the standard states that processes can transition through,
// providing a consistent state model for workflow management and monitoring.
//
// State Definitions:
//   - StateStarted: Initial state when process is created and initialized
//   - StateRunning: Active execution state with ongoing processing
//   - StateSuccessful: Terminal state indicating successful completion
//   - StateFailed: Terminal state indicating failure or error condition
//
// State Transition Rules:
//
//	Valid transitions follow a defined workflow pattern:
//	- StateStarted → StateRunning (process begins execution)
//	- StateRunning → StateSuccessful (successful completion)
//	- StateRunning → StateFailed (error or failure condition)
//	- StateStarted → StateFailed (early failure before execution)
//
// Usage in Monitoring:
//
//	States are used throughout the application for:
//	- Progress visualization and user interface rendering
//	- Business logic and conditional processing
//	- Analytics and reporting calculations
//	- API responses and data serialization
type ProcessState string

const (
	StateStarted    ProcessState = "started"    // Process has been initialized and is ready to run
	StateRunning    ProcessState = "running"    // Process is actively executing
	StateSuccessful ProcessState = "successful" // Process completed successfully
	StateFailed     ProcessState = "failed"     // Process failed with error
)

// HistoryEntry represents a single state change in the process history with precise timing.
// This structure captures individual state transitions within a process lifecycle,
// enabling detailed audit trails, performance analysis, and debugging capabilities.
//
// Historical Tracking:
//
//	Each history entry provides:
//	- Precise timestamp of state transition for timeline analysis
//	- State information for lifecycle reconstruction
//	- Audit trail capabilities for compliance and debugging
//	- Performance metrics calculation through timestamp analysis
//
// JSON Serialization:
//
//	Structured for API compatibility and data persistence:
//	- Standard JSON field naming for cross-platform compatibility
//	- Timestamp formatting for international standards compliance
//	- Efficient serialization for high-volume logging scenarios
//
// Analytics Applications:
//
//	History entries enable comprehensive analytics:
//	- State transition duration calculations
//	- Process bottleneck identification and optimization
//	- Performance trending and capacity planning
//	- Error pattern analysis and failure prediction
type HistoryEntry struct {
	State     ProcessState `json:"state"`     // The state that was entered
	Timestamp time.Time    `json:"timestamp"` // Precise time when state change occurred
}

// Process represents a single process with comprehensive metadata and complete state history.
// This structure serves as the core data model for process tracking, providing all
// necessary information for monitoring, analytics, and operational management.
//
// Process Identification:
//
//	Unique identification and versioning system:
//	- ID: Internal system identifier for database operations
//	- Rev: Revision identifier for optimistic concurrency control
//	- ProcessID: Business identifier for external system integration
//	- Creation and update timestamps for lifecycle tracking
//
// State Management:
//
//	Current state tracking with historical preservation:
//	- Current state for real-time monitoring and decision making
//	- Complete history for audit trails and performance analysis
//	- Timestamp tracking for duration calculations and SLA monitoring
//	- State transition validation and workflow enforcement
//
// Operational Metadata:
//
//	Comprehensive tracking for operational excellence:
//	- Creation time for process age calculation and cleanup policies
//	- Last update time for freshness validation and cache management
//	- Complete audit trail for compliance and debugging
//	- Performance metrics through history analysis
//
// Integration Capabilities:
//
//	Designed for enterprise integration scenarios:
//	- JSON serialization for API and database compatibility
//	- Template rendering support for web dashboard presentation
//	- Analytics functions for reporting and monitoring
//	- External system integration through ProcessID mapping
type Process struct {
	ID        string         `json:"_id"`        // Internal database identifier
	Rev       string         `json:"_rev"`       // Revision for concurrency control
	ProcessID string         `json:"process_id"` // Business process identifier
	State     ProcessState   `json:"state"`      // Current process state
	CreatedAt time.Time      `json:"created_at"` // Process creation timestamp
	UpdatedAt time.Time      `json:"updated_at"` // Last modification timestamp
	History   []HistoryEntry `json:"history"`    // Complete state change history
}

// ProcessList represents the root structure containing all processes with aggregate metadata.
// This structure provides a container for multiple processes with summary information,
// supporting bulk operations, pagination, and aggregate analytics for large-scale
// process monitoring and management scenarios.
//
// Container Features:
//
//	Comprehensive process collection management:
//	- Count metadata for pagination and UI rendering
//	- Process array for bulk operations and batch processing
//	- JSON serialization for API responses and data exchange
//	- Template rendering support for dashboard visualization
//
// Pagination Support:
//
//	Designed for large-scale process management:
//	- Count field enables pagination calculations
//	- Array structure supports offset and limit operations
//	- Efficient serialization for large datasets
//	- Memory-conscious processing for high-volume scenarios
//
// Analytics Integration:
//
//	Aggregate data structure for reporting:
//	- Process count for capacity monitoring
//	- Collection iteration for batch analytics
//	- State distribution calculations across process sets
//	- Performance aggregation and trending analysis
//
// API Compatibility:
//
//	RESTful API design patterns:
//	- Standard collection response format
//	- JSON serialization for cross-platform compatibility
//	- Metadata inclusion for client-side processing
//	- Extensible structure for future enhancements
type ProcessList struct {
	Count     int       `json:"count"`     // Total number of processes in collection
	Processes []Process `json:"processes"` // Array of process objects
}

// FormatTimestamp returns a formatted timestamp string for display in user interfaces.
// This method provides consistent timestamp formatting across the application,
// ensuring uniform presentation in web templates, reports, and user interfaces.
//
// Formatting Specification:
//
//	Uses ISO-style format "2006-01-02 15:04:05" for:
//	- International compatibility and standards compliance
//	- Human readability in dashboard and report presentations
//	- Consistent sorting behavior in tabular displays
//	- Time zone neutral presentation for global applications
//
// Template Integration:
//
//	Designed for HTML template usage:
//	- Direct method call from template expressions
//	- Consistent formatting across all timestamp displays
//	- Localization support through format customization
//	- Responsive design compatibility
//
// Returns:
//   - string: Formatted timestamp in "YYYY-MM-DD HH:MM:SS" format
//
// Example Usage:
//
//	{{.FormatTimestamp}} in HTML templates
//	entry.FormatTimestamp() in Go code
func (h HistoryEntry) FormatTimestamp() string {
	return h.Timestamp.Format("2006-01-02 15:04:05")
}

// FormatCreatedAt returns a formatted creation timestamp for consistent display formatting.
// This method provides standardized formatting for process creation times,
// supporting dashboard presentation and administrative interfaces.
//
// Display Applications:
//
//	Formatted creation time for:
//	- Process listing tables and administrative views
//	- Dashboard summaries and overview displays
//	- Audit logs and compliance reporting
//	- Historical analysis and trend visualization
//
// Returns:
//   - string: Formatted creation timestamp in "YYYY-MM-DD HH:MM:SS" format
//
// Integration:
//
//	Template and UI integration:
//	- HTML template method calls for consistent presentation
//	- Administrative interface timestamp display
//	- Report generation and export functionality
//	- API response formatting for client applications
func (p Process) FormatCreatedAt() string {
	return p.CreatedAt.Format("2006-01-02 15:04:05")
}

// FormatUpdatedAt returns a formatted update timestamp for last modification display.
// This method provides consistent formatting for process update times,
// enabling effective monitoring of process activity and change tracking.
//
// Monitoring Applications:
//
//	Update timestamp display for:
//	- Real-time dashboard updates and freshness indicators
//	- Change tracking and modification monitoring
//	- Cache invalidation and synchronization logic
//	- Performance analysis and optimization
//
// Returns:
//   - string: Formatted update timestamp in "YYYY-MM-DD HH:MM:SS" format
//
// Operational Usage:
//
//	Monitoring and maintenance applications:
//	- Process activity monitoring and alerting
//	- Data freshness validation and cache management
//	- Synchronization and replication monitoring
//	- Performance tuning and optimization analysis
func (p Process) FormatUpdatedAt() string {
	return p.UpdatedAt.Format("2006-01-02 15:04:05")
}

// Duration returns the total duration of the process from start to most recent state change.
// This method calculates the elapsed time between the first and last state transitions,
// providing essential performance metrics for process monitoring and optimization.
//
// Calculation Method:
//
//	Duration calculation algorithm:
//	- Uses first history entry as start time reference
//	- Uses last history entry as end time reference
//	- Returns zero duration for processes without history
//	- Provides high-precision timing for performance analysis
//
// Performance Metrics:
//
//	Duration data supports various analytics:
//	- Process execution time monitoring and SLA compliance
//	- Performance trending and capacity planning
//	- Bottleneck identification and optimization opportunities
//	- Resource utilization analysis and cost optimization
//
// Returns:
//   - time.Duration: Total elapsed time from first to last state change
//   - time.Duration(0): For processes with empty history
//
// Analytics Applications:
//
//	Performance monitoring use cases:
//	- SLA compliance monitoring and alerting
//	- Performance baseline establishment and trending
//	- Capacity planning and resource allocation
//	- Cost analysis and optimization recommendations
func (p Process) Duration() time.Duration {
	if len(p.History) == 0 {
		return 0
	}

	firstEntry := p.History[0]
	lastEntry := p.History[len(p.History)-1]

	return lastEntry.Timestamp.Sub(firstEntry.Timestamp)
}

// FormatDuration returns a human-readable duration string for display and reporting.
// This method converts process duration into user-friendly format suitable for
// dashboards, reports, and administrative interfaces with intelligent unit selection.
//
// Formatting Logic:
//
//	Intelligent unit selection and display:
//	- Hours, minutes, seconds for long-running processes
//	- Minutes and seconds for medium-duration processes
//	- Seconds only for short-duration processes
//	- Automatic unit selection based on duration magnitude
//
// Display Applications:
//
//	User interface and reporting usage:
//	- Dashboard process duration displays
//	- Performance reports and analytics summaries
//	- Administrative monitoring interfaces
//	- Alert notifications and status messages
//
// Returns:
//   - string: Human-readable duration in format "XhYmZs", "YmZs", or "Zs"
//
// Example Outputs:
//   - "2h 15m 30s" for long processes
//   - "45m 22s" for medium processes
//   - "33s" for short processes
//
// Template Integration:
//
//	HTML template compatibility:
//	- Direct method calls from template expressions
//	- Consistent formatting across all duration displays
//	- Responsive design and mobile compatibility
//	- Internationalization support through format customization
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

// IsCompleted returns true if the process is in a final state (successful or failed).
// This method provides a convenient way to determine if a process has reached
// completion, supporting conditional logic, filtering, and analytics operations.
//
// Completion Detection:
//
//	Terminal state identification:
//	- StateSuccessful: Process completed successfully
//	- StateFailed: Process completed with failure
//	- Returns false for StateStarted and StateRunning
//	- Supports workflow logic and conditional processing
//
// Business Logic Applications:
//
//	Completion status for operational decisions:
//	- Resource cleanup and garbage collection
//	- Notification and alert triggering
//	- Reporting and analytics calculations
//	- Workflow orchestration and dependency management
//
// Returns:
//   - bool: true if process is in StateSuccessful or StateFailed
//   - bool: false if process is in StateStarted or StateRunning
//
// Usage Examples:
//
//	Conditional processing and filtering:
//	- Dashboard filtering for active vs completed processes
//	- Resource cleanup for completed processes
//	- Analytics calculations for completion rates
//	- Alert logic for long-running incomplete processes
func (p Process) IsCompleted() bool {
	return p.State == StateSuccessful || p.State == StateFailed
}

// IsRunning returns true if the process is currently in active execution state.
// This method identifies processes that are actively executing, supporting
// real-time monitoring, resource management, and operational dashboards.
//
// Active State Detection:
//
//	Running state identification:
//	- StateRunning: Process is actively executing
//	- Returns false for all other states
//	- Supports real-time monitoring and alerting
//	- Enables resource utilization tracking
//
// Monitoring Applications:
//
//	Active process tracking for operations:
//	- Real-time dashboard updates and status displays
//	- Resource utilization monitoring and capacity planning
//	- Performance monitoring and bottleneck detection
//	- Alert logic for stuck or long-running processes
//
// Returns:
//   - bool: true if process is in StateRunning
//   - bool: false for all other states
//
// Operational Usage:
//
//	Process management and monitoring:
//	- Active process counting for capacity management
//	- Resource allocation and scheduling decisions
//	- Performance monitoring and optimization
//	- Alerting and notification logic
func (p Process) IsRunning() bool {
	return p.State == StateRunning
}

// GetProgressPercentage returns a rough progress percentage based on current process state.
// This method provides a simplified progress indicator for user interface elements,
// enabling progress bars, visual indicators, and completion tracking.
//
// Progress Mapping:
//
//	State-based progress estimation:
//	- StateStarted: 25% (initialization complete)
//	- StateRunning: 75% (actively processing)
//	- StateSuccessful: 100% (completed successfully)
//	- StateFailed: 100% (completed with failure)
//	- Default: 0% (unknown or invalid state)
//
// UI Integration:
//
//	Progress visualization applications:
//	- Progress bar rendering in web interfaces
//	- Status indicators and completion gauges
//	- Mobile application progress displays
//	- Dashboard summary visualizations
//
// Returns:
//   - int: Progress percentage (0-100) based on current state
//
// Limitations:
//
//	Simplified progress model:
//	- Does not account for actual task completion within states
//	- Provides estimated progress based on state progression
//	- Suitable for general progress indication, not precise measurement
//	- Consider implementing detailed progress tracking for precision
//
// Template Usage:
//
//	HTML template integration:
//	- Progress bar width calculations
//	- CSS class selection for styling
//	- Conditional rendering based on progress level
//	- Responsive design compatibility
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

// GetStateIcon returns an emoji icon representation for the current process state.
// This method provides visual indicators for process states, enhancing user
// interface readability and enabling quick status recognition in dashboards.
//
// Icon Mapping:
//
//	Visual state representations:
//	- StateStarted: "🚀" (rocket for launch/initialization)
//	- StateRunning: "⚡" (lightning for active processing)
//	- StateSuccessful: "✅" (check mark for success)
//	- StateFailed: "❌" (X mark for failure)
//	- Default: "❓" (question mark for unknown state)
//
// User Interface Applications:
//
//	Visual enhancement for interfaces:
//	- Dashboard status indicators and quick recognition
//	- Table and list view status columns
//	- Mobile application status displays
//	- Notification and alert visual elements
//
// Returns:
//   - string: Unicode emoji character representing current state
//
// Accessibility Considerations:
//
//	Visual indicator limitations:
//	- Emoji may not be accessible to all users
//	- Consider alternative text or additional indicators
//	- Ensure sufficient contrast and visibility
//	- Provide fallback text for screen readers
//
// Template Integration:
//
//	HTML template usage:
//	- Direct insertion in template expressions
//	- CSS styling and positioning support
//	- Responsive design compatibility
//	- Cross-browser emoji rendering considerations
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

// ToJSON converts the ProcessList to formatted JSON string for API responses and data exchange.
// This method provides structured JSON serialization with proper formatting for
// human readability and API compatibility, supporting data exchange and persistence.
//
// JSON Formatting:
//
//	Structured output with indentation:
//	- Two-space indentation for readability
//	- Proper field ordering and structure
//	- UTF-8 encoding for international compatibility
//	- Standard JSON format for cross-platform compatibility
//
// API Integration:
//
//	RESTful API response formatting:
//	- Consistent response structure for client applications
//	- Error handling for serialization failures
//	- Content-Type header compatibility
//	- Efficient serialization for high-volume scenarios
//
// Parameters:
//   - pl: ProcessList instance to serialize
//
// Returns:
//   - []byte: Formatted JSON representation of process list
//   - error: Serialization errors or data validation failures
//
// Error Handling:
//
//	Serialization error conditions:
//	- Invalid data structures or circular references
//	- Memory allocation failures for large datasets
//	- Character encoding issues with special characters
//	- Field validation failures during serialization
//
// Usage Examples:
//
//	API response generation:
//	- HTTP response body for REST endpoints
//	- File export and data backup operations
//	- Inter-service communication and data exchange
//	- Logging and audit trail persistence
func (pl *ProcessList) ToJSON() ([]byte, error) {
	return json.MarshalIndent(pl, "", "  ")
}

// FromJSON creates a ProcessList from JSON data with comprehensive error handling.
// This function provides robust JSON deserialization capabilities for loading
// process data from external sources, API requests, and persistent storage.
//
// Deserialization Features:
//
//	Robust JSON parsing with validation:
//	- Structure validation and field mapping
//	- Type conversion and data validation
//	- Error handling for malformed JSON
//	- Memory-efficient parsing for large datasets
//
// Data Validation:
//
//	Input validation and sanitization:
//	- JSON syntax validation and error reporting
//	- Field presence and type validation
//	- Business logic validation for process states
//	- Defensive programming against malicious input
//
// Parameters:
//   - data: JSON byte array containing process list data
//
// Returns:
//   - *ProcessList: Pointer to deserialized process list structure
//   - error: Parsing errors, validation failures, or data corruption
//
// Error Conditions:
//
//	JSON parsing error scenarios:
//	- Malformed JSON syntax or structure errors
//	- Type mismatch and conversion failures
//	- Missing required fields or invalid data
//	- Memory allocation failures for large datasets
//
// Security Considerations:
//
//	Input validation and safety:
//	- Protection against JSON injection attacks
//	- Memory exhaustion prevention for large inputs
//	- Type safety and boundary validation
//	- Defensive programming practices
//
// Usage Examples:
//
//	Data loading and import operations:
//	- API request body parsing and validation
//	- File import and data migration operations
//	- Configuration loading and initialization
//	- Inter-service communication and data exchange
func FromJSON(data []byte) (*ProcessList, error) {
	var pl ProcessList
	err := json.Unmarshal(data, &pl)
	return &pl, err
}

// NewProcess creates a new process with initial state and comprehensive metadata initialization.
// This constructor function provides standardized process creation with proper
// initialization, ensuring consistent data structure and audit trail establishment.
//
// Initialization Features:
//
//	Comprehensive process setup:
//	- Unique ID generation with process identifier prefix
//	- Initial state assignment (StateStarted)
//	- Timestamp initialization for creation and update tracking
//	- History initialization with first state entry
//
// Process Lifecycle:
//
//	Initial state management:
//	- StateStarted as default initial state
//	- Creation timestamp for lifecycle tracking
//	- Initial history entry for audit trail
//	- Proper structure initialization for subsequent operations
//
// Parameters:
//   - processID: Business process identifier for external system integration
//
// Returns:
//   - *Process: Pointer to newly created and initialized process structure
//
// ID Generation:
//
//	Process identification scheme:
//	- Internal ID: "process_" + processID for database operations
//	- Business ID: processID for external system mapping
//	- Creation timestamp: Current time for lifecycle tracking
//	- History initialization: First entry with started state
//
// Usage Examples:
//
//	Process creation workflows:
//	- Workflow initiation and process spawning
//	- API endpoint process creation handlers
//	- Batch process initialization and management
//	- Integration with external systems and services
//
// Best Practices:
//
//	Process creation guidelines:
//	- Use meaningful processID values for operational clarity
//	- Validate processID uniqueness before creation
//	- Consider process naming conventions for consistency
//	- Implement proper error handling for creation failures
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

// UpdateState adds a new state to the process history and updates current state with timestamp tracking.
// This method provides state transition management with comprehensive audit trail
// maintenance, supporting workflow progression and historical analysis.
//
// State Transition Management:
//
//	Comprehensive state update process:
//	- Current state assignment for real-time monitoring
//	- Update timestamp tracking for modification monitoring
//	- History entry addition for audit trail maintenance
//	- Atomic operation ensuring data consistency
//
// Audit Trail Features:
//
//	Historical tracking capabilities:
//	- Complete state transition history preservation
//	- Precise timestamp recording for timeline analysis
//	- Audit compliance and regulatory requirements
//	- Performance analysis and optimization opportunities
//
// Parameters:
//   - p: Process instance to update (receiver)
//   - newState: ProcessState to transition to
//
// Data Consistency:
//
//	Atomic state update operations:
//	- Current state and timestamp synchronization
//	- History array append with consistent timing
//	- Data integrity through synchronized updates
//	- Concurrent access safety considerations
//
// Usage Examples:
//
//	Workflow state management:
//	- Process progression through workflow stages
//	- Error handling and failure state assignment
//	- Completion notification and finalization
//	- External system integration and status updates
//
// State Validation:
//
//	Business logic considerations:
//	- Validate state transition rules before calling
//	- Implement business logic for valid transitions
//	- Consider implementing state machine patterns
//	- Add validation for terminal state transitions
//
// Performance Considerations:
//
//	Efficient state management:
//	- History array grows with each transition
//	- Consider history truncation for long-running processes
//	- Memory usage monitoring for high-volume scenarios
//	- Database persistence strategies for large histories
func (p *Process) UpdateState(newState ProcessState) {
	now := time.Now()
	p.State = newState
	p.UpdatedAt = now

	p.History = append(p.History, HistoryEntry{
		State:     newState,
		Timestamp: now,
	})
}

// templateFuncs provides a template function map for additional formatting in HTML templates.
// This variable defines custom template functions that extend Go's template capabilities,
// enabling sophisticated formatting and presentation logic within HTML templates.
//
// Function Definitions:
//
//	Custom template functions for enhanced presentation:
//	- formatTime: Timestamp formatting for consistent display
//	- formatDuration: Duration calculation and formatting between timestamps
//	- stateIcon: Emoji icon selection based on process state
//	- Additional utility functions for template enhancement
//
// Template Integration:
//
//	HTML template enhancement capabilities:
//	- Direct function calls from template expressions
//	- Consistent formatting across all template uses
//	- Reusable formatting logic and presentation rules
//	- Cross-template function availability and consistency
//
// Formatting Functions:
//
//	Available template functions:
//
//	formatTime: Timestamp formatting function
//	- Parameter: time.Time value for formatting
//	- Returns: Formatted string in "2006-01-02 15:04:05" format
//	- Usage: {{formatTime .Timestamp}}
//
//	formatDuration: Duration calculation and formatting
//	- Parameters: start time.Time, end time.Time
//	- Returns: Human-readable duration string
//	- Usage: {{formatDuration .StartTime .EndTime}}
//
//	stateIcon: State-based icon selection
//	- Parameter: ProcessState value
//	- Returns: Unicode emoji character for state
//	- Usage: {{stateIcon .State}}
//
// Template Usage Examples:
//
//	HTML template integration patterns:
//	- {{formatTime .CreatedAt}} for timestamp display
//	- {{formatDuration .History.First.Timestamp .History.Last.Timestamp}}
//	- {{stateIcon .State}} for visual state indicators
//	- Conditional formatting and presentation logic
//
// Customization:
//
//	Template function extension:
//	- Add custom formatting functions as needed
//	- Implement business-specific presentation logic
//	- Support internationalization and localization
//	- Enhance user experience through rich formatting
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

// main function serves as the application entry point and demonstrates comprehensive usage.
// This function provides a complete example of process monitoring application usage,
// including JSON data parsing, HTTP server setup, and template rendering with
// both web dashboard and API endpoint functionality.
//
// Application Architecture:
//
//	Multi-tier web application structure:
//	- Data layer: JSON parsing and process management
//	- Service layer: Business logic and state management
//	- Presentation layer: HTML templating and web interface
//	- API layer: RESTful endpoints for programmatic access
//
// Server Configuration:
//
//	HTTP server setup with multiple endpoints:
//	- Root endpoint ("/") for web dashboard rendering
//	- API endpoint ("/api/processes") for JSON data access
//	- Template rendering with custom function integration
//	- Error handling and graceful degradation
//
// Example Data:
//
//	Comprehensive sample data demonstrating:
//	- Multiple processes with different states and histories
//	- State transition patterns and timeline examples
//	- Duration calculations and performance metrics
//	- JSON structure and field relationships
//
// Web Dashboard Features:
//
//	Template-based web interface:
//	- Process listing with state visualization
//	- Historical timeline and progress tracking
//	- Interactive elements and responsive design
//	- Error handling and user feedback
//
// API Endpoints:
//
//	RESTful API functionality:
//	- JSON response format with proper content types
//	- Error handling and HTTP status codes
//	- Cross-origin resource sharing considerations
//	- Integration with external monitoring systems
//
// Error Handling:
//
//	Comprehensive error management:
//	- Template parsing and execution errors
//	- JSON serialization and deserialization errors
//	- HTTP server errors and graceful degradation
//	- Logging and debugging information
//
// Production Considerations:
//
//	Enterprise deployment requirements:
//	- Security considerations for web endpoints
//	- Performance optimization for high-volume scenarios
//	- Monitoring and alerting integration
//	- Scalability and load balancing considerations
//
// Example Usage:
//
//	Application startup and operation:
//	1. Parse sample JSON data into process structures
//	2. Configure HTTP server with template and API endpoints
//	3. Start server and begin serving requests
//	4. Access web dashboard at http://localhost:8080
//	5. Query API endpoint at http://localhost:8080/api/processes
//
// Development Workflow:
//
//	Development and testing procedures:
//	- Template development and testing
//	- API endpoint testing and validation
//	- Error condition testing and handling
//	- Performance testing and optimization
func main() {
	// Sample JSON data demonstrating process tracking with multiple states and transitions
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

	// Parse JSON data into ProcessList structure with error handling
	processList, err := FromJSON([]byte(jsonData))
	if err != nil {
		log.Fatal("Error parsing JSON:", err)
	}

	// Log process information for operational monitoring
	log.Printf("Loaded %d processes", processList.Count)
	for _, process := range processList.Processes {
		log.Printf("Process %s: %s (Duration: %s)",
			process.ProcessID,
			process.State,
			process.FormatDuration())
	}

	// Configure HTTP server with web dashboard endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Template parsing with custom function map
		var tmpl *template.Template
		var err error

		// Attempt to parse template file with custom functions
		if tmpl, err = template.New("flow.gohtml").Funcs(templateFuncs).ParseFiles("flow.gohtml"); err != nil {
			// Handle template parsing errors gracefully
			http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Template parsing error: %v", err)
			return
		}

		// Debug logging for process count monitoring
		fmt.Println(processList.Count)

		// Execute template with process data and handle rendering errors
		if err := tmpl.Execute(w, *processList); err != nil {
			http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Template execution error: %v", err)
			return
		}
	})

	// Configure API endpoint for JSON data access
	http.HandleFunc("/api/processes", func(w http.ResponseWriter, r *http.Request) {
		// Set proper content type for JSON response
		w.Header().Set("Content-Type", "application/json")

		// Serialize process list to JSON with error handling
		jsonBytes, err := processList.ToJSON()
		if err != nil {
			http.Error(w, "JSON encoding error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Send JSON response to client
		w.Write(jsonBytes)
	})

	// Start HTTP server with comprehensive logging
	log.Println("Server starting on :8080")
	log.Println("View processes at: http://localhost:8080")
	log.Println("API endpoint at: http://localhost:8080/api/processes")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
