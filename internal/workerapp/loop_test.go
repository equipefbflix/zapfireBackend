package workerapp

import (
	"context"
	"errors"
	"testing"
)

type fakeDelivery struct {
	body        []byte
	acked       bool
	nacked      bool
	nackRequeue bool
	attempt     int
}

func (d *fakeDelivery) Body() []byte {
	return d.body
}

func (d *fakeDelivery) Ack() error {
	d.acked = true
	return nil
}

func (d *fakeDelivery) Nack(requeue bool) error {
	d.nacked = true
	d.nackRequeue = requeue
	return nil
}

func (d *fakeDelivery) Attempt() int {
	if d.attempt <= 0 {
		return 1
	}
	return d.attempt
}

type fakeDeliveryConsumer struct {
	handler func(context.Context, []byte) error
}

func (c fakeDeliveryConsumer) Handle(ctx context.Context, body []byte) error {
	return c.handler(ctx, body)
}

func TestProcessDeliveryAckOnSuccess(t *testing.T) {
	delivery := &fakeDelivery{body: []byte(`{"type":"warming.job.due"}`)}

	err := ProcessDelivery(context.Background(), fakeDeliveryConsumer{
		handler: func(ctx context.Context, body []byte) error { return nil },
	}, delivery, 3)
	if err != nil {
		t.Fatalf("ProcessDelivery() error = %v", err)
	}
	if !delivery.acked {
		t.Fatal("acked = false")
	}
	if delivery.nacked {
		t.Fatal("nacked = true")
	}
}

func TestProcessDeliveryNackOnHandlerError(t *testing.T) {
	delivery := &fakeDelivery{body: []byte(`{"type":"warming.job.due"}`)}

	err := ProcessDelivery(context.Background(), fakeDeliveryConsumer{
		handler: func(ctx context.Context, body []byte) error { return errors.New("boom") },
	}, delivery, 3)
	if err == nil {
		t.Fatal("ProcessDelivery() error = nil")
	}
	if !delivery.nacked {
		t.Fatal("nacked = false")
	}
	if !delivery.nackRequeue {
		t.Fatal("nackRequeue = false")
	}
}

func TestProcessDeliveryNackDeadLetterWhenAttemptsExceeded(t *testing.T) {
	delivery := &fakeDelivery{body: []byte(`{"type":"warming.job.due"}`), attempt: 4}

	err := ProcessDelivery(context.Background(), fakeDeliveryConsumer{
		handler: func(ctx context.Context, body []byte) error { return errors.New("boom") },
	}, delivery, 3)
	if err == nil {
		t.Fatal("ProcessDelivery() error = nil")
	}
	if !delivery.nacked {
		t.Fatal("nacked = false")
	}
	if delivery.nackRequeue {
		t.Fatal("nackRequeue = true")
	}
}
