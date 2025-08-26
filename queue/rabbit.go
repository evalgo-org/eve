package queue

import (
	"context"
	"time"

	eve "eve.evalgo.org/common"
	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		eve.Logger.Info(msg+":", err)
	}
}

func RabbitMQPublish(connectionUrl, queueName string, message []byte) error {
	conn, err := amqp.Dial(connectionUrl)
	defer conn.Close()
	if err != nil {
		failOnError(err, "Failed to connect to RabbitMQ")
		return err
	}
	ch, err := conn.Channel()
	defer ch.Close()
	if err != nil {
		failOnError(err, "Failed to open a channel")
		return err
	}
	q, err := ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		failOnError(err, "Failed to declare a queue")
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		},
	)
	if err != nil {
		failOnError(err, "Failed to publish a message")
		return err
	}
	eve.Logger.Info(" [x] Sent ", string(message))

	return nil
}
