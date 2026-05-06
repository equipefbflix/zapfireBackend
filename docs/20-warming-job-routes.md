# Rotas de warming jobs

Esta feature implementa endpoints HTTP para criar e listar jobs de aquecimento. Um job agenda a execucao de um script entre dois numeros.

## Endpoints

```http
POST /api/v1/warming-jobs
GET /api/v1/warming-jobs
```

## POST /api/v1/warming-jobs

Request:

```json
{
  "phoneAId": "uuid",
  "phoneBId": "uuid",
  "scriptId": "uuid",
  "scheduledAt": "2026-05-04T15:00:00Z",
  "testRunId": "optional-test-run-id",
  "metadata": {
    "source": "manual"
  }
}
```

Response `201`:

```json
{
  "id": "uuid",
  "scriptId": "uuid",
  "phoneAId": "uuid",
  "phoneBId": "uuid",
  "status": "pending",
  "scheduledAt": "2026-05-04T15:00:00Z",
  "currentStepOrder": 0,
  "error": "",
  "metadata": {
    "source": "manual",
    "testRunId": "optional-test-run-id"
  }
}
```

## GET /api/v1/warming-jobs

Response `200`:

```json
{
  "items": []
}
```

## Resultado esperado

- Repository `Create`, `List` e cleanup por `testRunId`.
- Testes HTTP com fake store.
- Wiring no `cmd/api`.
- Validacao real via MCP no projeto Supabase `rxdophybnwoocsdyxyjm`.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `WarmingJobRepository.Create`, `WarmingJobRepository.List`, cleanup por `testRunId`, `POST /api/v1/warming-jobs` e `GET /api/v1/warming-jobs`.
2. `go test ./internal/httpserver ./internal/repository` falhou por ausencia dos contratos.
3. Repository, handlers, interface `WarmingJobStore` e wiring no `cmd/api` implementados.
4. `go test ./...` passou.
5. Insercao real de dois registros em `public.phone_numbers` e um registro em `public.warming_jobs` com `metadata.testRunId = codex-warming-job-route-validation`.
6. Exclusao real do job e dos telefones pelo mesmo `testRunId`.
7. Confirmacao final: `remaining_jobs = 0` e `remaining_phones = 0`.
