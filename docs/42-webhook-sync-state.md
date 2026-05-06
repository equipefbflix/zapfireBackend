# Webhook + sincronizacao de estado

Esta feature adiciona sincronizacao imediata apos persistir o webhook da Evolution.

## Escopo desta etapa

- persistir `evolution_events`
- sincronizar `instances` por `CONNECTION_UPDATE`
- sincronizar `execution_logs` por `MESSAGES_UPDATE`, `MESSAGES_UPSERT` e `SEND_MESSAGE`
- marcar `evolution_events.processed_at`

## TDD aplicado

1. Criados testes do service em `internal/evolutionsync/service_test.go`:
   - `TestServiceSyncConnectionUpdate`
   - `TestServiceSyncMessageUpdate`
   - `TestServiceSyncMessageFailure`
2. Criado teste do handler em `internal/httpserver/evolution_webhook_sync_test.go`.
3. Os testes falharam por ausencia de:
   - `evolutionsync.NewService`
   - `ServerConfig.EvolutionSync`
4. Implementado `internal/evolutionsync/service.go`.
5. Implementados updates de repository:
   - `InstanceRepository.UpdateConnectionStateByName`
   - `ExecutionLogRepository.UpdateStatusByMessageID`
   - `EvolutionEventRepository.MarkProcessed`
6. Ligado o sync ao `POST /api/v1/webhooks/evolution`.
7. Ligado o wiring real no `cmd/api/main.go`.

## Mapeamento inicial

### `CONNECTION_UPDATE`

Atualiza em `public.instances`:

- `status`
- `last_error`
- `last_connection_check_at`
- `last_connected_at` quando `open`
- `last_disconnected_at` quando `close`

### `MESSAGES_UPDATE`, `MESSAGES_UPSERT`, `SEND_MESSAGE`

Atualiza em `public.execution_logs` pelo `message id`:

- `status`
- `remote_jid`
- `response_payload`
- `error`

## Observacoes

- `MESSAGES_UPDATE` com status `error` ou `failed` vira `execution_logs.status = failed`.
- os demais estados mapeados nesta etapa viram `success`.
- o score agora e recalculado automaticamente em eventos de mensagem quando o backend consegue resolver `instanceName -> phoneNumberID`.

## Validacao

Testes executados:

```bash
go test ./internal/evolutionsync ./internal/httpserver ./internal/repository
go test -tags=integration ./internal/evolutionsync -run TestServiceSyncRecalculatesScoreRealDatabase -v
go test ./...
```

Resultado: `PASS`.
