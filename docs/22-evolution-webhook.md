# Webhook Evolution

Esta feature implementa a entrada de eventos da Evolution API e persiste o payload bruto em `public.evolution_events`.

## Endpoint

```http
POST /api/v1/webhooks/evolution
```

Header obrigatorio quando `WEBHOOK_EVOLUTION_SECRET` estiver configurado:

```http
X-Webhook-Secret: shared-secret
```

Request exemplo:

```json
{
  "event": "messages.upsert",
  "instance": "chip-sp-01",
  "data": {
    "key": {
      "id": "message-id"
    }
  }
}
```

Response `202`:

```json
{
  "id": "uuid",
  "eventType": "messages.upsert",
  "instanceName": "chip-sp-01"
}
```

## Regras

- Validar `X-Webhook-Secret` quando houver segredo configurado.
- Aceitar `event` ou `eventType` no payload.
- Aceitar `instance` ou `instanceName` no payload.
- Persistir payload bruto completo em `public.evolution_events.payload`.
- Responder rapido com `202 Accepted`.

## Resultado esperado

- Config `WEBHOOK_EVOLUTION_SECRET`.
- Repository `EvolutionEventRepository.Create`.
- Handler `POST /api/v1/webhooks/evolution`.
- Validacao real via MCP no projeto Supabase `rxdophybnwoocsdyxyjm`.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para config `WEBHOOK_EVOLUTION_SECRET`, `EvolutionEventRepository.Create` e `POST /api/v1/webhooks/evolution`.
2. `go test ./internal/config ./internal/httpserver ./internal/repository` falhou por ausencia dos contratos.
3. Config, repository, handler, interface `EvolutionEventStore` e wiring no `cmd/api` implementados.
4. `go test ./...` passou.
5. Insercao real em `public.evolution_events` com payload bruto.
6. Exclusao real pelo `id` retornado.
7. Confirmacao final: `remaining_events = 0`.
