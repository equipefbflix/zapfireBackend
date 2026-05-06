package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/instance"
	"aquecedor-evolution/backend/internal/queue"
	"aquecedor-evolution/backend/internal/repository"
	"aquecedor-evolution/backend/internal/runner"
	"aquecedor-evolution/backend/internal/worker"
	"aquecedor-evolution/backend/internal/workerapp"
)

func main() {
	if err := run(); err != nil {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	databaseConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		return err
	}
	rabbitConfig, err := config.LoadRabbitMQConfig()
	if err != nil {
		return err
	}
	plannerConfig := config.LoadPlannerConfig()

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
	if err := broker.SetPrefetch(rabbitConfig.Prefetch); err != nil {
		return err
	}

	executor := repository.NewPgxExecutor(pool)
	jobRepo := repository.NewWarmingJobRepository(executor)
	stepRepo := repository.NewConversationStepRepository(executor)
	instanceRepo := repository.NewInstanceRepository(executor)
	executionLogRepo := repository.NewExecutionLogRepository(executor)
	evolutionServerRepo := repository.NewEvolutionServerRepository(executor)

	instanceExecutors := runner.NewInstanceExecutorFactory(
		evolutionServerRepo,
		instance.EnvSecretResolver{},
		runner.DefaultStepClientFactory{},
	)
	concurrencyGate := runner.NewMaxConcurrencyGate(
		jobRepo,
		plannerConfig.MaxRunningJobsPerPair,
		plannerConfig.MaxRunningJobsPerEvolutionServer,
	)
	jobRunner := runner.NewWarmingJobRunner(jobRepo, stepRepo, instanceRepo, instanceExecutors, executionLogRepo, concurrencyGate)
	jobWorker := worker.NewWarmingJobWorker(jobRunner)
	consumer := queue.NewWarmingJobDueConsumer(jobWorker)

	deliveries, err := broker.Consume(rabbitConfig.WarmingJobsQueue)
	if err != nil {
		return err
	}

	slog.Info("worker consuming", "queue", rabbitConfig.WarmingJobsQueue, "prefetch", rabbitConfig.Prefetch)
	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return nil
			}
			if err := workerapp.ProcessDelivery(ctx, consumer, rabbitDelivery{delivery: delivery}, rabbitConfig.MaxRetries); err != nil {
				slog.Error("failed to process delivery", "error", err)
			}
		}
	}
}

type rabbitDelivery struct {
	delivery amqp.Delivery
}

func (d rabbitDelivery) Body() []byte {
	return d.delivery.Body
}

func (d rabbitDelivery) Ack() error {
	return d.delivery.Ack(false)
}

func (d rabbitDelivery) Nack(requeue bool) error {
	return d.delivery.Nack(false, requeue)
}

func (d rabbitDelivery) Attempt() int {
	attempt := 1
	if d.delivery.Headers == nil {
		return attempt
	}
	if deathRaw, ok := d.delivery.Headers["x-death"]; ok {
		if entries, ok := deathRaw.([]any); ok && len(entries) > 0 {
			if first, ok := entries[0].(amqp.Table); ok {
				if countRaw, ok := first["count"]; ok {
					switch count := countRaw.(type) {
					case int64:
						return int(count) + 1
					case int32:
						return int(count) + 1
					case int:
						return count + 1
					}
				}
			}
		}
	}
	return attempt
}
