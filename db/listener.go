// Package db provides PostgreSQL LISTEN/NOTIFY support for real-time event streaming.
package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StateEvent represents a state change notification from PostgreSQL.
type StateEvent struct {
	Type        string                 `json:"type"`
	WorkflowID  string                 `json:"workflow_id,omitempty"`
	ActionID    string                 `json:"action_id,omitempty"`
	Phase       string                 `json:"phase,omitempty"`
	Status      string                 `json:"status,omitempty"`
	ProgressPct int                    `json:"progress_pct,omitempty"`
	Stage       string                 `json:"stage,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Timestamp   string                 `json:"timestamp,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// StateEventHandler is called when a state event is received.
type StateEventHandler func(event *StateEvent)

// Listener subscribes to PostgreSQL NOTIFY channels and dispatches events.
type Listener struct {
	pool        *pgxpool.Pool
	channel     string
	handlers    []StateEventHandler
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	running     bool
	reconnectCh chan struct{}
}

// NewListener creates a new PostgreSQL LISTEN subscriber.
func NewListener(pool *pgxpool.Pool, channel string) *Listener {
	ctx, cancel := context.WithCancel(context.Background())
	return &Listener{
		pool:        pool,
		channel:     channel,
		handlers:    make([]StateEventHandler, 0),
		ctx:         ctx,
		cancel:      cancel,
		reconnectCh: make(chan struct{}, 1),
	}
}

// OnEvent registers a handler for state events.
func (l *Listener) OnEvent(handler StateEventHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers = append(l.handlers, handler)
}

// Start begins listening for notifications.
func (l *Listener) Start() error {
	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return nil
	}
	l.running = true
	l.mu.Unlock()

	go l.listenLoop()
	return nil
}

// Stop stops listening for notifications.
func (l *Listener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.running {
		return
	}

	l.running = false
	l.cancel()
}

// listenLoop maintains the LISTEN connection with reconnection support.
func (l *Listener) listenLoop() {
	for {
		select {
		case <-l.ctx.Done():
			return
		default:
			if err := l.listen(); err != nil {
				log.Printf("[Listener] Listen error: %v, reconnecting in 1s", err)
				select {
				case <-l.ctx.Done():
					return
				case <-time.After(time.Second):
					continue
				}
			}
		}
	}
}

// listen establishes a LISTEN connection and processes notifications.
func (l *Listener) listen() error {
	conn, err := l.pool.Acquire(l.ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Start listening
	_, err = conn.Exec(l.ctx, fmt.Sprintf("LISTEN %s", l.channel))
	if err != nil {
		return fmt.Errorf("failed to start LISTEN: %w", err)
	}

	log.Printf("[Listener] Listening on channel: %s", l.channel)

	for {
		notification, err := conn.Conn().WaitForNotification(l.ctx)
		if err != nil {
			return fmt.Errorf("notification wait error: %w", err)
		}

		// Parse notification payload
		var event StateEvent
		if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
			log.Printf("[Listener] Failed to parse notification: %v", err)
			continue
		}

		// Dispatch to handlers
		l.dispatch(&event)
	}
}

// dispatch sends event to all registered handlers.
func (l *Listener) dispatch(event *StateEvent) {
	l.mu.RLock()
	handlers := make([]StateEventHandler, len(l.handlers))
	copy(handlers, l.handlers)
	l.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}
