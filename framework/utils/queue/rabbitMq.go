package queue

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stratosnet/sds/framework/utils"
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

	return q
}

func (q *Queue) DeclareQueue(name string) {
	var err error
	q.queue, err = q.channel.QueueDeclare(
		name,  // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		utils.DebugLog(err.Error())
	}
}

func (q *Queue) GetQueueName() string {
	return q.queue.Name
}

func (q *Queue) Subscribe(key string) error {
	err := q.channel.QueueBind(q.queue.Name, key, EXCHANGE, false, nil)
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

func (q *Queue) Publish(key string, body []byte) error {
	return q.channel.Publish(EXCHANGE, key, false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})
}
