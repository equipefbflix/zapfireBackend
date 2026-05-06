# Observabilidade minima e cleanup de jobs presos

Data: 2026-05-05

## Objetivo

Fechar o bloco operacional que nao depende do RabbitMQ externo:

- health operacional
- metrics basicas
- cleanup de `warming_jobs` presos em `running`

## Rotas

- `GET /health`
- `GET /api/v1/health`
- `GET /api/v1/metrics`
- `POST /api/v1/warming-jobs/stale-cleanup`

## Dados expostos

`/api/v1/metrics` retorna:

- janela de observabilidade
- janela de stale running
- contagem de jobs por status
- quantidade de jobs `running` considerados stale
- quantidade de `execution_logs` com falha na janela
- quantidade de `evolution_events` na janela

`/health` agora:

- retorna `supabase.status = healthy` quando o backend esta com observabilidade ligada ao banco
- retorna `status = degraded` quando existem jobs stale
- incorpora o snapshot de metrics

## Cleanup de jobs stale

A rota `POST /api/v1/warming-jobs/stale-cleanup`:

- encontra jobs `running`
- compara `coalesce(started_at, updated_at, created_at)` com a janela configurada
- marca como `failed`
- grava `error` com o motivo configurado

## Variaveis novas

```env
OBSERVABILITY_LOOKBACK_MINUTES=60
WARMING_STALE_RUNNING_MINUTES=20
WARMING_STALE_CLEANUP_REASON=stale running job cleanup
```

## Validacao real

Validado no projeto Supabase `rxdophybnwoocsdyxyjm`:

1. criado 1 `warming_job` artificial em `running` com `started_at` antigo
2. `GET /api/v1/metrics` retornou:
   - `running = 1`
   - `staleRunningJobs = 1`
3. `GET /health` retornou `status = degraded`
4. `POST /api/v1/warming-jobs/stale-cleanup` retornou `{"affected":1}`
5. `GET /api/v1/metrics` voltou com:
   - `staleRunningJobs = 0`
6. `GET /health` voltou para `status = ok`

## Arquivos

- `internal/config/observability.go`
- `internal/observability/service.go`
- `internal/repository/warming_jobs.go`
- `internal/repository/execution_logs.go`
- `internal/repository/evolution_events.go`
- `internal/httpserver/server.go`
