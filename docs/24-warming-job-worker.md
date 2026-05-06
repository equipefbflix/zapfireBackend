# Worker de warming jobs

Esta feature implementa a primeira base do consumidor de jobs de aquecimento.

## Escopo inicial

O worker recebe uma mensagem `warming.job.due`, busca o job no Supabase e registra um `execution_log` de inicio de processamento.

Execucao de steps e chamadas Evolution entram no bloco seguinte.

## Entrada

```json
{
  "type": "warming.job.due",
  "version": 1,
  "jobId": "uuid",
  "testRunId": "optional-test-run-id",
  "publishedAt": "2026-05-04T15:00:00Z"
}
```

## Resultado esperado

- `WarmingJobRepository.GetByID`.
- `worker.WarmingJobWorker.HandleDue`.
- Registro em `execution_logs` com `status = running`.
- Testes unitarios com fakes.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `WarmingJobRepository.GetByID` e `worker.WarmingJobWorker.HandleDue`.
2. `go test ./internal/repository ./internal/worker` falhou por ausencia dos contratos.
3. `GetByID` e worker inicial implementados.
4. `go test ./...` passou.
5. Sem validacao MCP propria neste bloco; ele usa tabelas ja validadas em `warming_jobs` e `execution_logs`.
