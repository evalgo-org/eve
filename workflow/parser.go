package workflow

import (
	"encoding/json"
	"fmt"

	"eve.evalgo.org/semantic"
)

// Parser handles JSON-LD parsing for workflow definitions

// ParseWorkflow parses JSON-LD workflow definition into internal representation
func ParseWorkflow(jsonld []byte) (*semantic.WorkflowDefinition, error) {
	// First, detect the type
	var typeDetector struct {
		Type string `json:"@type"`
	}

	if err := json.Unmarshal(jsonld, &typeDetector); err != nil {
		return nil, fmt.Errorf("failed to detect workflow type: %w", err)
	}

	switch typeDetector.Type {
	case "ItemList":
		return parseItemList(jsonld)
	case "HowTo":
		return parseHowTo(jsonld)
	case "ScheduledAction":
		return parseScheduledAction(jsonld)
	case "MapAction":
		return parseMapAction(jsonld)
	default:
		return nil, fmt.Errorf("unsupported workflow type: %s", typeDetector.Type)
	}
}

// parseItemList parses a Schema.org ItemList into WorkflowDefinition
func parseItemList(jsonld []byte) (*semantic.WorkflowDefinition, error) {
	var itemList semantic.SemanticItemList
	if err := json.Unmarshal(jsonld, &itemList); err != nil {
		return nil, fmt.Errorf("failed to parse ItemList: %w", err)
	}

	// Validate
	if itemList.Type != "ItemList" {
		return nil, fmt.Errorf("expected @type 'ItemList', got '%s'", itemList.Type)
	}

	if len(itemList.ItemListElement) == 0 {
		return nil, fmt.Errorf("ItemList has no items")
	}

	// Convert to internal representation
	// Prefer @id over identifier
	workflowID := itemList.ID
	if workflowID == "" {
		workflowID = itemList.Identifier
	}
	if workflowID == "" {
		return nil, fmt.Errorf("workflow must have either @id or identifier")
	}

	workflow := &semantic.WorkflowDefinition{
		ID:          workflowID,
		Name:        itemList.Name,
		Description: itemList.Description,
		Type:        semantic.WorkflowTypeItemList,
		Actions:     make([]semantic.WorkflowAction, 0, len(itemList.ItemListElement)),
	}

	// Process each list item
	for _, listItem := range itemList.ItemListElement {
		if listItem.Type != "ListItem" {
			return nil, fmt.Errorf("expected itemListElement @type 'ListItem', got '%s'", listItem.Type)
		}

		action := semantic.WorkflowAction{
			Type:     "action",
			Action:   listItem.Item,
			Position: listItem.Position,
		}

		// Inherit dependencies from ItemList
		if len(itemList.DependsOn) > 0 {
			action.DependsOn = itemList.DependsOn
		}

		// Add action-specific dependencies
		if listItem.Item != nil && len(listItem.Item.Requires) > 0 {
			action.DependsOn = append(action.DependsOn, listItem.Item.Requires...)
		}

		workflow.Actions = append(workflow.Actions, action)
	}

	return workflow, nil
}

// parseHowTo parses a Schema.org HowTo into WorkflowDefinition
func parseHowTo(jsonld []byte) (*semantic.WorkflowDefinition, error) {
	var howTo semantic.SemanticHowTo
	if err := json.Unmarshal(jsonld, &howTo); err != nil {
		return nil, fmt.Errorf("failed to parse HowTo: %w", err)
	}

	// Validate
	if howTo.Type != "HowTo" {
		return nil, fmt.Errorf("expected @type 'HowTo', got '%s'", howTo.Type)
	}

	if len(howTo.Step) == 0 {
		return nil, fmt.Errorf("HowTo has no steps")
	}

	// Convert to internal representation
	// Prefer @id over identifier
	workflowID := howTo.ID
	if workflowID == "" {
		workflowID = howTo.Identifier
	}
	if workflowID == "" {
		return nil, fmt.Errorf("workflow must have either @id or identifier")
	}

	workflow := &semantic.WorkflowDefinition{
		ID:          workflowID,
		Name:        howTo.Name,
		Description: howTo.Description,
		Type:        semantic.WorkflowTypeHowTo,
		Actions:     make([]semantic.WorkflowAction, 0, len(howTo.Step)),
	}

	// Process each step
	for _, step := range howTo.Step {
		if step.Type != "HowToStep" {
			return nil, fmt.Errorf("expected step @type 'HowToStep', got '%s'", step.Type)
		}

		// Determine what type of action this step contains
		action, err := parseStepElement(step.ItemListElement, step.Position)
		if err != nil {
			return nil, fmt.Errorf("failed to parse step '%s': %w", step.Name, err)
		}

		workflow.Actions = append(workflow.Actions, *action)
	}

	return workflow, nil
}

// parseStepElement determines the type of element in a HowToStep
func parseStepElement(element interface{}, position int) (*semantic.WorkflowAction, error) {
	// Marshal back to JSON to re-parse with correct type
	data, err := json.Marshal(element)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal step element: %w", err)
	}

	var typeDetector struct {
		Type string `json:"@type"`
	}
	if err := json.Unmarshal(data, &typeDetector); err != nil {
		return nil, fmt.Errorf("failed to detect step element type: %w", err)
	}

	switch typeDetector.Type {
	case "ScheduledAction":
		var action semantic.SemanticScheduledAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse ScheduledAction: %w", err)
		}
		return &semantic.WorkflowAction{
			Type:     "action",
			Action:   &action,
			Position: position,
		}, nil

	case "ItemList":
		var itemList semantic.SemanticItemList
		if err := json.Unmarshal(data, &itemList); err != nil {
			return nil, fmt.Errorf("failed to parse ItemList: %w", err)
		}
		return &semantic.WorkflowAction{
			Type:     "loop",
			Loop:     &itemList,
			Position: position,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported step element type: %s", typeDetector.Type)
	}
}

// parseScheduledAction parses a single ScheduledAction into WorkflowDefinition
func parseScheduledAction(jsonld []byte) (*semantic.WorkflowDefinition, error) {
	var action semantic.SemanticScheduledAction
	if err := json.Unmarshal(jsonld, &action); err != nil {
		return nil, fmt.Errorf("failed to parse ScheduledAction: %w", err)
	}

	// Validate
	if action.Type != "ScheduledAction" {
		return nil, fmt.Errorf("expected @type 'ScheduledAction', got '%s'", action.Type)
	}

	// Convert to internal representation
	workflow := &semantic.WorkflowDefinition{
		ID:          action.Identifier,
		Name:        action.Name,
		Description: action.Description,
		Type:        semantic.WorkflowTypeScheduledAction,
		Actions: []semantic.WorkflowAction{
			{
				Type:      "action",
				Action:    &action,
				DependsOn: action.Requires,
				Position:  1,
			},
		},
	}

	return workflow, nil
}

// parseMapAction parses a single MapAction into WorkflowDefinition
func parseMapAction(jsonld []byte) (*semantic.WorkflowDefinition, error) {
	var action semantic.SemanticScheduledAction
	if err := json.Unmarshal(jsonld, &action); err != nil {
		return nil, fmt.Errorf("failed to parse MapAction: %w", err)
	}

	// Validate
	if action.Type != "MapAction" {
		return nil, fmt.Errorf("expected @type 'MapAction', got '%s'", action.Type)
	}

	// Convert to internal representation
	workflow := &semantic.WorkflowDefinition{
		ID:          action.Identifier,
		Name:        action.Name,
		Description: action.Description,
		Type:        semantic.WorkflowTypeMapAction,
		Actions: []semantic.WorkflowAction{
			{
				Type:      "action",
				Action:    &action,
				DependsOn: action.Requires,
				Position:  1,
			},
		},
	}

	return workflow, nil
}
