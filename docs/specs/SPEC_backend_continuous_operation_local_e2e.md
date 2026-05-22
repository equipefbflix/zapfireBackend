# SPEC: Backend Continuous Operation Local E2E

## Contexto

O backend ja possui scheduler, worker, runner e topologia RabbitMQ, mas a camada de operacao continua precisa ficar formalmente fechada com um caminho repetivel em ambiente local controlado.

## Solucao Proposta

Consolidar como caminho oficial local:

- `cmd/scheduler`
- `cmd/worker`
- RabbitMQ local
- banco real
- runner real
- `execution_logs`

Manter e validar um E2E real com broker local aceitavel e registrar um runbook curto para reproducao.

## Arquivos que Serao Criados/Modificados

- `backend/docs/specs/SPEC_backend_continuous_operation_local_e2e.md`
- `backend/internal/workerapp/rabbitmq_local_e2e_test.go`
- `backend/docs/35-worker-local-e2e.md`
- `backend/docs/45-local-continuous-operation-runbook.md`

## Criterios de Aceite

- [ ] Existe E2E real local `scheduler -> RabbitMQ -> worker -> runner -> execution_logs`
- [ ] O teste usa `testRunId` e limpa os dados reais criados
- [ ] O runbook documenta como subir broker local, scheduler e worker
- [ ] O criterio nao depende de RabbitMQ externo

## Casos de Borda

- broker local indisponivel
- job publicado mas nao consumido
- runner falha e job precisa marcar `failed`

## Impacto e Riscos

- risco baixo de codigo, foco maior em consolidacao operacional
- depende de ambiente local com RabbitMQ acessivel

## Estrutura de Testes Planejada

- `TestRabbitMQLocalRealFlow`
- `TestRabbitMQLocalRealFlowMarksFailedOnRunnerError`

## Status: APROVADO PARA IMPLEMENTACAO
