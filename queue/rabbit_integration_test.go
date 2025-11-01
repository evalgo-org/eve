//go:build integration

package queue

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	eve "eve.evalgo.org/common"
)

// setupRabbitMQContainer starts a RabbitMQ container for testing
func setupRabbitMQContainer(t *testing.T) (string, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "rabbitmq:3.13-management-alpine",
		ExposedPorts: []string{"5672/tcp", "15672/tcp"},
		Env: map[string]string{
			"RABBITMQ_DEFAULT_USER": "guest",
			"RABBITMQ_DEFAULT_PASS": "guest",
		},
		WaitingFor: wait.ForLog("Server startup complete").
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start RabbitMQ container")

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5672")
	require.NoError(t, err)

	url := fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port.Port())

	// Wait a bit for RabbitMQ to be fully ready
	time.Sleep(2 * time.Second)

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return url, cleanup
}

// TestRabbitMQService_Integration_NewService tests service creation
func TestRabbitMQService_Integration_NewService(t *testing.T) {
	url, cleanup := setupRabbitMQContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		RabbitMQURL: url,
		QueueName:   "test_queue",
	}

	t.Run("create service successfully", func(t *testing.T) {
		service, err := NewRabbitMQService(config)
		require.NoError(t, err, "Failed to create RabbitMQ service")
		assert.NotNil(t, service)
		assert.NotNil(t, service.connection)
		assert.NotNil(t, service.channel)
		service.Close()
	})

	t.Run("fail with invalid URL", func(t *testing.T) {
		badConfig := eve.FlowConfig{
			RabbitMQURL: "amqp://invalid:5672/",
			QueueName:   "test_queue",
		}

		service, err := NewRabbitMQService(badConfig)
		assert.Error(t, err, "Should fail with invalid URL")
		assert.Nil(t, service)
	})
}

// TestRabbitMQService_Integration_PublishMessage tests message publishing
func TestRabbitMQService_Integration_PublishMessage(t *testing.T) {
	url, cleanup := setupRabbitMQContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		RabbitMQURL: url,
		QueueName:   "test_publish_queue",
	}

	service, err := NewRabbitMQService(config)
	require.NoError(t, err)
	defer service.Close()

	t.Run("publish valid message", func(t *testing.T) {
		msg := eve.FlowProcessMessage{
			ProcessID:   "test-001",
			State:       eve.StateStarted,
			Timestamp:   time.Now(),
			Description: "Test message",
			Metadata: map[string]interface{}{
				"key": "value",
			},
		}

		err := service.PublishMessage(msg)
		require.NoError(t, err, "Failed to publish message")
	})

	t.Run("publish multiple messages", func(t *testing.T) {
		messages := []eve.FlowProcessMessage{
			{ProcessID: "test-002", State: eve.StateStarted, Timestamp: time.Now()},
			{ProcessID: "test-003", State: eve.StateRunning, Timestamp: time.Now()},
			{ProcessID: "test-004", State: eve.StateSuccessful, Timestamp: time.Now()},
		}

		for _, msg := range messages {
			err := service.PublishMessage(msg)
			require.NoError(t, err, "Failed to publish message %s", msg.ProcessID)
		}
	})

	t.Run("publish message with error state", func(t *testing.T) {
		msg := eve.FlowProcessMessage{
			ProcessID:   "test-error-001",
			State:       eve.StateFailed,
			Timestamp:   time.Now(),
			ErrorMsg:    "Test error message",
			Description: "Failed process",
		}

		err := service.PublishMessage(msg)
		require.NoError(t, err)
	})
}

// TestRabbitMQService_Integration_ConsumeMessages tests message consumption
func TestRabbitMQService_Integration_ConsumeMessages(t *testing.T) {
	url, cleanup := setupRabbitMQContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		RabbitMQURL: url,
		QueueName:   "test_consume_queue",
	}

	// Create service and publish messages
	service, err := NewRabbitMQService(config)
	require.NoError(t, err)
	defer service.Close()

	// Publish test messages
	messages := []eve.FlowProcessMessage{
		{ProcessID: "consume-001", State: eve.StateStarted, Timestamp: time.Now()},
		{ProcessID: "consume-002", State: eve.StateRunning, Timestamp: time.Now()},
		{ProcessID: "consume-003", State: eve.StateSuccessful, Timestamp: time.Now()},
	}

	for _, msg := range messages {
		err := service.PublishMessage(msg)
		require.NoError(t, err)
	}

	// Create a consumer to verify messages
	msgs, err := service.channel.Consume(
		config.QueueName, // queue
		"",               // consumer
		true,             // auto-ack
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,              // args
	)
	require.NoError(t, err)

	// Read messages with timeout
	timeout := time.After(5 * time.Second)
	receivedCount := 0

	for receivedCount < len(messages) {
		select {
		case msg := <-msgs:
			receivedCount++
			assert.NotEmpty(t, msg.Body, "Message body should not be empty")
			t.Logf("Received message %d: %s", receivedCount, string(msg.Body))
		case <-timeout:
			t.Fatalf("Timeout waiting for messages. Received %d of %d", receivedCount, len(messages))
		}
	}

	assert.Equal(t, len(messages), receivedCount, "Should receive all published messages")
}

// TestRabbitMQService_Integration_QueueProperties tests queue configuration
func TestRabbitMQService_Integration_QueueProperties(t *testing.T) {
	url, cleanup := setupRabbitMQContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		RabbitMQURL: url,
		QueueName:   "test_durable_queue",
	}

	service, err := NewRabbitMQService(config)
	require.NoError(t, err)
	defer service.Close()

	// Inspect queue properties
	queue, err := service.channel.QueueInspect(config.QueueName)
	require.NoError(t, err)

	assert.Equal(t, config.QueueName, queue.Name)
	assert.Greater(t, queue.Messages, -1, "Queue should exist and have message count >= 0")
}

// TestRabbitMQService_Integration_Close tests resource cleanup
func TestRabbitMQService_Integration_Close(t *testing.T) {
	url, cleanup := setupRabbitMQContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		RabbitMQURL: url,
		QueueName:   "test_close_queue",
	}

	t.Run("close gracefully", func(t *testing.T) {
		service, err := NewRabbitMQService(config)
		require.NoError(t, err)

		// Publish a message before closing
		msg := eve.FlowProcessMessage{
			ProcessID: "close-test-001",
			State:     eve.StateStarted,
			Timestamp: time.Now(),
		}
		err = service.PublishMessage(msg)
		require.NoError(t, err)

		// Close should not panic
		assert.NotPanics(t, func() {
			service.Close()
		})
	})

	t.Run("close multiple times", func(t *testing.T) {
		service, err := NewRabbitMQService(config)
		require.NoError(t, err)

		// Multiple closes should not panic
		assert.NotPanics(t, func() {
			service.Close()
			service.Close()
			service.Close()
		})
	})
}

// TestRabbitMQService_Integration_Reconnection tests connection recovery
func TestRabbitMQService_Integration_Reconnection(t *testing.T) {
	url, cleanup := setupRabbitMQContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		RabbitMQURL: url,
		QueueName:   "test_reconnect_queue",
	}

	service, err := NewRabbitMQService(config)
	require.NoError(t, err)
	defer service.Close()

	// Publish initial message
	msg1 := eve.FlowProcessMessage{
		ProcessID: "reconnect-001",
		State:     eve.StateStarted,
		Timestamp: time.Now(),
	}
	err = service.PublishMessage(msg1)
	require.NoError(t, err)

	// Close the connection
	service.Close()

	// Create new service (simulating reconnection)
	service2, err := NewRabbitMQService(config)
	require.NoError(t, err, "Should be able to reconnect")
	defer service2.Close()

	// Publish another message with new connection
	msg2 := eve.FlowProcessMessage{
		ProcessID: "reconnect-002",
		State:     eve.StateSuccessful,
		Timestamp: time.Now(),
	}
	err = service2.PublishMessage(msg2)
	require.NoError(t, err, "Should be able to publish after reconnection")
}

// TestRabbitMQService_Integration_ConcurrentPublish tests concurrent message publishing
func TestRabbitMQService_Integration_ConcurrentPublish(t *testing.T) {
	url, cleanup := setupRabbitMQContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		RabbitMQURL: url,
		QueueName:   "test_concurrent_queue",
	}

	service, err := NewRabbitMQService(config)
	require.NoError(t, err)
	defer service.Close()

	// Publish messages concurrently
	numMessages := 50
	var wg sync.WaitGroup
	errChan := make(chan error, numMessages)

	wg.Add(numMessages)
	for i := 0; i < numMessages; i++ {
		go func(id int) {
			defer wg.Done()
			msg := eve.FlowProcessMessage{
				ProcessID: fmt.Sprintf("concurrent-%d", id),
				State:     eve.StateRunning,
				Timestamp: time.Now(),
			}
			errChan <- service.PublishMessage(msg)
		}(i)
	}

	// Wait for all publishes to complete
	wg.Wait()
	close(errChan)

	// Check all publishes succeeded
	for err := range errChan {
		assert.NoError(t, err, "Concurrent publish should succeed")
	}

	// Give RabbitMQ a moment to process all messages
	time.Sleep(100 * time.Millisecond)

	// Verify queue has messages
	queue, err := service.channel.QueueInspect(config.QueueName)
	require.NoError(t, err)
	assert.Equal(t, numMessages, queue.Messages, "Queue should have all published messages")
}

// TestRabbitMQService_Integration_MessagePersistence tests message durability
func TestRabbitMQService_Integration_MessagePersistence(t *testing.T) {
	url, cleanup := setupRabbitMQContainer(t)
	defer cleanup()

	queueName := "test_persistent_queue"
	config := eve.FlowConfig{
		RabbitMQURL: url,
		QueueName:   queueName,
	}

	// Publish messages
	service1, err := NewRabbitMQService(config)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		msg := eve.FlowProcessMessage{
			ProcessID: fmt.Sprintf("persistent-%d", i),
			State:     eve.StateStarted,
			Timestamp: time.Now(),
		}
		err := service1.PublishMessage(msg)
		require.NoError(t, err)
	}

	// Close service
	service1.Close()

	// Create new service and verify messages still exist
	service2, err := NewRabbitMQService(config)
	require.NoError(t, err)
	defer service2.Close()

	queue, err := service2.channel.QueueInspect(queueName)
	require.NoError(t, err)
	assert.Equal(t, 5, queue.Messages, "Messages should persist after reconnection")
}

// TestRabbitMQService_Integration_LargeMessages tests handling of large messages
func TestRabbitMQService_Integration_LargeMessages(t *testing.T) {
	url, cleanup := setupRabbitMQContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		RabbitMQURL: url,
		QueueName:   "test_large_queue",
	}

	service, err := NewRabbitMQService(config)
	require.NoError(t, err)
	defer service.Close()

	// Create large metadata
	largeMetadata := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeMetadata[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d_with_some_extra_data_to_make_it_larger", i)
	}

	msg := eve.FlowProcessMessage{
		ProcessID:   "large-message-001",
		State:       eve.StateRunning,
		Timestamp:   time.Now(),
		Description: "Message with large metadata",
		Metadata:    largeMetadata,
	}

	err = service.PublishMessage(msg)
	require.NoError(t, err, "Should be able to publish large messages")
}
