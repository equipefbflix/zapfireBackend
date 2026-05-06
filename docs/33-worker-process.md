# Processo cmd/worker

Esta feature adiciona o binario `cmd/worker` para consumo real da fila de warming jobs.

## Fluxo

1. Carregar `DATABASE_URL`, `RABBITMQ_URL` e configs.
2. Conectar no Postgres.
3. Conectar no RabbitMQ.
4. Declarar topologia.
5. Aplicar `prefetch`.
6. Consumir `RABBITMQ_QUEUE_WARMING_JOBS`.
7. Para cada delivery:
   - passar `delivery.Body` para `queue.WarmingJobDueConsumer`;
   - `ack` em sucesso;
   - `nack(requeue=true)` em erro.

## Resultado esperado

- `internal/workerapp` com loop de consumo testavel.
- `cmd/worker/main.go`.
- Testes unitarios para `ack` e `nack`.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `ack` em sucesso e `nack(requeue=true)` em falha.
2. `go test ./internal/workerapp` falhou por ausencia de `ProcessDelivery`.
3. Loop de consumo testavel implementado em `internal/workerapp`.
4. Em seguida foi implementado:
   - `queue.RabbitMQBroker.SetPrefetch`;
   - `queue.RabbitMQBroker.Consume`;
   - `cmd/worker/main.go`;
   - `EvolutionServerRepository.GetByID` para suportar executor por instancia.
5. `go test ./...` passou.
6. A validacao AMQP real continua pendente por conectividade externa.
