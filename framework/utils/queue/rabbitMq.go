package queue

import (
	"github.com/stratosnet/sds/framework/utils"
	"strconv"
	"time"

	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	RECONNECT_INTERVAL   = 5 * time.Second
	QUEUE_NAME           = "database_topics"
	EXCHANGE             = "all_actions"
	DEAD_LETTER_EXCHANGE = "dl_all_actions"
	EXCHANGE_IMPORT      = "data_migration"
)

// Redis client
type Queue struct {
	conn          *amqp.Connection
	channel       *amqp.Channel
	queue         amqp.Queue
	dlqueue       amqp.Queue
	msgs          <-chan amqp.Delivery
	connNotify    chan *amqp.Error
	channelNotify chan *amqp.Error
	done          chan bool
	isConnected   bool
}

func NewQueue(url string) *Queue {
	q := &Queue{
		done: make(chan bool),
	}
	go q.handleReconnect(url, EXCHANGE, false)

	for {
		if q.isConnected {
			break
		}
	}

	return q
}

func NewQueueWithParams(url string, exchangeName string, noWait bool) *Queue {
	q := &Queue{
		done: make(chan bool),
	}
	go q.handleReconnect(url, exchangeName, noWait)

	for {
		if q.isConnected {
			break
		}
	}

	return q
}

func (q *Queue) handleReconnect(url string, exchangeName string, noWait bool) {
	for {
		if !q.isConnected {
			for !q.connect(url, exchangeName, noWait) {
				utils.DebugLog("Attempting to connect to RabbitMq...")
				time.Sleep(RECONNECT_INTERVAL)
			}
		}

		select {
		case <-q.done:
			return
		case <-q.channelNotify:
			q.isConnected = false
		case <-q.connNotify:
			q.isConnected = false
		}
		time.Sleep(RECONNECT_INTERVAL)
	}
}

func (q *Queue) connect(url string, exchangeName string, noWait bool) bool {
	conn, err := amqp.Dial(url)
	if err != nil {
		utils.ErrorLogf("Failed to connect to RabbitMq: %v", err)
		return false
	}
	ch, err := conn.Channel()
	if err != nil {
		utils.ErrorLogf("Failed to open RabbitMq channel: %v", err)
		return false
	}
	err = ch.ExchangeDeclare(
		exchangeName,       // name
		amqp.ExchangeTopic, // type
		true,               // durable
		false,              // auto-deleted
		false,              // internal
		noWait,             // no-wait
		nil,                // arguments
	)
	if err != nil {
		utils.ErrorLogf("Failed declaring the exchange in RabbitMq: %v", err)
		return false
	}

	q.setConnection(conn, ch)
	q.isConnected = true
	utils.Log("Connected to RabbitMq")
	return true
}

func (q *Queue) setConnection(connection *amqp.Connection, channel *amqp.Channel) {
	q.conn = connection
	q.channel = channel
	q.connNotify = make(chan *amqp.Error)
	q.channelNotify = make(chan *amqp.Error)
	q.conn.NotifyClose(q.connNotify)
	q.channel.NotifyClose(q.channelNotify)
}

func (q *Queue) Close() error {
	if !q.isConnected {
		return errors.New("Close connection error: RabbitMq connection already closed")
	}
	q.done <- true
	err := q.channel.Close()
	if err != nil {
		return err
	}
	err = q.conn.Close()
	if err != nil {
		return err
	}
	close(q.done)
	q.isConnected = false
	return nil
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
		return errors.Wrap(err, "Failed declaring the exchange in RabbitMq")
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

func (q *Queue) GetMsg() <-chan amqp.Delivery {
	var err error
	q.msgs, err = q.channel.Consume(q.queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil
	}
	return q.msgs
}

func (q *Queue) GetDLMsg() <-chan amqp.Delivery {
	var err error
	q.msgs, err = q.channel.Consume(q.dlqueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil
	}
	return q.msgs
}

func (q *Queue) GetMsgWithManualAck() <-chan amqp.Delivery {
	var err error
	q.msgs, err = q.channel.Consume(q.queue.Name, "", false, false, false, false, nil)
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

func (q *Queue) SendDeadLetter(key string, body []byte, retry_count int32) error {
	expiration := strconv.FormatInt(int64(retry_count^2*10000), 10)
	return q.channel.Publish(DEAD_LETTER_EXCHANGE, key, false, false,
		amqp.Publishing{
			DeliveryMode: 2, //persistent
			Headers:      amqp.Table{"retry_count": retry_count},
			Expiration:   expiration,
			ContentType:  "text/plain",
			Body:         body,
		})
}
