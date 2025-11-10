package semantic

import (
	"fmt"
	"sync"

	"github.com/labstack/echo/v4"
)

// ActionHandler is a function that handles a specific semantic action type
// The action parameter can be either *SemanticAction or *SemanticScheduledAction
type ActionHandler func(c echo.Context, action interface{}) error

// ActionRegistry manages action handlers
type ActionRegistry struct {
	handlers map[string]ActionHandler
	mu       sync.RWMutex
}

// NewActionRegistry creates a new action registry
func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		handlers: make(map[string]ActionHandler),
	}
}

// Register registers a handler for a specific action type
// This allows services to register their action handlers at initialization
func (r *ActionRegistry) Register(actionType string, handler ActionHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[actionType]; exists {
		return fmt.Errorf("handler for action type %s already registered", actionType)
	}

	r.handlers[actionType] = handler
	return nil
}

// MustRegister registers a handler and panics if it fails
// Useful for initialization code
func (r *ActionRegistry) MustRegister(actionType string, handler ActionHandler) {
	if err := r.Register(actionType, handler); err != nil {
		panic(err)
	}
}

// Unregister removes a handler for a specific action type
func (r *ActionRegistry) Unregister(actionType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, actionType)
}

// Handle dispatches an action to the appropriate handler
// Accepts either *SemanticAction or *SemanticScheduledAction
func (r *ActionRegistry) Handle(c echo.Context, actionInterface interface{}) error {
	// Extract the Type field and underlying SemanticAction
	var actionType string
	var baseAction *SemanticAction

	if sa, ok := actionInterface.(*SemanticScheduledAction); ok {
		actionType = sa.Type
		baseAction = &sa.SemanticAction
	} else if a, ok := actionInterface.(*SemanticAction); ok {
		actionType = a.Type
		baseAction = a
	} else {
		return fmt.Errorf("invalid action type: %T", actionInterface)
	}

	r.mu.RLock()
	handler, exists := r.handlers[actionType]
	r.mu.RUnlock()

	if !exists {
		return ReturnActionError(c, baseAction, fmt.Sprintf("Unsupported action type: %s", actionType), nil)
	}

	return handler(c, actionInterface)
}

// GetRegisteredActions returns a list of all registered action types
func (r *ActionRegistry) GetRegisteredActions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actions := make([]string, 0, len(r.handlers))
	for actionType := range r.handlers {
		actions = append(actions, actionType)
	}
	return actions
}

// HasHandler checks if a handler is registered for a specific action type
func (r *ActionRegistry) HasHandler(actionType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.handlers[actionType]
	return exists
}

// DefaultRegistry is a global registry for services to use
var DefaultRegistry = NewActionRegistry()

// Register is a convenience function that registers with the default registry
func Register(actionType string, handler ActionHandler) error {
	return DefaultRegistry.Register(actionType, handler)
}

// MustRegister is a convenience function that registers with the default registry
func MustRegister(actionType string, handler ActionHandler) {
	DefaultRegistry.MustRegister(actionType, handler)
}

// Handle is a convenience function that handles actions using the default registry
// Accepts either *SemanticAction or *SemanticScheduledAction
func Handle(c echo.Context, action interface{}) error {
	return DefaultRegistry.Handle(c, action)
}

// GetRegisteredActions returns registered actions from the default registry
func GetRegisteredActions() []string {
	return DefaultRegistry.GetRegisteredActions()
}
