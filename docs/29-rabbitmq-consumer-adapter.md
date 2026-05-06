# Adapter consumidor RabbitMQ

Esta feature implementa o adapter que recebe o corpo JSON de uma mensagem RabbitMQ `warming.job.due`, valida o envelope e chama o handler de job.

## Escopo

- Decodificar JSON.
- Validar `type = warming.job.due`.
- Validar `version = 1`.
- Validar `jobId` obrigatorio.
- Chamar handler.

Ack/nack e consumo real do canal RabbitMQ ficam para o adapter de infraestrutura, depois que conectividade AMQP real estiver liberada.

## Resultado esperado

- `queue.WarmingJobDueConsumer`.
- Testes unitarios para mensagem valida e invalida.
- `go test ./...` passando.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para mensagem valida, tipo invalido e `jobId` ausente.
2. `go test ./internal/queue` falhou por ausencia de `NewWarmingJobDueConsumer`.
3. Consumer local implementado.
4. `go test ./...` passou.
