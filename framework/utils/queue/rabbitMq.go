package queue

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QUEUE_NAME = "database_topics"
)

// Redis client
type Queue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	msgs    <-chan amqp.Delivery
}

func NewQueue() *Queue {
	q := &Queue{}
	q.conn, _ = amqp.Dial("amqp://guest:guest@localhost:5672/")
	q.channel, _ = q.conn.Channel()
	err := q.channel.ExchangeDeclare(
		QUEUE_NAME, // name
		"topic",    // type
		true,       // durable
		false,      // auto-deleted
		false,      // internal
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {

	}
	return q
}

func (q *Queue) Subscribe(action string) error {
	err := q.channel.QueueBind(QUEUE_NAME, action, "all_actions", false, nil)
	if err != nil {
		return err
	}
	q.msgs, err = q.channel.Consume(QUEUE_NAME, action, true, false, false, false, nil)
	return nil
}

func (q *Queue) GetMsg() <-chan amqp.Delivery {
	return q.msgs
}
