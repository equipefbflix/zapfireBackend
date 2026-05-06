package workerapp

import "context"

type BodyHandler interface {
	Handle(ctx context.Context, body []byte) error
}

type Delivery interface {
	Body() []byte
	Ack() error
	Nack(requeue bool) error
	Attempt() int
}

func ProcessDelivery(ctx context.Context, handler BodyHandler, delivery Delivery, maxRetries int) error {
	if err := handler.Handle(ctx, delivery.Body()); err != nil {
		requeue := delivery.Attempt() <= maxRetries
		_ = delivery.Nack(requeue)
		return err
	}
	return delivery.Ack()
}
