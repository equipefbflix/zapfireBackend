# Retry e dead-letter no worker

Esta feature adiciona a politica inicial de reprocessamento no consumo do worker.

## Politica

- `attempt <= RABBITMQ_MAX_RETRIES`: `nack(requeue=true)`
- `attempt > RABBITMQ_MAX_RETRIES`: `nack(requeue=false)` para seguir ao dead-letter

## Escopo desta etapa

- `RabbitMQConfig.MaxRetries`
- `workerapp.ProcessDelivery(..., maxRetries)`
- `Delivery.Attempt()`
- `rabbitDelivery.Attempt()` lendo `x-death.count`

## TDD aplicado

1. testes unitarios atualizados em `internal/workerapp/loop_test.go`
2. testes de config atualizados em `internal/config/rabbitmq_test.go`
3. os testes falharam por ausencia de:
   - `MaxRetries`
   - `Attempt()`
   - assinatura nova de `ProcessDelivery`
4. implementacao concluida
5. `go test ./...` passou

## Validacao local real

Foi criado teste integration:

- `internal/workerapp/dead_letter_integration_test.go`

Esse teste depende de RabbitMQ local ou remoto acessivel.

Na sessao atual, a tentativa de subir RabbitMQ local via Docker ficou bloqueada porque o Docker daemon nao estava disponivel.

## Env novo

```env
RABBITMQ_MAX_RETRIES=3
```
