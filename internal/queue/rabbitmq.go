package queue

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQBroker struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func DialRabbitMQ(url string) (*RabbitMQBroker, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	return &RabbitMQBroker{
		conn:    conn,
		channel: channel,
	}, nil
}

func (b *RabbitMQBroker) Close() error {
	if b == nil {
		return nil
	}

	var closeErr error
	if b.channel != nil {
		closeErr = b.channel.Close()
	}
	if b.conn != nil {
		if err := b.conn.Close(); closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func (b *RabbitMQBroker) DeclareExchange(ctx context.Context, exchange ExchangeSpec) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return b.channel.ExchangeDeclare(
		exchange.Name,
		exchange.Kind,
		exchange.Durable,
		false,
		false,
		false,
		nil,
	)
}

func (b *RabbitMQBroker) DeclareQueue(ctx context.Context, queue QueueSpec) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	args := amqp.Table{}
	if !queue.DeadLetter {
		if queue.DeadLetterExchange != "" {
			args["x-dead-letter-exchange"] = queue.DeadLetterExchange
		}
		if queue.DeadLetterRoutingKey != "" {
			args["x-dead-letter-routing-key"] = queue.DeadLetterRoutingKey
		}
	}

	_, err := b.channel.QueueDeclare(
		queue.Name,
		queue.Durable,
		false,
		false,
		false,
		args,
	)
	return err
}

func (b *RabbitMQBroker) BindQueue(ctx context.Context, binding BindingSpec) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return b.channel.QueueBind(
		binding.Queue,
		binding.RoutingKey,
		binding.Exchange,
		false,
		nil,
	)
}

func (b *RabbitMQBroker) Publish(ctx context.Context, exchange, routingKey string, content PublishContent) error {
	deliveryMode := amqp.Transient
	if content.Persistent {
		deliveryMode = amqp.Persistent
	}

	return b.channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  content.ContentType,
			DeliveryMode: deliveryMode,
			Body:         content.Body,
		},
	)
}

func (b *RabbitMQBroker) SetPrefetch(count int) error {
	return b.channel.Qos(count, 0, false)
}

func (b *RabbitMQBroker) Consume(queue string) (<-chan amqp.Delivery, error) {
	return b.channel.Consume(
		queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}
