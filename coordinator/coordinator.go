package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Config holds configuration for the Coordinator.
type Config struct {
	// WhenURL is the WebSocket URL to connect to (e.g., "ws://localhost:8080/v1/coordination")
	WhenURL string

	// ServiceName is the name of this service (e.g., "containerservice")
	ServiceName string

	// ServiceID is a unique identifier for this service instance
	ServiceID string

	// Capabilities lists what this service can do
	Capabilities []string

	// Version is the service version
	Version string

	// Reconnect settings
	ReconnectInitialDelay  time.Duration
	ReconnectMaxDelay      time.Duration
	ReconnectBackoffFactor float64
	ReconnectMaxAttempts   int // 0 = infinite

	// PingInterval is how often to send pings
	PingInterval time.Duration

	// Logger for coordinator messages
	Logger *logrus.Entry
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ReconnectInitialDelay:  1 * time.Second,
		ReconnectMaxDelay:      30 * time.Second,
		ReconnectBackoffFactor: 2.0,
		ReconnectMaxAttempts:   0, // infinite
		PingInterval:           30 * time.Second,
	}
}

// Coordinator manages WebSocket communication with when-v3.
type Coordinator struct {
	config Config
	logger *logrus.Entry

	conn      *websocket.Conn
	connMu    sync.RWMutex
	connected bool

	// Phase management
	phases *PhaseManager

	// Message handling
	handlers   map[MessageType]MessageHandler
	handlersMu sync.RWMutex

	// Outgoing messages
	sendChan chan *WSMessage

	// Lifecycle
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	registered bool
	serviceID  string // Assigned by when-v3

	// Callbacks
	onConnected    func()
	onDisconnected func(error)
	onRegistered   func(serviceID string)
}

// MessageHandler is a function that handles incoming messages.
type MessageHandler func(msg *WSMessage) error

// New creates a new Coordinator.
func New(config Config) *Coordinator {
	if config.Logger == nil {
		config.Logger = logrus.NewEntry(logrus.StandardLogger())
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Coordinator{
		config:   config,
		logger:   config.Logger.WithField("component", "coordinator"),
		phases:   NewPhaseManager(),
		handlers: make(map[MessageType]MessageHandler),
		sendChan: make(chan *WSMessage, 100),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Register default handlers
	c.registerDefaultHandlers()

	// Set up phase change notifications
	c.phases.OnPhaseChanged(func(state *PhaseState) {
		c.sendPhaseChanged(state)
	})

	return c
}

// registerDefaultHandlers sets up handlers for standard message types.
func (c *Coordinator) registerDefaultHandlers() {
	c.handlers[MessageTypePing] = c.handlePing
	c.handlers[MessageTypeRegistered] = c.handleRegistered
	c.handlers[MessageTypePause] = c.handlePause
	c.handlers[MessageTypeResume] = c.handleResume
	c.handlers[MessageTypeCancel] = c.handleCancel
	c.handlers[MessageTypeStatus] = c.handleStatus
}

// OnMessage registers a custom handler for a message type.
func (c *Coordinator) OnMessage(msgType MessageType, handler MessageHandler) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()
	c.handlers[msgType] = handler
}

// OnConnected sets a callback for when connection is established.
func (c *Coordinator) OnConnected(fn func()) {
	c.onConnected = fn
}

// OnDisconnected sets a callback for when connection is lost.
func (c *Coordinator) OnDisconnected(fn func(error)) {
	c.onDisconnected = fn
}

// OnRegistered sets a callback for when registration completes.
func (c *Coordinator) OnRegistered(fn func(serviceID string)) {
	c.onRegistered = fn
}

// Phases returns the phase manager for direct access.
func (c *Coordinator) Phases() *PhaseManager {
	return c.phases
}

// Connect establishes the WebSocket connection and starts processing.
func (c *Coordinator) Connect() error {
	c.wg.Add(1)
	go c.connectionLoop()
	return nil
}

// Close shuts down the coordinator.
func (c *Coordinator) Close() error {
	c.cancel()
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.connMu.Unlock()
	c.wg.Wait()
	return nil
}

// IsConnected returns whether the WebSocket is connected.
func (c *Coordinator) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.connected
}

// connectionLoop manages connection and reconnection.
func (c *Coordinator) connectionLoop() {
	defer c.wg.Done()

	delay := c.config.ReconnectInitialDelay
	attempts := 0

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		err := c.connect()
		if err != nil {
			attempts++
			c.logger.WithError(err).WithField("attempt", attempts).Warn("Connection failed")

			if c.config.ReconnectMaxAttempts > 0 && attempts >= c.config.ReconnectMaxAttempts {
				c.logger.Error("Max reconnection attempts reached")
				return
			}

			// Wait with backoff
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(delay):
			}

			// Increase delay for next attempt
			delay = time.Duration(float64(delay) * c.config.ReconnectBackoffFactor)
			if delay > c.config.ReconnectMaxDelay {
				delay = c.config.ReconnectMaxDelay
			}
			continue
		}

		// Reset on successful connection
		delay = c.config.ReconnectInitialDelay
		attempts = 0

		// Run connection until it fails
		err = c.runConnection()
		if err != nil {
			c.logger.WithError(err).Warn("Connection lost")
			if c.onDisconnected != nil {
				c.onDisconnected(err)
			}
		}

		c.connMu.Lock()
		c.connected = false
		c.registered = false
		c.connMu.Unlock()
	}
}

// connect establishes the WebSocket connection.
func (c *Coordinator) connect() error {
	c.logger.WithField("url", c.config.WhenURL).Info("Connecting to when-v3")

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	headers := http.Header{}
	headers.Set("X-Service-Name", c.config.ServiceName)
	if c.config.ServiceID != "" {
		headers.Set("X-Service-ID", c.config.ServiceID)
	}

	conn, _, err := dialer.DialContext(c.ctx, c.config.WhenURL, headers)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.connMu.Lock()
	c.conn = conn
	c.connected = true
	c.connMu.Unlock()

	c.logger.Info("Connected to when-v3")
	if c.onConnected != nil {
		c.onConnected()
	}

	// Send registration
	if err := c.sendRegistration(); err != nil {
		conn.Close()
		return fmt.Errorf("registration failed: %w", err)
	}

	return nil
}

// sendRegistration sends the register message.
func (c *Coordinator) sendRegistration() error {
	msg := NewMessage(MessageTypeRegister)
	msg.SetPayload(RegisterPayload{
		ServiceName:  c.config.ServiceName,
		ServiceID:    c.config.ServiceID,
		Capabilities: c.config.Capabilities,
		Version:      c.config.Version,
	})

	return c.sendMessage(msg)
}

// runConnection handles the connection lifecycle.
func (c *Coordinator) runConnection() error {
	// Start sender goroutine
	senderDone := make(chan struct{})
	go func() {
		defer close(senderDone)
		c.senderLoop()
	}()

	// Start ping goroutine
	pingDone := make(chan struct{})
	go func() {
		defer close(pingDone)
		c.pingLoop()
	}()

	// Read messages
	err := c.readLoop()

	// Close connection to stop sender
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.connMu.Unlock()

	// Wait for sender to finish
	<-senderDone
	<-pingDone

	return err
}

// readLoop reads and dispatches incoming messages.
func (c *Coordinator) readLoop() error {
	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()

		if conn == nil {
			return fmt.Errorf("connection closed")
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		msg, err := ParseMessage(data)
		if err != nil {
			c.logger.WithError(err).Warn("Failed to parse message")
			continue
		}

		c.handleMessage(msg)
	}
}

// senderLoop sends outgoing messages.
func (c *Coordinator) senderLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-c.sendChan:
			if !ok {
				return
			}
			if err := c.sendMessage(msg); err != nil {
				c.logger.WithError(err).Warn("Failed to send message")
			}
		}
	}
}

// pingLoop sends periodic pings.
func (c *Coordinator) pingLoop() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				return
			}

			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				c.logger.WithError(err).Debug("Ping failed")
			}
		}
	}
}

// sendMessage sends a message immediately.
func (c *Coordinator) sendMessage(msg *WSMessage) error {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	data, err := msg.JSON()
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// Send queues a message for sending.
func (c *Coordinator) Send(msg *WSMessage) {
	select {
	case c.sendChan <- msg:
	default:
		c.logger.Warn("Send channel full, dropping message")
	}
}

// handleMessage dispatches a message to its handler.
func (c *Coordinator) handleMessage(msg *WSMessage) {
	c.handlersMu.RLock()
	handler, ok := c.handlers[msg.Type]
	c.handlersMu.RUnlock()

	if !ok {
		c.logger.WithField("type", msg.Type).Debug("No handler for message type")
		return
	}

	if err := handler(msg); err != nil {
		c.logger.WithError(err).WithField("type", msg.Type).Warn("Handler error")
	}
}

// Default handlers

func (c *Coordinator) handlePing(msg *WSMessage) error {
	pong := NewMessage(MessageTypePong)
	pong.ID = msg.ID // Use same ID for correlation
	return c.sendMessage(pong)
}

func (c *Coordinator) handleRegistered(msg *WSMessage) error {
	payload, err := msg.GetRegisterPayload()
	if err != nil {
		// Try direct extraction
		if sid, ok := msg.Payload["service_id"].(string); ok {
			c.serviceID = sid
		}
	} else {
		c.serviceID = payload.ServiceID
	}

	c.connMu.Lock()
	c.registered = true
	c.connMu.Unlock()

	c.logger.WithField("service_id", c.serviceID).Info("Registered with when-v3")

	if c.onRegistered != nil {
		c.onRegistered(c.serviceID)
	}

	// Report state of all active workflows
	for _, state := range c.phases.GetActiveWorkflows() {
		c.sendPhaseChanged(state)
	}

	return nil
}

func (c *Coordinator) handlePause(msg *WSMessage) error {
	payload, err := msg.GetPausePayload()
	if err != nil {
		return fmt.Errorf("invalid pause payload: %w", err)
	}

	c.logger.WithField("workflow_id", payload.WorkflowID).Info("Received pause command")

	return c.phases.Pause(payload.WorkflowID, payload.Reason)
}

func (c *Coordinator) handleResume(msg *WSMessage) error {
	payload, err := msg.GetResumePayload()
	if err != nil {
		return fmt.Errorf("invalid resume payload: %w", err)
	}

	c.logger.WithField("workflow_id", payload.WorkflowID).Info("Received resume command")

	return c.phases.Resume(payload.WorkflowID, payload.FromCheckpoint)
}

func (c *Coordinator) handleCancel(msg *WSMessage) error {
	payload, err := msg.GetCancelPayload()
	if err != nil {
		return fmt.Errorf("invalid cancel payload: %w", err)
	}

	c.logger.WithField("workflow_id", payload.WorkflowID).Info("Received cancel command")

	return c.phases.Cancel(payload.WorkflowID, payload.Reason)
}

func (c *Coordinator) handleStatus(msg *WSMessage) error {
	workflowID := msg.WorkflowID
	if workflowID == "" {
		if wid, ok := msg.Payload["workflow_id"].(string); ok {
			workflowID = wid
		}
	}

	state, ok := c.phases.GetState(workflowID)
	if !ok {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	response := NewMessageWithWorkflow(MessageTypeStatusResponse, workflowID)
	response.ID = msg.ID // Correlation
	response.SetPayload(StatusResponsePayload{
		WorkflowID:    workflowID,
		Phase:         state.Phase,
		Progress:      state.Progress,
		CurrentAction: state.CurrentAction,
	})

	return c.sendMessage(response)
}

// Outgoing message helpers

// sendPhaseChanged sends a phase_changed message to when-v3.
func (c *Coordinator) sendPhaseChanged(state *PhaseState) {
	if !c.IsConnected() {
		return
	}

	msg := NewMessageWithWorkflow(MessageTypePhaseChanged, state.WorkflowID)
	msg.SetPayload(PhaseChangedPayload{
		WorkflowID:   state.WorkflowID,
		FromPhase:    state.PreviousPhase,
		ToPhase:      state.Phase,
		CheckpointID: state.CheckpointID,
		Reason:       state.Reason,
	})

	c.Send(msg)
}

// SendWorkflowCreated notifies when-v3 of a new child workflow.
func (c *Coordinator) SendWorkflowCreated(workflowID, parentWorkflowID, rootWorkflowID, actionID, actionType string) {
	msg := NewMessageWithWorkflow(MessageTypeWorkflowCreated, workflowID)
	msg.SetPayload(WorkflowCreatedPayload{
		WorkflowID:       workflowID,
		ParentWorkflowID: parentWorkflowID,
		RootWorkflowID:   rootWorkflowID,
		ActionID:         actionID,
		ActionType:       actionType,
	})
	c.Send(msg)
}

// SendCheckpoint notifies when-v3 of a checkpoint.
func (c *Coordinator) SendCheckpoint(workflowID, checkpointID, reason string) {
	msg := NewMessageWithWorkflow(MessageTypeCheckpoint, workflowID)
	msg.SetPayload(CheckpointPayload{
		WorkflowID:   workflowID,
		CheckpointID: checkpointID,
		Reason:       reason,
	})
	c.Send(msg)
}

// SendError notifies when-v3 of an error.
func (c *Coordinator) SendError(workflowID, actionID, errorMsg string, recoverable bool) {
	msg := NewMessageWithWorkflow(MessageTypeError, workflowID)
	msg.SetPayload(ErrorPayload{
		WorkflowID:  workflowID,
		ActionID:    actionID,
		Error:       errorMsg,
		Recoverable: recoverable,
	})
	c.Send(msg)
}

// SendProgress notifies when-v3 of progress.
func (c *Coordinator) SendProgress(workflowID, actionID string, percent float64, stage, message string) {
	msg := NewMessageWithWorkflow(MessageTypeProgress, workflowID)
	msg.SetPayload(ProgressPayload{
		WorkflowID: workflowID,
		ActionID:   actionID,
		Percent:    percent,
		Stage:      stage,
		Message:    message,
	})
	c.Send(msg)
}

// Helper function to generate message IDs
func generateMessageID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("msg-%s-%d", string(b), time.Now().UnixNano()%1000000)
}

// PayloadToStruct converts a message payload to a typed struct.
func PayloadToStruct(payload map[string]interface{}, target interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
