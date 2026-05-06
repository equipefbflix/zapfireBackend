package runner

import (
	"context"
	"fmt"
)

type RunningJobCounter interface {
	CountRunningByPair(ctx context.Context, phoneAID string, phoneBID string) (int, error)
	CountRunningByEvolutionServer(ctx context.Context, evolutionServerID string) (int, error)
}

type MaxConcurrencyGate struct {
	counter            RunningJobCounter
	maxPerPair         int
	maxPerServer       int
}

func NewMaxConcurrencyGate(counter RunningJobCounter, maxPerPair int, maxPerServer int) MaxConcurrencyGate {
	return MaxConcurrencyGate{
		counter:      counter,
		maxPerPair:   maxPerPair,
		maxPerServer: maxPerServer,
	}
}

func (g MaxConcurrencyGate) Check(ctx context.Context, serverID string, phoneAID string, phoneBID string) error {
	if g.counter == nil {
		return nil
	}
	if g.maxPerPair > 0 {
		count, err := g.counter.CountRunningByPair(ctx, phoneAID, phoneBID)
		if err != nil {
			return err
		}
		if count >= g.maxPerPair {
			return fmt.Errorf("pair concurrency exceeded")
		}
	}
	if g.maxPerServer > 0 && serverID != "" {
		count, err := g.counter.CountRunningByEvolutionServer(ctx, serverID)
		if err != nil {
			return err
		}
		if count >= g.maxPerServer {
			return fmt.Errorf("evolution server concurrency exceeded")
		}
	}
	return nil
}
