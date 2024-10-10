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
	queue   amqp.Queue
	msgs    <-chan amqp.Delivery
}

func NewQueue(url string) *Queue {
	q := &Queue{}
	var err error
	q.conn, err = amqp.Dial(url)
	if err != nil {
		return nil
	}
	q.channel, _ = q.conn.Channel()
	err = q.channel.ExchangeDeclare(
		EXCHANGE, // name
		"topic",  // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {

	}
	q.queue, _ = q.channel.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	return q
}

func (q *Queue) Subscribe(action string) error {
	err := q.channel.QueueBind(q.queue.Name, action, EXCHANGE, false, nil)
	if err != nil {
		return err
	}

	return nil
}

func (q *Queue) GetMsg() <-chan amqp.Delivery {
	var err error
	q.msgs, err = q.channel.Consume(q.queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil
	}
	return q.msgs
}

func (q *Queue) Publish(action string, body []byte) error {
	return q.channel.Publish(EXCHANGE, action, false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})
}
