# Plano dedicado: resiliencia operacional

Objetivo: endurecer o backend para operacao continua.

## Escopo

- retry controlado
- dead-letter handling
- limites por Evolution server
- limites por instancia
- logs e metricas basicas

## TDD

1. Documentar politicas de retry e descarte.
2. Criar testes unitarios:
   - step falho gera `nack/requeue`
   - excedeu tentativas vai para dead-letter
   - limite de concorrencia barra nova execucao
3. Rodar e confirmar falha.
4. Implementar politicas no worker/runner.
5. Ajustar ate passar.
6. Criar teste integration/local:
   - publicar mensagem invalida
   - verificar dead-letter
   - simular job concorrente

## Politica inicial

- `attempt <= RABBITMQ_MAX_RETRIES`: `nack(requeue=true)`
- `attempt > RABBITMQ_MAX_RETRIES`: `nack(requeue=false)` para seguir ao dead-letter exchange

## Entregaveis

- retry policy
- dead-letter policy
- locks/limites de execucao
- health/observability minima

## Bloqueios

Nenhum para a parte local. Para validacao externa completa, depende do broker real.

## Status em 2026-05-05

Primeiro subbloco concluido:

- politica de retry por tentativa
- descarte para dead-letter acima do limite
- `RABBITMQ_MAX_RETRIES`
- leitura de tentativa via `x-death.count`

Pendencias deste plano:

- validacao integration real do dead-letter com broker acessivel
- observabilidade minima

Subbloco concluido em 2026-05-05:

- limites de concorrencia por par
- limites de concorrencia por Evolution server
- transicao de status do job no runner: `running`, `success`, `failed`
- validacao integration real no Supabase para contadores e `UpdateStatus`

Variaveis novas:

```env
WARMING_MAX_RUNNING_JOBS_PER_PAIR=1
WARMING_MAX_RUNNING_JOBS_PER_EVOLUTION_SERVER=5
```

Pendencias remanescentes:

- validacao integration real do dead-letter com broker acessivel
- observabilidade minima
