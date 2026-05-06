# Worker ligado ao runner

Esta feature substitui o worker inicial de warming jobs por uma ponte para o `runner.WarmingJobRunner`.

## Fluxo

1. O consumer RabbitMQ decodifica `warming.job.due`.
2. O worker recebe `jobId`.
3. O worker chama o runner real.

## Resultado esperado

- `worker.WarmingJobWorker` deixa de depender de repository/logs diretos.
- `worker.WarmingJobWorker` depende de uma interface `Run(ctx, jobID)`.
- Testes unitarios validam que o `jobId` da mensagem chega ao runner.

## Validacao realizada

Fluxo executado:

1. Teste criado antes da implementacao para garantir que `jobId` da mensagem chega ao runner.
2. `go test ./internal/worker` falhou porque o construtor ainda exigia stores antigos.
3. Worker simplificado para delegar ao runner.
4. `go test ./...` passou.
