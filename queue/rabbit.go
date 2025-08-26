package queue

import (
	"encoding/json"
	"fmt"
	"log"

	eve "eve.evalgo.org/common"
	"github.com/streadway/amqp"
)

type RabbitMQService struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	config     eve.FlowConfig
}

func NewRabbitMQService(config eve.FlowConfig) (*RabbitMQService, error) {
	conn, err := amqp.Dial(config.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare the queue
	_, err = ch.QueueDeclare(
		config.QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
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

func (r *RabbitMQService) PublishMessage(message eve.FlowProcessMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = r.channel.Publish(
		"",                 // exchange
		r.config.QueueName, // routing key
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

func (r *RabbitMQService) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.connection != nil {
		r.connection.Close()
	}
}
