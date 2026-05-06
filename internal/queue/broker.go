package queue

import (
	"context"
	"encoding/json"
	"fmt"
)

type TopologyDeclarer interface {
	DeclareExchange(ctx context.Context, exchange ExchangeSpec) error
	DeclareQueue(ctx context.Context, queue QueueSpec) error
	BindQueue(ctx context.Context, binding BindingSpec) error
}

type PublishContent struct {
	ContentType string
	Body        []byte
	Persistent  bool
}

type MessagePublisher interface {
	Publish(ctx context.Context, exchange, routingKey string, content PublishContent) error
}

type Publisher struct {
	broker   MessagePublisher
	exchange string
}

func DeclareTopology(ctx context.Context, declarer TopologyDeclarer, topology Topology) error {
	if err := declarer.DeclareExchange(ctx, topology.Exchange); err != nil {
		return fmt.Errorf("declare exchange %q: %w", topology.Exchange.Name, err)
	}

	for _, queue := range topology.Queues {
		if err := declarer.DeclareQueue(ctx, queue); err != nil {
			return fmt.Errorf("declare queue %q: %w", queue.Name, err)
		}
	}

	for _, binding := range topology.Bindings {
		if err := declarer.BindQueue(ctx, binding); err != nil {
			return fmt.Errorf("bind queue %q: %w", binding.Queue, err)
		}
	}

	return nil
}

func NewPublisher(broker MessagePublisher, exchange string) Publisher {
	return Publisher{
		broker:   broker,
		exchange: exchange,
	}
}

func (p Publisher) PublishWarmingJobDue(ctx context.Context, msg WarmingJobDueMessage) error {
	return p.publishJSON(ctx, RoutingKeyWarmingJobDue, msg)
}

func (p Publisher) PublishEvolutionEventReceived(ctx context.Context, msg EvolutionEventReceivedMessage) error {
	return p.publishJSON(ctx, RoutingKeyEvolutionEventReceived, msg)
}

func (p Publisher) publishJSON(ctx context.Context, routingKey string, msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal queue message: %w", err)
	}

	return p.broker.Publish(ctx, p.exchange, routingKey, PublishContent{
		ContentType: "application/json",
		Body:        body,
		Persistent:  true,
	})
}
