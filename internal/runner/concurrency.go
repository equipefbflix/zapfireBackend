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
	counter      RunningJobCounter
	maxPerPair   int
	maxPerServer int
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

type DailyMessageCounter interface {
	GetDailyMessageCount(ctx context.Context, phoneNumberID string) (int, error)
}

type dailyLimitGateImpl struct {
	counter      DailyMessageCounter
	maxPerNumber int
	maxPerPair   int
}

func NewDailyLimitGate(counter DailyMessageCounter, maxPerNumber int, maxPerPair int) DailyLimitGate {
	return dailyLimitGateImpl{
		counter:      counter,
		maxPerNumber: maxPerNumber,
		maxPerPair:   maxPerPair,
	}
}

func (g dailyLimitGateImpl) Check(ctx context.Context, phoneAID string, phoneBID string) error {
	if g.counter == nil {
		return nil
	}
	if g.maxPerNumber > 0 {
		countA, err := g.counter.GetDailyMessageCount(ctx, phoneAID)
		if err != nil {
			return fmt.Errorf("check daily limit for phone A: %w", err)
		}
		if countA >= g.maxPerNumber {
			return fmt.Errorf("phone A daily limit exceeded (%d/%d)", countA, g.maxPerNumber)
		}

		countB, err := g.counter.GetDailyMessageCount(ctx, phoneBID)
		if err != nil {
			return fmt.Errorf("check daily limit for phone B: %w", err)
		}
		if countB >= g.maxPerNumber {
			return fmt.Errorf("phone B daily limit exceeded (%d/%d)", countB, g.maxPerNumber)
		}
	}
	if g.maxPerPair > 0 {
		countA, err := g.counter.GetDailyMessageCount(ctx, phoneAID)
		if err != nil {
			return fmt.Errorf("check pair daily limit for phone A: %w", err)
		}
		countB, err := g.counter.GetDailyMessageCount(ctx, phoneBID)
		if err != nil {
			return fmt.Errorf("check pair daily limit for phone B: %w", err)
		}
		pairTotal := countA + countB
		if pairTotal >= g.maxPerPair {
			return fmt.Errorf("pair daily limit exceeded (%d/%d)", pairTotal, g.maxPerPair)
		}
	}
	return nil
}
