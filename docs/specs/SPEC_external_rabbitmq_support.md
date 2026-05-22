# SPEC: Suporte ao RabbitMQ Externo

## Contexto

O backend ja suporta RabbitMQ por `RABBITMQ_URL` e ja foi validado contra broker local. Agora o sistema precisa operar tambem contra um RabbitMQ externo fornecido pelo ambiente da Zaapfire, para que scheduler, worker e loop reativo possam rodar sem depender do broker local.

O requisito novo e:

- suportar uso do broker externo por ambiente;
- retestar publish/consume e fluxo operacional real contra esse broker;
- nao gravar credenciais sensiveis em arquivos versionados.

## Solucao Proposta

Manter a configuracao do broker por `RABBITMQ_URL`, sem embedar segredo em codigo ou docs operacionais versionaveis.

Entregas deste slice:

1. revisar a camada de config/queue para garantir compatibilidade com broker externo;
2. adicionar testes que exercitem explicitamente o caminho de broker externo via ambiente;
3. validar o fluxo real de publish/consume;
4. validar pelo menos um fluxo operacional real usando scheduler/worker/runner no broker externo;
5. atualizar o runbook para deixar claro como apontar o backend para o broker externo usando somente env.

Observacao importante de conectividade:

- a URL fornecida hoje usa host `rabbitmq`, que normalmente e um nome interno de rede Docker/Kubernetes;
- para acesso externo, o host esperado tende a ser `rabbitmq.zaapfire.com.br`;
- o slice deve confirmar isso em teste real antes de marcar suporte externo como concluido.

## Arquivos que Serao Criados/Modificados

- `backend/docs/specs/SPEC_external_rabbitmq_support.md` — especificacao do suporte ao broker externo
- `backend/internal/config/rabbitmq_test.go` — cobrir DSN externo e validacoes associadas
- `backend/internal/queue/rabbitmq_integration_test.go` — revalidar publish/consume no broker externo via env
- `backend/internal/workerapp/rabbitmq_local_e2e_test.go` — generalizar o E2E para broker real configurado por ambiente, nao so local
- `backend/internal/httpserver/reactive_loop_real_e2e_test.go` — permitir execucao real usando broker externo configurado por ambiente
- `backend/docs/05-env.md` — reforcar uso via env sem gravar segredos
- `backend/docs/09-rabbitmq-queues.md` — registrar suporte operacional ao broker externo
- `backend/docs/35-worker-local-e2e.md` — atualizar runbook para incluir broker externo

## Criterios de Aceite

- [ ] o backend continua lendo RabbitMQ exclusivamente por `RABBITMQ_URL`
- [ ] nenhum segredo do broker externo e gravado em codigo, docs versionaveis ou fixtures
- [ ] os testes de integracao de fila passam contra o broker externo quando `ENABLE_REAL_TESTS=true` e `RABBITMQ_URL` apontam para ele
- [ ] pelo menos um E2E real de operacao (`queue -> worker -> runner -> execution_logs`) passa usando o broker externo
- [ ] o loop reativo real continua funcional quando configurado com o broker externo
- [ ] a documentacao deixa claro como configurar o broker externo por ambiente

## Casos de Borda

- host `rabbitmq` nao resolver fora da rede interna
- broker externo exigir TLS em `amqps://` na porta `5671`, em vez de `amqp://` na `5672`
- credencial valida mas vhost/permissoes insuficientes para declarar exchange, queues ou bindings
- dead-letter funcionar de forma diferente se o broker estiver preconfigurado com politicas proprias
- latencia externa expor timeouts nao vistos no broker local

## Impacto e Riscos

- risco de timeout ou falha de DNS se a URL externa estiver incorreta
- risco de ambiente externo ter politicas de permissao diferentes das usadas no broker local
- risco operacional se testes reais usarem nomes de fila de producao sem isolamento; todos os testes devem continuar usando nomes com `testRunId`

## Estrutura de Testes Planejada

- `TestLoadRabbitMQConfigAcceptsExternalURL`
- `TestRabbitMQPublishConsumeRealExternalBroker`
- `TestRabbitMQWorkerFlowRealExternalBroker`
- `TestReactiveLoopRealE2EExternalBroker`

## Status: AGUARDANDO APROVACAO
