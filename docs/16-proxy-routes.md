# Rotas de proxies

Esta feature implementa endpoints HTTP para cadastrar e listar proxies.

## Endpoints

```http
POST /api/v1/proxies
GET /api/v1/proxies
```

## POST /api/v1/proxies

Request:

```json
{
  "name": "proxy-01",
  "host": "proxy.example.com",
  "port": 8000,
  "protocol": "http",
  "username": "user",
  "passwordSecretName": "PROXY_01_PASSWORD",
  "enabled": true,
  "maxInstances": 20,
  "testRunId": "optional-test-run-id",
  "metadata": {
    "provider": "manual"
  }
}
```

Response `201`:

```json
{
  "id": "uuid",
  "name": "proxy-01",
  "host": "proxy.example.com",
  "port": 8000,
  "protocol": "http",
  "username": "user",
  "passwordSecretName": "PROXY_01_PASSWORD",
  "enabled": true,
  "maxInstances": 20,
  "currentInstances": 0,
  "metadata": {
    "provider": "manual",
    "testRunId": "optional-test-run-id"
  }
}
```

## GET /api/v1/proxies

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

1. Testes criados antes da implementacao para `POST /api/v1/proxies`, `GET /api/v1/proxies` e `ProxyRepository.List`.
2. `go test ./internal/httpserver ./internal/repository` falhou por ausencia dos contratos.
3. Handlers, interface `ProxyStore`, `ProxyRepository.List` e wiring no `cmd/api` implementados.
4. `go test ./...` passou.
5. Insercao real em `public.proxies` com `metadata.testRunId = codex-proxy-route-validation`.
6. Exclusao real pelo mesmo `testRunId`.
7. Confirmacao final: `remaining = 0`.
