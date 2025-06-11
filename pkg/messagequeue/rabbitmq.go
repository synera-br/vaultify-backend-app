package messagequeue

import (
	"log"
	"github.com/streadway/amqp"
)

// RabbitMQService implements the MessageQueue interface using RabbitMQ.
type RabbitMQService struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQServiceConfig contains options for creating a new RabbitMQService.
type NewRabbitMQServiceConfig struct {
	URL string
}

// NewRabbitMQService creates a new instance of RabbitMQService.
func NewRabbitMQService(cfg NewRabbitMQServiceConfig) (MessageQueue, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open a channel: %v", err)
		conn.Close() // Close connection if channel opening fails
		return nil, err
	}

	log.Println("Successfully connected to RabbitMQ and opened a channel")
	return &RabbitMQService{conn: conn, channel: ch}, nil
}

// Publish sends a message to a RabbitMQ queue.
func (s *RabbitMQService) Publish(queueName string, body []byte) error {
	q, err := s.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Printf("Failed to declare a queue %s: %v", queueName, err)
		return err
	}

	err = s.channel.Publish(
		"",     // exchange
		q.Name, // routing key (queue name)
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
			DeliveryMode: amqp.Persistent, // Make message persistent
		})
	if err != nil {
		log.Printf("Failed to publish a message to queue %s: %v", queueName, err)
		return err
	}
	log.Printf("Successfully published message to queue %s", queueName)
	return nil
}

// Consume starts consuming messages from a RabbitMQ queue.
// The handler function is called for each received message.
func (s *RabbitMQService) Consume(queueName string, handler func(body []byte)) error {
	q, err := s.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Printf("Failed to declare a queue %s for consuming: %v", queueName, err)
		return err
	}

	msgs, err := s.channel.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack (set to false for manual ack)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Printf("Failed to register a consumer for queue %s: %v", queueName, err)
		return err
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message from queue %s: %s", queueName, d.Body)
			handler(d.Body)
		}
	}()

	log.Printf("Waiting for messages on queue %s. To exit press CTRL+C", q.Name)
	<-forever // Keep the consumer running

	return nil
}

// Close closes the RabbitMQ channel and connection.
func (s *RabbitMQService) Close() error {
	var lastErr error
	if s.channel != nil {
		if err := s.channel.Close(); err != nil {
			log.Printf("Error closing RabbitMQ channel: %v", err)
			lastErr = err
		}
	}
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			log.Printf("Error closing RabbitMQ connection: %v", err)
			lastErr = err
		}
	}
	if lastErr == nil {
		log.Println("RabbitMQ channel and connection closed successfully.")
	}
	return lastErr
}
