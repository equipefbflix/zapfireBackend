# Rotas de message templates

Esta feature implementa endpoints HTTP para cadastrar e listar templates de mensagens usados nas conversas de aquecimento.

## Endpoints

```http
POST /api/v1/message-templates
GET /api/v1/message-templates
```

## POST /api/v1/message-templates

Request:

```json
{
  "category": "casual",
  "title": "bom dia simples",
  "body": "Bom dia, tudo certo por ai?",
  "weight": 10,
  "enabled": true,
  "minWarmingScore": 0,
  "maxWarmingScore": 40,
  "testRunId": "optional-test-run-id",
  "metadata": {
    "tone": "friendly"
  }
}
```

Response `201`:

```json
{
  "id": "uuid",
  "category": "casual",
  "title": "bom dia simples",
  "body": "Bom dia, tudo certo por ai?",
  "weight": 10,
  "enabled": true,
  "minWarmingScore": 0,
  "maxWarmingScore": 40,
  "metadata": {
    "tone": "friendly",
    "testRunId": "optional-test-run-id"
  }
}
```

## GET /api/v1/message-templates

Response `200`:

```json
{
  "items": []
}
```

## Resultado esperado

- Testes HTTP com fake store.
- Repository `Create`, `List` e cleanup por `testRunId`.
- Wiring no `cmd/api`.
- Validacao real via MCP no projeto Supabase `rxdophybnwoocsdyxyjm`.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `POST /api/v1/message-templates`, `GET /api/v1/message-templates`, `MessageTemplateRepository.Create`, `MessageTemplateRepository.List` e cleanup por `testRunId`.
2. `go test ./internal/httpserver ./internal/repository` falhou por ausencia dos contratos.
3. Repository, handlers, interface `MessageTemplateStore` e wiring no `cmd/api` implementados.
4. `go test ./...` passou.
5. Insercao real em `public.message_templates` com `metadata.testRunId = codex-message-template-route-validation`.
6. Exclusao real pelo mesmo `testRunId`.
7. Confirmacao final: `remaining = 0`.
