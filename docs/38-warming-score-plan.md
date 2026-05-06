# Plano dedicado: warming score

Objetivo: calcular e persistir a porcentagem de aquecimento por numero, com transicoes de status.

## Escopo

- score percentual em `phone_numbers`
- regras vindas do `.env`
- transicoes de `phone_status`
- atualizacao a partir de `execution_logs` e eventos Evolution

## TDD

1. Documentar regras de score e thresholds.
2. Criar testes unitarios para o calculador:
   - soma ponderada por action
   - limites minimo e maximo
   - transicao `new -> warming -> warm`
   - casos de regressao por falha/bloqueio
3. Rodar e confirmar falha.
4. Implementar `internal/warmingscore`.
5. Ajustar ate passar.
6. Criar teste integration com banco real:
   - criar phone
   - criar execution logs reais
   - recalcular score
   - persistir score/status
   - limpar dados

## Entregaveis

- service de score
- repositorio/update de `phone_numbers`
- job de recalc
- doc de envs e pesos

## Bloqueios

Nenhum externo. Pode seguir sem RabbitMQ real.

## Status em 2026-05-05

Primeiro subbloco concluido:

- `WarmingConfig` carregado de env
- metricas consolidadas a partir de `execution_logs`
- service `warmingscore.Recalculate`
- persistencia de `warming_score` e `phone_status`
- teste integration real em banco passou

Pendencias deste plano:

- recalculo disparado automaticamente por webhook
- endpoint/cron dedicado de recalc em lote
- incluir mais sinais no score alem de logs basicos
