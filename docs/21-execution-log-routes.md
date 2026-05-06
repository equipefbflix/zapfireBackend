# Rotas de execution logs

Esta feature implementa endpoints HTTP para registrar e listar logs de execucao. Cada chamada externa ou step executado deve gerar um log auditavel.

## Endpoints

```http
POST /api/v1/execution-logs
GET /api/v1/execution-logs
```

## POST /api/v1/execution-logs

Request:

```json
{
  "warmingJobId": "uuid",
  "instanceId": "uuid",
  "actionType": "send_text",
  "status": "success",
  "requestPayload": { "text": "Bom dia" },
  "responsePayload": { "messageId": "abc" },
  "evolutionMessageKey": { "id": "abc" },
  "remoteJid": "5511999999999@s.whatsapp.net",
  "error": "",
  "durationMs": 120
}
```

Response `201`:

```json
{
  "id": "uuid",
  "warmingJobId": "uuid",
  "instanceId": "uuid",
  "actionType": "send_text",
  "status": "success",
  "requestPayload": {},
  "responsePayload": {},
  "evolutionMessageKey": {},
  "remoteJid": "5511999999999@s.whatsapp.net",
  "error": "",
  "durationMs": 120,
  "createdAt": "2026-05-04T15:00:00Z"
}
```

## GET /api/v1/execution-logs

Response `200`:

```json
{
  "items": []
}
```

## Resultado esperado

- Repository `Create` e `List`.
- Testes HTTP com fake store.
- Wiring no `cmd/api`.
- Validacao real via MCP no projeto Supabase `rxdophybnwoocsdyxyjm`.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `ExecutionLogRepository.Create`, `ExecutionLogRepository.List`, `POST /api/v1/execution-logs` e `GET /api/v1/execution-logs`.
2. `go test ./internal/httpserver ./internal/repository` falhou por ausencia dos contratos.
3. Repository, handlers, interface `ExecutionLogStore` e wiring no `cmd/api` implementados.
4. `go test ./...` passou.
5. Insercao real em `public.execution_logs` com payloads JSON e `duration_ms`.
6. Exclusao real pelo `id` retornado.
7. Confirmacao final: `remaining_logs = 0`.
