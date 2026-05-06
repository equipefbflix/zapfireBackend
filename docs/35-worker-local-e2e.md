# E2E local do worker

Esta feature adiciona um teste integrado de banco para o caminho:

`warming.job.due JSON -> consumer -> worker -> runner -> execution_logs`

## Escopo

- Usa banco real via `DATABASE_URL`.
- Nao usa RabbitMQ real.
- Nao usa Evolution real.
- Usa `StepExecutor` com client fake para validar persistencia e encadeamento.

## Execucao

```bash
ENABLE_REAL_TESTS=true \
TEST_RUN_ID=worker-local-e2e-001 \
DATABASE_URL='postgres://...' \
go test -tags=integration ./internal/workerapp -run TestWorkerLocalFlowRealDatabase -v
```

## Resultado esperado

- Cria dados reais minimos no Supabase.
- Processa um job real a partir de um body JSON.
- Gera `execution_logs` de sucesso.
- Limpa todos os dados de teste.

## Validacao realizada

Fluxo executado:

1. Teste integration criado em `internal/workerapp/workerapp_integration_test.go`.
2. O caminho compila com `go test -tags=integration ./internal/workerapp ./...`.
3. Em 2026-05-04, a execucao real foi rodada com `ENABLE_REAL_TESTS=true` e `DATABASE_URL` valido.
4. Resultado real: `PASS` em `TestWorkerLocalFlowRealDatabase`.
5. O teste criou dados reais no projeto `rxdophybnwoocsdyxyjm`, gerou `execution_logs` e executou cleanup no final.
6. `go test ./...` continua passando fora da tag `integration`.
