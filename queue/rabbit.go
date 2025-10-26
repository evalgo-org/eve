// Package queue provides utilities for working with message queues using RabbitMQ.
// It implements a service for connecting to RabbitMQ, publishing messages,
// and managing the connection lifecycle.
//
// Features:
//   - RabbitMQ connection management
//   - Message publishing to durable queues
//   - JSON message serialization
//   - Clean resource cleanup
//   - Error handling with wrapped errors
//
// The package is designed to work with the FlowProcessMessage type from the eve package,
// making it suitable for process flow management systems.
package queue

import (
	"encoding/json"
	"fmt"
	"log"

	eve "eve.evalgo.org/common"
	"github.com/streadway/amqp"
)

// MessagePublisher defines the interface for publishing flow process messages.
// This interface allows for easy mocking and testing of message publishing functionality.
type MessagePublisher interface {
	// PublishMessage publishes a flow process message to the queue.
	// Returns an error if message serialization or publishing fails.
	PublishMessage(message eve.FlowProcessMessage) error

	// Close closes the connection to the message queue.
	// Returns an error if closing fails.
	Close() error
}

// RabbitMQService represents a service for interacting with RabbitMQ.
// It manages a connection and channel to a RabbitMQ server and provides
// methods for publishing messages to a queue.
//
// Fields:
//   - connection: The AMQP connection to the RabbitMQ server
//   - channel: The AMQP channel used for publishing messages
//   - config: Configuration for the RabbitMQ connection and queue
type RabbitMQService struct {
	connection AMQPConnection
	channel    AMQPChannel
	config     eve.FlowConfig
}

// NewRabbitMQService creates a new RabbitMQ service with the provided configuration.
// This function establishes a connection to RabbitMQ, opens a channel,
// and declares the queue specified in the configuration.
//
// Parameters:
//   - config: Configuration containing RabbitMQ URL and queue name
//
// Returns:
//   - *RabbitMQService: A new RabbitMQ service instance
//   - error: If connection, channel creation, or queue declaration fails
//
// The function:
//  1. Connects to the RabbitMQ server using the URL from config
//  2. Opens a channel on the connection
//  3. Declares a durable queue with the name from config
//  4. Returns a new RabbitMQService instance
//
// The queue is declared as durable, meaning it will survive server restarts.
// If any step fails, the function cleans up any created resources before returning the error.
func NewRabbitMQService(config eve.FlowConfig) (*RabbitMQService, error) {
	dialer := &RealAMQPDialer{}
	return NewRabbitMQServiceWithDialer(config, dialer)
}

// NewRabbitMQServiceWithDialer creates a new RabbitMQ service with dependency injection.
// This function allows injecting a custom dialer for testing purposes.
func NewRabbitMQServiceWithDialer(config eve.FlowConfig, dialer AMQPDialer) (*RabbitMQService, error) {
	// Connect to RabbitMQ
	conn, err := dialer.Dial(config.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Open a channel
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare the queue as durable
	_, err = ch.QueueDeclare(
		config.QueueName, // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &RabbitMQService{
		connection: conn,
		channel:    ch,
		config:     config,
	}, nil
}

// PublishMessage publishes a message to the RabbitMQ queue.
// This function serializes the message to JSON and publishes it to the queue
// specified in the service configuration.
//
// Parameters:
//   - message: The message to publish (will be marshaled to JSON)
//
// Returns:
//   - error: If message marshaling or publishing fails
//
// The function:
//  1. Marshals the message to JSON
//  2. Publishes the message to the default exchange with the queue name as routing key
//  3. Logs the published message's process ID
func (r *RabbitMQService) PublishMessage(message eve.FlowProcessMessage) error {
	// Marshal the message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish the message to the queue
	err = r.channel.Publish(
		"",                 // exchange (empty string means default exchange)
		r.config.QueueName, // routing key (queue name)
		false,              // mandatory
		false,              // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published message for process ID: %s", message.ProcessID)
	return nil
}

// Close closes the RabbitMQ connection and channel.
// This method should be called when the RabbitMQService is no longer needed
// to properly clean up resources.
//
// The function:
//  1. Closes the channel if it exists
//  2. Closes the connection if it exists
//  3. Handles nil pointers gracefully
//
// Returns:
//   - error: Always returns nil in the current implementation
func (r *RabbitMQService) Close() error {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.connection != nil {
		r.connection.Close()
	}
	return nil
}
