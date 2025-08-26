package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	eve "eve.evalgo.org/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ProcessState represents the possible states of a process
type ProcessState string

const (
	StateStarted    ProcessState = "started"
	StateRunning    ProcessState = "running"
	StateSuccessful ProcessState = "successful"
	StateFailed     ProcessState = "failed"
)

// ProcessMessage represents the incoming RabbitMQ message
type ProcessMessage struct {
	ProcessID   string                 `json:"process_id"`
	State       ProcessState           `json:"state"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ErrorMsg    string                 `json:"error_message,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// ProcessDocument represents the CouchDB document structure
type ProcessDocument struct {
	ID          string                 `json:"_id"`
	Rev         string                 `json:"_rev,omitempty"`
	ProcessID   string                 `json:"process_id"`
	State       ProcessState           `json:"state"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	History     []StateChange          `json:"history"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ErrorMsg    string                 `json:"error_message,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// StateChange represents a state transition in the history
type StateChange struct {
	State     ProcessState `json:"state"`
	Timestamp time.Time    `json:"timestamp"`
	ErrorMsg  string       `json:"error_message,omitempty"`
}

// CouchDBResponse represents the response from CouchDB operations
type CouchDBResponse struct {
	OK  bool   `json:"ok"`
	ID  string `json:"id"`
	Rev string `json:"rev"`
}

// CouchDBError represents an error response from CouchDB
type CouchDBError struct {
	Error  string `json:"error"`
	Reason string `json:"reason"`
}

// Config holds the application configuration
type Config struct {
	RabbitMQURL string
	QueueName   string
	CouchDBURL  string
	CouchDBName string
	ApiKey      string
}

// Consumer handles the RabbitMQ consumer logic
type Consumer struct {
	config     Config
	connection *amqp.Connection
	channel    *amqp.Channel
	httpClient *http.Client
}

func init() {
	RootCmd.AddCommand(consumeCmd)
	consumeCmd.PersistentFlags().String("rabbitmq-url", "", "RabbitMQ connection URL")
	consumeCmd.PersistentFlags().String("queue-name", "", "RabbitMQ queue name")
	consumeCmd.PersistentFlags().String("couchdb-url", "", "CouchDB connection URL")
	consumeCmd.PersistentFlags().String("database-name", "", "CouchDB database name")

	viper.BindPFlag("rabbitmq.url", consumeCmd.PersistentFlags().Lookup("rabbitmq-url"))
	viper.BindPFlag("rabbitmq.queue_name", consumeCmd.PersistentFlags().Lookup("queue-name"))
	viper.BindPFlag("couchdb.url", consumeCmd.PersistentFlags().Lookup("couchdb-url"))
	viper.BindPFlag("couchdb.database_name", consumeCmd.PersistentFlags().Lookup("database-name"))
}

var consumeCmd = &cobra.Command{
	Use:   "consume",
	Short: "consume an rabbit message queue",
	Long:  `consume an rabbit message queue`,
	Run: func(cmd *cobra.Command, args []string) {
		rabbitmq_url, _ := cmd.Flags().GetString("rabbitmq-url")
		if rabbitmq_url == "" {
			rabbitmq_url = viper.GetString("rabbitmq.url")
		}
		queue_name, _ := cmd.Flags().GetString("queue-name")
		if queue_name == "" {
			queue_name = viper.GetString("rabbitmq.queue_name")
		}
		couchdb_url, _ := cmd.Flags().GetString("couchdb-url")
		if couchdb_url == "" {
			couchdb_url = viper.GetString("couchdb.url")
		}
		database_name, _ := cmd.Flags().GetString("database-name")
		if database_name == "" {
			database_name = viper.GetString("couchdb.database_name")
		}
		eve.Logger.Info(rabbitmq_url, queue_name, couchdb_url, database_name)
		ConsumerStart(Config{RabbitMQURL: rabbitmq_url, QueueName: queue_name, CouchDBURL: couchdb_url, CouchDBName: database_name})
	},
}

func ConsumerStart(config Config) {
	consumer := NewConsumer(config)
	defer consumer.Close()

	// Create CouchDB database if it doesn't exist
	if err := consumer.createCouchDBDatabase(); err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Connect to RabbitMQ
	if err := consumer.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// Start consuming
	if err := consumer.StartConsuming(); err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Consumer is running. Press CTRL+C to exit...")
	<-sigChan

	log.Println("Shutting down consumer...")
}

type Props struct {
	InFile string
	PID    string
}

// NewConsumer creates a new consumer instance
func NewConsumer(config Config) *Consumer {
	return &Consumer{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Connect establishes connection to RabbitMQ
func (c *Consumer) Connect() error {
	var err error
	c.connection, err = amqp.Dial(c.config.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	c.channel, err = c.connection.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare queue
	_, err = c.channel.QueueDeclare(
		c.config.QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Set QoS to process one message at a time
	err = c.channel.Qos(1, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	return nil
}

// Close closes the connection
func (c *Consumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.connection != nil {
		c.connection.Close()
	}
}

// StartConsuming starts consuming messages
func (c *Consumer) StartConsuming() error {
	msgs, err := c.channel.Consume(
		c.config.QueueName,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("Consumer started. Waiting for messages...")

	go func() {
		for msg := range msgs {
			if err := c.processMessage(msg); err != nil {
				log.Printf("Error processing message: %v", err)
				// Reject message and requeue
				msg.Nack(false, true)
			} else {
				// Acknowledge message
				msg.Ack(false)
			}
		}
	}()

	return nil
}

// processMessage processes a single RabbitMQ message
func (c *Consumer) processMessage(msg amqp.Delivery) error {
	log.Printf("Received message: %s", string(msg.Body))

	var processMsg ProcessMessage
	if err := json.Unmarshal(msg.Body, &processMsg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Validate message
	if processMsg.ProcessID == "" {
		return fmt.Errorf("process_id is required")
	}

	if processMsg.State == "" {
		return fmt.Errorf("state is required")
	}

	if processMsg.Timestamp.IsZero() {
		processMsg.Timestamp = time.Now()
	}

	// Process the message based on state
	switch processMsg.State {
	case StateStarted:
		return c.createProcessDocument(processMsg)
	case StateRunning, StateSuccessful, StateFailed:
		return c.updateProcessDocument(processMsg)
	default:
		return fmt.Errorf("invalid state: %s", processMsg.State)
	}
}

// createProcessDocument creates a new process document in CouchDB
func (c *Consumer) createProcessDocument(msg ProcessMessage) error {
	docID := fmt.Sprintf("process_%s", msg.ProcessID)

	doc := ProcessDocument{
		ID:          docID,
		ProcessID:   msg.ProcessID,
		State:       msg.State,
		CreatedAt:   msg.Timestamp,
		UpdatedAt:   msg.Timestamp,
		History:     []StateChange{{State: msg.State, Timestamp: msg.Timestamp, ErrorMsg: msg.ErrorMsg}},
		Metadata:    msg.Metadata,
		ErrorMsg:    msg.ErrorMsg,
		Description: msg.Description,
	}

	return c.saveToCouchDB(doc, false)
}

// updateProcessDocument updates an existing process document in CouchDB
func (c *Consumer) updateProcessDocument(msg ProcessMessage) error {
	docID := fmt.Sprintf("process_%s", msg.ProcessID)

	// First, get the existing document
	existingDoc, err := c.getFromCouchDB(docID)
	if err != nil {
		return fmt.Errorf("failed to get existing document: %w", err)
	}

	// Update the document
	existingDoc.State = msg.State
	existingDoc.UpdatedAt = msg.Timestamp
	existingDoc.ErrorMsg = msg.ErrorMsg

	if msg.Description != "" {
		existingDoc.Description = msg.Description
	}

	// Merge metadata
	if msg.Metadata != nil {
		if existingDoc.Metadata == nil {
			existingDoc.Metadata = make(map[string]interface{})
		}
		for k, v := range msg.Metadata {
			existingDoc.Metadata[k] = v
		}
	}

	// Add to history
	stateChange := StateChange{
		State:     msg.State,
		Timestamp: msg.Timestamp,
		ErrorMsg:  msg.ErrorMsg,
	}
	existingDoc.History = append(existingDoc.History, stateChange)

	return c.saveToCouchDB(*existingDoc, true)
}

// getFromCouchDB retrieves a document from CouchDB
func (c *Consumer) getFromCouchDB(docID string) (*ProcessDocument, error) {
	url := fmt.Sprintf("%s/%s/%s", c.config.CouchDBURL, c.config.CouchDBName, docID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("document not found: %s", docID)
	}

	if resp.StatusCode != http.StatusOK {
		var couchErr CouchDBError
		if err := json.NewDecoder(resp.Body).Decode(&couchErr); err == nil {
			return nil, fmt.Errorf("CouchDB error: %s - %s", couchErr.Error, couchErr.Reason)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var doc ProcessDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to decode document: %w", err)
	}

	return &doc, nil
}

// saveToCouchDB saves a document to CouchDB
func (c *Consumer) saveToCouchDB(doc ProcessDocument, isUpdate bool) error {
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	var url string
	var method string

	if isUpdate {
		url = fmt.Sprintf("%s/%s/%s", c.config.CouchDBURL, c.config.CouchDBName, doc.ID)
		method = "PUT"
	} else {
		url = fmt.Sprintf("%s/%s", c.config.CouchDBURL, c.config.CouchDBName)
		method = "POST"
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var couchErr CouchDBError
		if err := json.NewDecoder(resp.Body).Decode(&couchErr); err == nil {
			return fmt.Errorf("CouchDB error: %s - %s", couchErr.Error, couchErr.Reason)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response CouchDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.OK {
		return fmt.Errorf("CouchDB operation failed")
	}

	log.Printf("Document %s saved successfully with rev: %s", response.ID, response.Rev)
	return nil
}

// createCouchDBDatabase creates the database if it doesn't exist
func (c *Consumer) createCouchDBDatabase() error {
	url := fmt.Sprintf("%s/%s", c.config.CouchDBURL, c.config.CouchDBName)

	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		log.Printf("Database %s created successfully", c.config.CouchDBName)
	} else if resp.StatusCode == http.StatusPreconditionFailed {
		log.Printf("Database %s already exists", c.config.CouchDBName)
	} else {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// getEnvOrDefault returns environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func listFiles(dir string) []string {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			files = append(files, path)
		}
		// if !d.IsDir() && filepath.Ext(path) == ".md" {
		// 	files = append(files, path)
		// }
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return files
}

func RunCommand(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
