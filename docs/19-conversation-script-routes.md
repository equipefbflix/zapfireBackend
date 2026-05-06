# Rotas de conversation scripts

Esta feature implementa endpoints HTTP para cadastrar e listar scripts de conversa com steps. Um script define a sequencia de acoes entre dois numeros no aquecimento.

## Endpoints

```http
POST /api/v1/conversation-scripts
GET /api/v1/conversation-scripts
```

## POST /api/v1/conversation-scripts

Request:

```json
{
  "name": "conversa_basica_manha",
  "category": "casual",
  "enabled": true,
  "weight": 10,
  "minWarmingScore": 0,
  "maxWarmingScore": 40,
  "steps": [
    {
      "stepOrder": 1,
      "senderRole": "a",
      "actionType": "send_presence",
      "payload": { "presence": "composing" },
      "minDelaySeconds": 1,
      "maxDelaySeconds": 3
    },
    {
      "stepOrder": 2,
      "senderRole": "a",
      "actionType": "send_text",
      "templateId": "uuid",
      "minDelaySeconds": 5,
      "maxDelaySeconds": 30
    }
  ]
}
```

Response `201`:

```json
{
  "id": "uuid",
  "name": "conversa_basica_manha",
  "category": "casual",
  "enabled": true,
  "weight": 10,
  "minWarmingScore": 0,
  "maxWarmingScore": 40,
  "steps": []
}
```

## GET /api/v1/conversation-scripts

Response `200`:

```json
{
  "items": []
}
```

## Resultado esperado

- Repositories para `conversation_scripts` e `conversation_steps`.
- Service para criar scripts com steps.
- Testes HTTP com fake service.
- Wiring no `cmd/api`.
- Validacao real via MCP no projeto Supabase `rxdophybnwoocsdyxyjm`.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `ConversationScriptRepository`, `ConversationStepRepository`, `POST /api/v1/conversation-scripts` e `GET /api/v1/conversation-scripts`.
2. `go test ./internal/httpserver ./internal/repository` falhou por ausencia dos contracts e do pacote `internal/conversation`.
3. Repositories, service `conversation.Service`, handlers e wiring no `cmd/api` implementados.
4. `go test ./...` passou.
5. Insercao real em `public.conversation_scripts` com 1 registro em `public.conversation_steps`.
6. Exclusao real pelo `conversation_scripts.name = codex-conversation-script-route-validation`.
7. Confirmacao final: `remaining_scripts = 0` e `remaining_steps = 0`.
