# Idempotencia por step

Esta feature evita envio duplicado quando uma mensagem RabbitMQ e reprocessada ou quando o worker reinicia no meio de um job.

## Regra

Antes de executar um `conversation_step`, consultar `execution_logs` do `warmingJobId` procurando log com:

- `status = success`;
- `request_payload.stepId = <step.id>`.

Se existir, o runner pula o step.

## Resultado esperado

- `ExecutionLogRepository.ExistsSuccessfulStep`.
- Runner usando a consulta antes do envio.
- Testes unitarios cobrindo skip de step ja executado.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `ExecutionLogRepository.ExistsSuccessfulStep` e skip no runner.
2. `go test ./internal/repository ./internal/runner` falhou por ausencia do metodo e porque o runner ainda executava o step.
3. Consulta de idempotencia implementada e runner ajustado.
4. O teste revelou um bug de contagem: steps pulados ainda eram retornados como executados.
5. Contador corrigido para retornar somente execucoes reais.
6. `go test ./...` passou.
