# E2E real de criacao de instancia

Esta feature fecha o ciclo real de criacao de instancia:

`service.Create -> Evolution /instance/create -> persistencia em public.instances -> Evolution /instance/delete -> cleanup no banco`

## Escopo

- Usa Postgres real via `DATABASE_URL`.
- Usa Evolution real via `EVOLUTION_TEST_BASE_URL` e `EVOLUTION_TEST_API_KEY`.
- Cria `phone_numbers` e `evolution_servers` de teste no Supabase.
- Cria uma instancia real na Evolution.
- Persiste a instancia em `public.instances`.
- Remove a instancia na Evolution.
- Remove os dados de teste do banco.

## TDD aplicado

1. Teste unitario criado para `Client.DeleteInstance`.
2. Teste unitario criado para aceitar resposta real de `CreateInstance` com `hash` como `string`.
3. `go test ./internal/evolution ./internal/instance` falhou porque o client nao tinha `DeleteInstance` e nao aceitava o contrato real do `hash`.
4. Implementado `DeleteInstance`.
5. Implementado parser flexivel para `CreateInstanceResponse.Hash`, aceitando:
   - `{ "apikey": "..." }`
   - `"API_KEY_EM_STRING"`
6. E2E real criado em `internal/instance/service_e2e_test.go`.
7. E2E real executado com cleanup confirmado.

## Execucao

```bash
ENABLE_REAL_TESTS=true \
DATABASE_URL='postgresql://postgres:***@db.rxdophybnwoocsdyxyjm.supabase.co:5432/postgres' \
EVOLUTION_TEST_BASE_URL='https://evo.askgeni.us' \
EVOLUTION_TEST_API_KEY='***' \
TEST_RUN_ID='instance-e2e-manual-002' \
go test -tags=e2e ./internal/instance -run TestServiceCreateRealEvolutionInstanceE2E -v
```

## Resultado real em 2026-05-04

- `TestServiceCreateRealEvolutionInstanceE2E`: `PASS`
- Tempo observado: aproximadamente `8.78s`

## Contratos reais observados

Na Evolution real usada neste projeto:

- `POST /instance/create` devolveu `hash` como `string`, nao como objeto.
- `DELETE /instance/delete/{instance}` funcionou com `200`.
- `GET /instance/fetchInstances?instanceName=...` apos delete devolveu `404` com mensagem `Instance "...\" not found`, e isso foi tratado como estado valido de cleanup.

## Cleanup

- Banco:
  - `public.instances`
  - `public.phone_numbers`
  - `public.evolution_servers`
- Evolution:
  - `DELETE /instance/delete/{instance}`

## Arquivos

- `internal/evolution/client.go`
- `internal/evolution/client_test.go`
- `internal/instance/service_e2e_test.go`
