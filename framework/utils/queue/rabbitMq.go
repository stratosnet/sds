package queue

import (
	"strconv"

	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QUEUE_NAME           = "database_topics"
	EXCHANGE             = "all_actions"
	DEAD_LETTER_EXCHANGE = "dl_all_actions"
	EXCHANGE_IMPORT      = "data_migration"
)

type queueParameters struct {
	url          string
	exchangeName string
	noWait       bool
}

// RabbitMQ client
type Queue struct {
	conn          *amqp.Connection
	channel       *amqp.Channel
	queue         amqp.Queue
	dlqueue       amqp.Queue
	msgs          <-chan amqp.Delivery
	params        queueParameters
	connNotify    chan *amqp.Error
	channelNotify chan *amqp.Error
	channelReturn chan amqp.Return
}

func NewQueue(url string) (*Queue, error) {
	q := &Queue{
		params: queueParameters{
			url:          url,
			exchangeName: EXCHANGE,
			noWait:       false,
		},
	}
	err := q.connect()
	if err != nil {
		return nil, err
	}
	return q, nil
}

func NewQueueWithParams(url string, exchangeName string, noWait bool) (*Queue, error) {
	q := &Queue{
		params: queueParameters{
			url:          url,
			exchangeName: exchangeName,
			noWait:       noWait,
		},
	}
	err := q.connect()
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (q *Queue) connect() error {
	var err error
	q.conn, err = amqp.Dial(q.params.url)
	if err != nil {
		return errors.Wrap(err, "failed dialing RabbitMQ server")
	}
	q.channel, _ = q.conn.Channel()
	err = q.channel.ExchangeDeclare(
		q.params.exchangeName, // name
		amqp.ExchangeTopic,    // type
		true,                  // durable
		false,                 // auto-deleted
		false,                 // internal
		q.params.noWait,       // no-wait
		nil,                   // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed declaring the exchange in RabbitMQ")
	}
	if q.connNotify == nil {
		q.connNotify = make(chan *amqp.Error) // Don't recreate chan on reconnect
	}
	if q.channelNotify == nil {
		q.channelNotify = make(chan *amqp.Error) // Don't recreate chan on reconnect
	}
	if q.channelReturn == nil {
		q.channelReturn = make(chan amqp.Return) // Don't recreate chan on reconnect
	}
	q.conn.NotifyClose(q.connNotify)
	q.channel.NotifyClose(q.channelNotify)
	q.channel.NotifyReturn(q.channelReturn)

	return nil
}

func (q *Queue) Reconnect() error {
	if q.conn != nil {
		_ = q.conn.Close()
	}
	return q.connect()
}

func (q *Queue) DeclareQueue(name string) error {
	var err error
	args := amqp.Table{ // queue args
		amqp.QueueMaxLenArg:   10,
		amqp.QueueTypeArg:     amqp.QueueTypeQuorum,
		amqp.QueueOverflowArg: amqp.QueueOverflowRejectPublish,
	}

	q.queue, err = q.channel.QueueDeclare(
		name,  // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,  // arguments
	)
	return err
}

func (q *Queue) DeclareDeadLetterQueue(key string) error {
	var err error
	name := "dl_" + key
	// declare the dead letter exchange
	err = q.channel.ExchangeDeclare(
		DEAD_LETTER_EXCHANGE, // name
		amqp.ExchangeTopic,   // type
		true,                 // durable
		false,                // auto-deleted
		false,                // internal
		false,                // no-wait
		nil,                  // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed declaring the exchange in RabbitMQ")
	}

	args := amqp.Table{ // queue args
		amqp.QueueMaxLenArg:      10,
		amqp.QueueOverflowArg:    amqp.QueueOverflowRejectPublish,
		"x-dead-letter-exchange": EXCHANGE,
	}
	q.dlqueue, err = q.channel.QueueDeclare(
		name,  // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,  // arguments
	)
	if err != nil {
		return err
	}

	err = q.channel.QueueBind(q.dlqueue.Name, key, DEAD_LETTER_EXCHANGE, false, nil)
	if err != nil {
		return err
	}
	return err
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

func (q *Queue) GetConnNotify() chan *amqp.Error {
	return q.connNotify
}

func (q *Queue) GetChannelNotify() chan *amqp.Error {
	return q.channelNotify
}

func (q *Queue) GetChannelReturn() chan amqp.Return {
	return q.channelReturn
}

func (q *Queue) GetMsg() (<-chan amqp.Delivery, error) {
	var err error
	q.msgs, err = q.channel.Consume(q.queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	return q.msgs, nil
}

func (q *Queue) GetDLMsg() (<-chan amqp.Delivery, error) {
	var err error
	q.msgs, err = q.channel.Consume(q.dlqueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	return q.msgs, nil
}

func (q *Queue) GetMsgWithManualAck() (<-chan amqp.Delivery, error) {
	var err error
	q.msgs, err = q.channel.Consume(q.queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	return q.msgs, nil
}

func (q *Queue) Publish(key string, body []byte) error {
	return q.channel.Publish(EXCHANGE, key, true, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})
}

func (q *Queue) SendDeadLetter(key string, body []byte, retry_count int32) error {
	expiration := strconv.FormatInt(int64(retry_count^2*10000), 10)
	return q.channel.Publish(DEAD_LETTER_EXCHANGE, key, true, false,
		amqp.Publishing{
			DeliveryMode: 2, //persistent
			Headers:      amqp.Table{"retry_count": retry_count},
			Expiration:   expiration,
			ContentType:  "text/plain",
			Body:         body,
		})
}
