# Rotas de Evolution servers

Esta feature implementa endpoints HTTP para cadastrar e listar servidores Evolution API disponiveis para redundancia.

## Endpoints

```http
POST /api/v1/evolution-servers
GET /api/v1/evolution-servers
```

## POST /api/v1/evolution-servers

Request:

```json
{
  "name": "evo-01",
  "baseUrl": "https://evo.example.com",
  "apiKeySecretName": "EVOLUTION_EVO_01_API_KEY",
  "enabled": true,
  "weight": 1,
  "maxConcurrentJobs": 5,
  "testRunId": "optional-test-run-id",
  "metadata": {
    "region": "br"
  }
}
```

Response `201`:

```json
{
  "id": "uuid",
  "name": "evo-01",
  "baseUrl": "https://evo.example.com",
  "apiKeySecretName": "EVOLUTION_EVO_01_API_KEY",
  "enabled": true,
  "weight": 1,
  "maxConcurrentJobs": 5,
  "healthStatus": "unknown",
  "metadata": {
    "region": "br",
    "testRunId": "optional-test-run-id"
  }
}
```

## GET /api/v1/evolution-servers

Response `200`:

```json
{
  "items": []
}
```

## Resultado esperado

- Testes HTTP com fake store.
- Repository com `List`.
- Wiring no `cmd/api`.
- Validacao real via MCP no projeto Supabase `rxdophybnwoocsdyxyjm`.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `POST /api/v1/evolution-servers`, `GET /api/v1/evolution-servers` e `EvolutionServerRepository.List`.
2. `go test ./internal/httpserver ./internal/repository` falhou por ausencia dos contratos.
3. Handlers, interface `EvolutionServerStore`, `EvolutionServerRepository.List` e wiring no `cmd/api` implementados.
4. `go test ./...` passou.
5. Insercao real em `public.evolution_servers` com `metadata.testRunId = codex-evolution-server-route-validation`.
6. Exclusao real pelo mesmo `testRunId`.
7. Confirmacao final: `remaining = 0`.
