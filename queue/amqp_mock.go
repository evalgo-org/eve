package queue

import (
	"fmt"

	"github.com/streadway/amqp"
)

// MockAMQPConnection is a mock implementation of AMQPConnection for testing
type MockAMQPConnection struct {
	// MockChannel is the channel to return from Channel()
	MockChannel AMQPChannel
	// Error to return from operations
	ChannelErr error
	CloseErr   error
	// Track function calls
	ChannelCalled bool
	CloseCalled   bool
}

// Channel returns the mock channel
func (m *MockAMQPConnection) Channel() (AMQPChannel, error) {
	m.ChannelCalled = true
	if m.ChannelErr != nil {
		return nil, m.ChannelErr
	}
	return m.MockChannel, nil
}

// Close mocks closing the connection
func (m *MockAMQPConnection) Close() error {
	m.CloseCalled = true
	return m.CloseErr
}

// MockAMQPChannel is a mock implementation of AMQPChannel for testing
type MockAMQPChannel struct {
	// PublishedMessages stores all published messages for verification
	PublishedMessages []amqp.Publishing
	// PublishedKeys stores routing keys for published messages
	PublishedKeys []string
	// Errors to return from operations
	QueueDeclareErr error
	PublishErr      error
	CloseErr        error
	// Track function calls
	QueueDeclareCalled bool
	PublishCalled      bool
	CloseCalled        bool
	// Store last call parameters
	LastQueueName string
	LastExchange  string
	LastKey       string
}

// QueueDeclare mocks declaring a queue
func (m *MockAMQPChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	m.QueueDeclareCalled = true
	m.LastQueueName = name
	if m.QueueDeclareErr != nil {
		return amqp.Queue{}, m.QueueDeclareErr
	}
	return amqp.Queue{
		Name:      name,
		Messages:  0,
		Consumers: 0,
	}, nil
}

// Publish mocks publishing a message
func (m *MockAMQPChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	m.PublishCalled = true
	m.LastExchange = exchange
	m.LastKey = key
	if m.PublishErr != nil {
		return m.PublishErr
	}
	m.PublishedMessages = append(m.PublishedMessages, msg)
	m.PublishedKeys = append(m.PublishedKeys, key)
	return nil
}

// Close mocks closing the channel
func (m *MockAMQPChannel) Close() error {
	m.CloseCalled = true
	return m.CloseErr
}

// MockAMQPDialer is a mock implementation of AMQPDialer for testing
type MockAMQPDialer struct {
	// MockConnection is the connection to return from Dial()
	MockConnection AMQPConnection
	// Error to return from Dial
	DialErr error
	// Track function calls
	DialCalled bool
	// Store last call parameters
	LastURL string
}

// Dial mocks dialing an AMQP connection
func (m *MockAMQPDialer) Dial(url string) (AMQPConnection, error) {
	m.DialCalled = true
	m.LastURL = url
	if m.DialErr != nil {
		return nil, m.DialErr
	}
	return m.MockConnection, nil
}

// NewMockAMQPDialer creates a new mock AMQP dialer with a successful setup
func NewMockAMQPDialer() *MockAMQPDialer {
	mockChannel := &MockAMQPChannel{
		PublishedMessages: make([]amqp.Publishing, 0),
		PublishedKeys:     make([]string, 0),
	}

	mockConn := &MockAMQPConnection{
		MockChannel: mockChannel,
	}

	return &MockAMQPDialer{
		MockConnection: mockConn,
	}
}

// NewMockAMQPDialerWithError creates a mock dialer that returns an error
func NewMockAMQPDialerWithError(err error) *MockAMQPDialer {
	return &MockAMQPDialer{
		DialErr: err,
	}
}

// GetMockChannel is a helper to get the mock channel from the dialer
func (m *MockAMQPDialer) GetMockChannel() *MockAMQPChannel {
	if m.MockConnection == nil {
		return nil
	}
	mockConn, ok := m.MockConnection.(*MockAMQPConnection)
	if !ok || mockConn.MockChannel == nil {
		return nil
	}
	ch, ok := mockConn.MockChannel.(*MockAMQPChannel)
	if !ok {
		return nil
	}
	return ch
}

// SetupMockDialerForTest creates a fully configured mock dialer for testing
func SetupMockDialerForTest() (*MockAMQPDialer, *MockAMQPChannel, *MockAMQPConnection) {
	mockChannel := &MockAMQPChannel{
		PublishedMessages: make([]amqp.Publishing, 0),
		PublishedKeys:     make([]string, 0),
	}

	mockConn := &MockAMQPConnection{
		MockChannel: mockChannel,
	}

	mockDialer := &MockAMQPDialer{
		MockConnection: mockConn,
	}

	return mockDialer, mockChannel, mockConn
}

// SetupMockDialerWithChannelError creates a mock dialer that fails on channel creation
func SetupMockDialerWithChannelError() *MockAMQPDialer {
	mockConn := &MockAMQPConnection{
		ChannelErr: fmt.Errorf("failed to open channel"),
	}

	return &MockAMQPDialer{
		MockConnection: mockConn,
	}
}

// SetupMockDialerWithQueueError creates a mock dialer that fails on queue declaration
func SetupMockDialerWithQueueError() (*MockAMQPDialer, *MockAMQPChannel) {
	mockChannel := &MockAMQPChannel{
		QueueDeclareErr: fmt.Errorf("failed to declare queue"),
	}

	mockConn := &MockAMQPConnection{
		MockChannel: mockChannel,
	}

	mockDialer := &MockAMQPDialer{
		MockConnection: mockConn,
	}

	return mockDialer, mockChannel
}
