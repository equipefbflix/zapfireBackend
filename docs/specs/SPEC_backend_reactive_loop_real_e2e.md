# SPEC: Backend Reactive Loop Real E2E

## Contexto

O backend ja possui as pecas do loop reativo:

- webhook da Evolution;
- persistencia de `evolution_events`;
- `evolutionsync`;
- `conversationloop`;
- planner;
- scheduler;
- worker;
- runner.

Tambem ja existe documentacao operacional local e validacoes pontuais com:

- webhook real via ngrok;
- planner criando `warming_jobs`;
- scheduler + RabbitMQ local + worker + runner gravando `execution_logs`.

O gap atual e fechar isso como um fluxo ponta a ponta reproduzivel, com teste real protegido, dados marcados por `testRunId` e cleanup obrigatorio. Hoje o sistema funciona por partes, mas ainda nao existe um artefato unico de validacao que prove de forma repetivel:

`webhook real -> evolutionsync -> conversationloop -> warming_job -> scheduler -> RabbitMQ -> worker -> runner -> execution_logs`

## Solucao Proposta

Implementar um slice dedicado de validacao real do loop reativo, com:

1. teste E2E protegido por tag/build adequado e `ENABLE_REAL_TESTS=true`;
2. uso de RabbitMQ local real;
3. uso de banco real via `DATABASE_URL`;
4. uso de Evolution real via instancia conectada e webhook publico configurado;
5. dados de teste marcados por `testRunId`;
6. cleanup de jobs, logs, scripts, eventos e registros auxiliares.

Esse slice deve produzir dois resultados concretos:

- um teste real reproduzivel;
- um runbook curto e correto para repetir a operacao manualmente.

## Arquivos que Serao Criados/Modificados

- `backend/internal/...` — testes e ajustes minimos necessarios para fechar o fluxo reativo ponta a ponta
- `backend/docs/51-local-reactive-operations.md` — consolidar o runbook do loop reativo
- `backend/docs/07-open-items.md` — atualizar o status do gap

## Criterios de Aceite

- [ ] Existe um teste real protegido que valida o fluxo `webhook -> job -> queue -> worker -> logs`
- [ ] O teste usa `testRunId` e remove todos os dados criados ao final
- [ ] O teste confirma que o `warming_job` criado pelo inbound reativo chega a `success` ou registra erro esperado e verificavel
- [ ] O teste confirma a existencia de `execution_logs` vinculados ao job gerado
- [ ] O runbook local descreve pre-requisitos, subida de RabbitMQ local, API, scheduler, worker, webhook publico e passo de validacao
- [ ] A documentacao deixa explicito o que depende de infraestrutura manual externa, especialmente webhook publico e instancias de QA conectadas

## Casos de Borda

- O webhook chega para instancia nao gerenciada pelo aquecedor
- O inbound chega dentro do cooldown e nenhum novo job deve ser criado
- O webhook chega sem segredo valido quando `WEBHOOK_EVOLUTION_SECRET` estiver configurado
- O RabbitMQ local nao esta ativo
- O webhook cria o job, mas o worker nao consome por indisponibilidade do broker

## Impacto e Riscos

- Esse slice toca integracao real com servicos externos e pode falhar por infraestrutura, nao por regressao de codigo
- O teste precisa ser explicitamente protegido para nao rodar em CI sem ambiente preparado
- O cleanup precisa ser rigoroso para nao deixar lixo em `warming_jobs`, `execution_logs`, `evolution_events` e scripts de QA

## Estrutura de Testes Planejada

- `TestReactiveLoopRealE2E` — valida o fluxo completo com webhook real, job criado, fila consumida e logs persistidos
- `TestReactiveLoopRealE2ERequiresBroker` — valida falha clara quando o RabbitMQ local nao estiver acessivel, se esse comportamento ainda nao estiver coberto por teste existente

## Status: AGUARDANDO APROVACAO
