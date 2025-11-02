package semantic

import "time"

// Workflow-specific Schema.org semantic types
// Used by 'when' and other orchestration tools

// SemanticItemList represents a Schema.org ItemList
// Used for loops/iterations in workflows
type SemanticItemList struct {
	Context         string             `json:"@context"`
	Type            string             `json:"@type"` // Must be "ItemList"
	Identifier      string             `json:"identifier,omitempty"`
	Name            string             `json:"name"`
	Description     string             `json:"description,omitempty"`
	NumberOfItems   int                `json:"numberOfItems,omitempty"`
	ItemListElement []SemanticListItem `json:"itemListElement"`

	// Workflow execution config (extensions to Schema.org)
	DependsOn     []string `json:"dependsOn,omitempty"`
	Parallel      bool     `json:"parallel,omitempty"`
	Concurrency   int      `json:"concurrency,omitempty"`   // Max parallel tasks
	MaxIterations int      `json:"maxIterations,omitempty"` // Safety limit
}

// SemanticListItem represents a Schema.org ListItem
type SemanticListItem struct {
	Type     string                   `json:"@type"` // Must be "ListItem"
	Position int                      `json:"position"`
	Item     *SemanticScheduledAction `json:"item"` // The actual action
}

// SemanticHowTo represents a multi-step workflow
type SemanticHowTo struct {
	Context     string              `json:"@context"`
	Type        string              `json:"@type"` // Must be "HowTo"
	Identifier  string              `json:"identifier"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Step        []SemanticHowToStep `json:"step"`
	Created     time.Time           `json:"dateCreated,omitempty"`
	Modified    time.Time           `json:"dateModified,omitempty"`
}

// SemanticHowToStep represents a step in a workflow
type SemanticHowToStep struct {
	Type            string      `json:"@type"` // Must be "HowToStep"
	Name            string      `json:"name"`
	Position        int         `json:"position,omitempty"`
	Text            string      `json:"text,omitempty"`  // Human-readable description
	ItemListElement interface{} `json:"itemListElement"` // Can be ScheduledAction, ItemList, or HowTo
}

// SemanticDigitalDocument represents the target of an HTTP action
type SemanticDigitalDocument struct {
	Type       string            `json:"@type"` // Must be "DigitalDocument"
	URL        string            `json:"url"`
	HTTPMethod string            `json:"httpMethod,omitempty"` // GET, POST, etc.
	Headers    map[string]string `json:"headers,omitempty"`
}

// WorkflowDefinition is the internal representation after parsing JSON-LD
// This is NOT a Schema.org type - it's for internal use
type WorkflowDefinition struct {
	ID          string
	Name        string
	Description string
	Type        WorkflowType // ItemList, HowTo, ScheduledAction
	Actions     []WorkflowAction
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkflowType indicates the semantic type of the workflow
type WorkflowType string

const (
	WorkflowTypeItemList        WorkflowType = "ItemList"
	WorkflowTypeHowTo           WorkflowType = "HowTo"
	WorkflowTypeScheduledAction WorkflowType = "ScheduledAction"
)

// WorkflowAction is an internal representation of an action in a workflow
// This is NOT a Schema.org type - it's for internal use
type WorkflowAction struct {
	Type      string                   // "action", "loop", "step"
	Action    *SemanticScheduledAction // For single actions
	Loop      *SemanticItemList        // For loops
	DependsOn []string                 // Task dependencies
	Variables map[string]interface{}   // For variable substitution
	Position  int
}

// LoopExecutionState tracks the state of a loop during execution
// This is NOT a Schema.org type - it's for internal use
type LoopExecutionState struct {
	LoopID          string
	TotalItems      int
	CurrentPosition int
	Status          string // "pending", "running", "completed", "failed"
	StartedAt       time.Time
	FinishedAt      *time.Time
	Errors          []string
}
