# Plano dedicado: sync por webhook

Objetivo: refletir no banco o estado real das instancias e mensagens a partir dos webhooks da Evolution.

## Escopo

- mapear `CONNECTION_UPDATE`
- mapear `MESSAGES_UPSERT`
- mapear `MESSAGES_UPDATE`
- atualizar `instances`, `execution_logs` e score

## TDD

1. Documentar tabela de mapeamento `event -> efeito no banco`.
2. Criar testes de handler/service:
   - webhook de conexao abre instancia
   - webhook de desconexao pausa instancia
   - webhook de mensagem enviada atualiza `execution_log`
3. Rodar e confirmar falha.
4. Implementar service de sincronizacao.
5. Ajustar ate passar.
6. Criar teste integration com banco real:
   - inserir dados reais minimos
   - postar webhook de teste
   - verificar atualizacao no banco
   - limpar dados

## Entregaveis

- normalizador de eventos
- service de sync
- update em `instances`
- update em `execution_logs`

## Bloqueios

Para validacao real completa, precisa de instancia conectada emitindo eventos reais.

## Status em 2026-05-04

Primeiro subbloco concluido:

- `CONNECTION_UPDATE` sincroniza `instances`
- `MESSAGES_UPDATE`, `MESSAGES_UPSERT` e `SEND_MESSAGE` sincronizam `execution_logs`
- `evolution_events` passam a ser marcados com `processed_at`
- eventos de mensagem passam a disparar recalc de `warming_score`

Pendencias deste plano:
- teste integration com banco real
- validacao com eventos reais de instancia conectada

Atualizacao:

- o teste integration com banco real do recalc por webhook ja passou
- ainda falta a validacao com eventos reais de instancia conectada
