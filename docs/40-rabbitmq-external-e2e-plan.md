# Plano dedicado: E2E externo com RabbitMQ

Objetivo: fechar o fluxo real:

`scheduler -> RabbitMQ -> worker -> runner -> execution_logs`

## Escopo

- publicar job real
- consumir no worker real
- `ack` em sucesso
- `nack/requeue` em erro

## TDD

1. Documentar topologia final de exchange, queues e dead-letter.
2. Criar teste integration real contra broker:
   - declarar topologia
   - publicar `warming.job.due`
   - consumir
   - `ack`
3. Rodar e confirmar falha, se houver.
4. Ajustar infraestrutura/codigo.
5. Criar E2E do worker:
   - dados reais no banco
   - mensagem na fila
   - worker consome
   - runner grava `execution_logs`
   - cleanup

## Entregaveis

- integration RabbitMQ real
- E2E real do worker
- documentacao operacional de filas

## Bloqueios

Hoje o bloqueio e externo: timeout TCP em `rabbitmq.askgeni.us:5672`.

## Observacao

A validacao local em Docker ja passou. O problema restante nao esta no codigo da camada RabbitMQ.
