package messagequeue

// MessageQueue defines the interface for message queue services.
type MessageQueue interface {
	Publish(queueName string, body []byte) error
	Consume(queueName string, handler func(body []byte)) error
	Close() error
}
