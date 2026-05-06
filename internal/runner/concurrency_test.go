package runner

import (
	"context"
	"errors"
	"testing"
)

type fakeRunningJobCounter struct {
	pairCount   int
	serverCount int
	pairErr     error
	serverErr   error
}

func (c fakeRunningJobCounter) CountRunningByPair(ctx context.Context, phoneAID string, phoneBID string) (int, error) {
	if c.pairErr != nil {
		return 0, c.pairErr
	}
	return c.pairCount, nil
}

func (c fakeRunningJobCounter) CountRunningByEvolutionServer(ctx context.Context, evolutionServerID string) (int, error) {
	if c.serverErr != nil {
		return 0, c.serverErr
	}
	return c.serverCount, nil
}

func TestMaxConcurrencyGateAllowsBelowLimits(t *testing.T) {
	gate := NewMaxConcurrencyGate(fakeRunningJobCounter{pairCount: 0, serverCount: 2}, 1, 3)

	if err := gate.Check(context.Background(), "server-id", "phone-a", "phone-b"); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestMaxConcurrencyGateBlocksPairLimit(t *testing.T) {
	gate := NewMaxConcurrencyGate(fakeRunningJobCounter{pairCount: 1}, 1, 0)

	err := gate.Check(context.Background(), "server-id", "phone-a", "phone-b")
	if err == nil || err.Error() != "pair concurrency exceeded" {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestMaxConcurrencyGateBlocksServerLimit(t *testing.T) {
	gate := NewMaxConcurrencyGate(fakeRunningJobCounter{pairCount: 0, serverCount: 5}, 1, 5)

	err := gate.Check(context.Background(), "server-id", "phone-a", "phone-b")
	if err == nil || err.Error() != "evolution server concurrency exceeded" {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestMaxConcurrencyGateReturnsCounterErrors(t *testing.T) {
	want := errors.New("count failed")
	gate := NewMaxConcurrencyGate(fakeRunningJobCounter{pairErr: want}, 1, 1)

	err := gate.Check(context.Background(), "server-id", "phone-a", "phone-b")
	if !errors.Is(err, want) {
		t.Fatalf("Check() error = %v", err)
	}
}
