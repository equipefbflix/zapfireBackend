package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/queue"
	"aquecedor-evolution/backend/internal/repository"
	schedulerpkg "aquecedor-evolution/backend/internal/scheduler"
)

func main() {
	if err := run(); err != nil {
		slog.Error("scheduler stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	schedulerConfig := config.LoadSchedulerConfig()
	if !schedulerConfig.Enabled {
		slog.Info("scheduler disabled")
		return nil
	}

	databaseConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		return err
	}
	rabbitConfig, err := config.LoadRabbitMQConfig()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, databaseConfig)
	if err != nil {
		return err
	}
	defer pool.Close()

	broker, err := queue.DialRabbitMQ(rabbitConfig.URL)
	if err != nil {
		return err
	}
	defer broker.Close()

	topology := queue.DefaultTopology(queue.TopologyConfig{
		Exchange:             rabbitConfig.Exchange,
		WarmingJobsQueue:     rabbitConfig.WarmingJobsQueue,
		EvolutionEventsQueue: rabbitConfig.EvolutionEventsQueue,
		DeadLetterQueue:      rabbitConfig.DeadLetterQueue,
	})
	if err := queue.DeclareTopology(ctx, broker, topology); err != nil {
		return err
	}

	executor := repository.NewPgxExecutor(pool)
	jobRepo := repository.NewWarmingJobRepository(executor)
	phoneRepo := repository.NewPhoneNumberRepository(executor)
	publisher := queue.NewPublisher(broker, topology.Exchange.Name)
	scheduler := schedulerpkg.NewWarmingJobScheduler(jobRepo, publisher, 100)

	ticker := time.NewTicker(schedulerConfig.TickInterval)
	defer ticker.Stop()

	lastResetDate := time.Now().UTC().Format("2006-01-02")

	slog.Info("scheduler publishing due jobs", "tickInterval", schedulerConfig.TickInterval)
	if err := publishOnce(ctx, scheduler, rabbitConfig.PublishTimeout); err != nil {
		slog.Error("scheduler publish failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			currentDate := time.Now().UTC().Format("2006-01-02")
			if currentDate != lastResetDate {
				runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				resetCount, err := phoneRepo.ResetDailyMessageCounts(runCtx)
				cancel()
				if err != nil {
					slog.Error("daily message count reset failed", "error", err)
				} else if resetCount > 0 {
					slog.Info("daily message counts reset", "phonesReset", resetCount)
				}
				lastResetDate = currentDate
			}
			if err := publishOnce(ctx, scheduler, rabbitConfig.PublishTimeout); err != nil {
				slog.Error("scheduler publish failed", "error", err)
			}
		}
	}
}

func publishOnce(ctx context.Context, scheduler schedulerpkg.WarmingJobScheduler, timeout time.Duration) error {
	runCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	published, err := scheduler.PublishDue(runCtx, time.Now().UTC())
	if err != nil {
		return err
	}
	if published > 0 {
		slog.Info("scheduler published due jobs", "count", published)
	}
	return nil
}
