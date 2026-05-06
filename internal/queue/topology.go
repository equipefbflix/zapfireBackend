package queue

const (
	RoutingKeyWarmingJobDue          = "warming.job.due"
	RoutingKeyEvolutionEventReceived = "evolution.event.received"
	RoutingKeyDeadLetter             = "dead_letter"
)

type TopologyConfig struct {
	Exchange             string
	WarmingJobsQueue     string
	EvolutionEventsQueue string
	DeadLetterQueue      string
}

type Topology struct {
	Exchange ExchangeSpec
	Queues   []QueueSpec
	Bindings []BindingSpec
}

type ExchangeSpec struct {
	Name    string
	Kind    string
	Durable bool
}

type QueueSpec struct {
	Name                 string
	Durable              bool
	DeadLetter           bool
	DeadLetterExchange   string
	DeadLetterRoutingKey string
}

type BindingSpec struct {
	Queue      string
	Exchange   string
	RoutingKey string
}

func DefaultTopology(cfg TopologyConfig) Topology {
	exchange := ExchangeSpec{
		Name:    cfg.Exchange,
		Kind:    "direct",
		Durable: true,
	}

	return Topology{
		Exchange: exchange,
		Queues: []QueueSpec{
			{Name: cfg.WarmingJobsQueue, Durable: true, DeadLetterExchange: exchange.Name, DeadLetterRoutingKey: RoutingKeyDeadLetter},
			{Name: cfg.EvolutionEventsQueue, Durable: true, DeadLetterExchange: exchange.Name, DeadLetterRoutingKey: RoutingKeyDeadLetter},
			{Name: cfg.DeadLetterQueue, Durable: true, DeadLetter: true},
		},
		Bindings: []BindingSpec{
			{Queue: cfg.WarmingJobsQueue, Exchange: exchange.Name, RoutingKey: RoutingKeyWarmingJobDue},
			{Queue: cfg.EvolutionEventsQueue, Exchange: exchange.Name, RoutingKey: RoutingKeyEvolutionEventReceived},
			{Queue: cfg.DeadLetterQueue, Exchange: exchange.Name, RoutingKey: RoutingKeyDeadLetter},
		},
	}
}
