package queue

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QUEUE_NAME = "database_topics"
	EXCHANGE   = "all_actions"
)

// Redis client
type Queue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	msgs    <-chan amqp.Delivery
}

func NewQueue(url string) *Queue {
	q := &Queue{}
	q.conn, _ = amqp.Dial(url)
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
	err := q.channel.QueueBind(QUEUE_NAME, action, EXCHANGE, false, nil)
	if err != nil {
		return err
	}
	q.msgs, err = q.channel.Consume(QUEUE_NAME, action, true, false, false, false, nil)
	return nil
}

func (q *Queue) GetMsg() <-chan amqp.Delivery {
	return q.msgs
}

func (q *Queue) Publish(action string, body []byte) error {
	return q.channel.Publish(EXCHANGE, action, false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})
}
