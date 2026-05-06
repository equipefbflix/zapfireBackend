# Warming score

Esta feature implementa o primeiro calculo persistido de score de aquecimento por numero.

## Escopo desta etapa

- carregar pesos de score por `.env`
- consolidar metricas por `phone_number` a partir de `execution_logs`
- recalcular score
- atualizar `phone_numbers.warming_score`
- atualizar `phone_numbers.status`

## Formula inicial

```text
score =
  send_text_success     * WARMING_SCORE_MESSAGE_SUCCESS
  + send_reply_success  * WARMING_SCORE_REPLY_SUCCESS
  + send_reaction_success * WARMING_SCORE_REACTION_SUCCESS
  + bonus_atividade_dia * WARMING_SCORE_DAILY_ACTIVE_BONUS
  - failures            * WARMING_SCORE_FAILURE_PENALTY
  - disconnects         * WARMING_SCORE_DISCONNECTED_PENALTY
```

Clamp:

- minimo `0`
- maximo `100`

Transicao de status:

- `0` -> `new`
- `> 0` e `< WARMING_MIN_SCORE_TO_MARK_WARM` -> `warming`
- `>= WARMING_MIN_SCORE_TO_MARK_WARM` -> `warm`

## TDD aplicado

1. testes de config:
   - `internal/config/warming_test.go`
2. teste de repository:
   - `PhoneNumberRepository.UpdateWarmingState`
3. testes do service:
   - `TestServiceRecalculateWarm`
   - `TestServiceRecalculateWarming`
   - `TestServiceRecalculateNewWhenNoActivity`
4. os testes falharam por ausencia de:
   - `WarmingConfig`
   - `UpdateWarmingState`
   - `PhoneWarmingMetrics`
   - `warmingscore.Service`
5. implementacao concluida
6. validacao real em banco com `service_integration_test.go`

## Validacao local

```bash
go test ./internal/config ./internal/repository ./internal/warmingscore
```

## Validacao real

```bash
ENABLE_REAL_TESTS=true \
DATABASE_URL='postgresql://postgres:***@db.rxdophybnwoocsdyxyjm.supabase.co:5432/postgres' \
TEST_RUN_ID='warming-score-manual-001' \
go test -tags=integration ./internal/warmingscore -run TestServiceRecalculateRealDatabase -v
```
