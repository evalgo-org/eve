package queue

import (
	"github.com/streadway/amqp"
)

// AMQPConnection defines the interface for AMQP connection operations.
// This interface abstracts the RabbitMQ connection to enable dependency injection
// and testing with mock implementations.
type AMQPConnection interface {
	// Channel opens a channel on the connection
	Channel() (AMQPChannel, error)

	// Close closes the connection
	Close() error
}

// AMQPChannel defines the interface for AMQP channel operations.
// This interface abstracts the RabbitMQ channel to enable dependency injection
// and testing with mock implementations.
type AMQPChannel interface {
	// QueueDeclare declares a queue
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)

	// Publish publishes a message to the specified exchange
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error

	// Consume starts consuming messages from a queue
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)

	// QueueInspect retrieves queue information
	QueueInspect(name string) (amqp.Queue, error)

	// Close closes the channel
	Close() error
}

// AMQPDialer defines the interface for dialing AMQP connections.
// This interface allows injecting custom dialers for testing.
type AMQPDialer interface {
	// Dial connects to the AMQP server
	Dial(url string) (AMQPConnection, error)
}

// RealAMQPConnection wraps a real amqp.Connection to implement AMQPConnection interface
type RealAMQPConnection struct {
	conn *amqp.Connection
}

// Channel opens a channel on the real connection
func (r *RealAMQPConnection) Channel() (AMQPChannel, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, err
	}
	return &RealAMQPChannel{ch: ch}, nil
}

// Close closes the real connection
func (r *RealAMQPConnection) Close() error {
	return r.conn.Close()
}

// RealAMQPChannel wraps a real amqp.Channel to implement AMQPChannel interface
type RealAMQPChannel struct {
	ch *amqp.Channel
}

// QueueDeclare declares a queue on the real channel
func (r *RealAMQPChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return r.ch.QueueDeclare(name, durable, autoDelete, exclusive, noWait, args)
}

// Publish publishes a message to the real channel
func (r *RealAMQPChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return r.ch.Publish(exchange, key, mandatory, immediate, msg)
}

// Consume starts consuming messages from a queue on the real channel
func (r *RealAMQPChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return r.ch.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

// QueueInspect retrieves queue information from the real channel
func (r *RealAMQPChannel) QueueInspect(name string) (amqp.Queue, error) {
	return r.ch.QueueInspect(name)
}

// Close closes the real channel
func (r *RealAMQPChannel) Close() error {
	return r.ch.Close()
}

// RealAMQPDialer implements AMQPDialer using the real AMQP library
type RealAMQPDialer struct{}

// Dial connects to the AMQP server using the real library
func (r *RealAMQPDialer) Dial(url string) (AMQPConnection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return &RealAMQPConnection{conn: conn}, nil
}
