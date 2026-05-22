# SPEC: Replace Legacy Reactive Instances With Evolution Go Fixtures

## Contexto

As instancias antigas `connect_a_20260505` e `connect_b_20260505` foram removidas da Evolution antiga e limpas do banco porque nao serao mais usadas.

Ainda existem testes, script operacional e documentacao que dependem explicitamente dessas fixtures removidas:

- `backend/internal/httpserver/reactive_loop_real_e2e_test.go`
- `backend/scripts/send-inbound-probe.sh`
- `backend/docs/51-local-reactive-operations.md`

Hoje isso deixa o fluxo reativo real quebrado por fixture invalida. Alem disso, o backend ja foi migrado para o contrato da `evolution-go`, entao as novas fixtures precisam existir e ser operadas nesse novo ambiente, sem mocks e sem comportamento inventado.

## Solucao Proposta

Trocar todas as referencias operacionais fixas das instancias antigas por um novo par de fixtures reais na `evolution-go`.

A entrega deve:

1. criar ou referenciar um novo par de instancias reais de QA na `evolution-go`;
2. persistir essas instancias no banco com `instances`, `phone_numbers` e segredos correspondentes;
3. atualizar o E2E reativo real para usar as novas fixtures;
4. atualizar o script `send-inbound-probe.sh` para usar o contrato real atual da `evolution-go` e as novas fixtures;
5. atualizar o runbook `51-local-reactive-operations.md` para refletir:
   - `SERVER_URL` novo;
   - fluxo de webhook compatível com o backend atual;
   - nomes das novas instancias;
   - forma real de disparar mensagem inbound;
6. validar o fluxo real:
   - webhook -> `evolution_events` -> `conversationloop` -> `warming_job` -> RabbitMQ -> worker -> runner -> `execution_logs`

Nao entra neste slice:

- criar UI nova;
- manter compatibilidade com as instancias antigas;
- mockar Evolution ou RabbitMQ.

## Arquivos que Serao Criados/Modificados

- `backend/docs/specs/SPEC_replace_legacy_reactive_instances_with_evolution_go_fixtures.md` — especificacao deste slice
- `backend/internal/httpserver/reactive_loop_real_e2e_test.go` — trocar fixtures antigas por fixtures novas reais da `evolution-go`
- `backend/scripts/send-inbound-probe.sh` — usar instancias novas e o endpoint real compativel com o ambiente atual
- `backend/docs/51-local-reactive-operations.md` — atualizar o runbook operacional para a nova topologia
- `backend/internal/evolution/*` — somente se os testes mostrarem delta real adicional de contrato
- `backend/internal/instance/*` — somente se os testes mostrarem necessidade real para preparar fixtures novas

## Criterios de Aceite

- [ ] Existe um par novo de fixtures reais de QA na `evolution-go`, substituindo `connect_a_20260505` e `connect_b_20260505`
- [ ] `TestReactiveLoopRealE2E` usa apenas as novas fixtures e nao referencia mais as antigas
- [ ] `TestReactiveLoopRealE2E` falha inicialmente pelo motivo correto antes da implementacao
- [ ] `TestReactiveLoopRealE2E` passa com banco real, RabbitMQ real e Evolution real
- [ ] `send-inbound-probe.sh` nao referencia mais `connect_b_20260505`
- [ ] `send-inbound-probe.sh` usa o contrato real do ambiente atual
- [ ] `51-local-reactive-operations.md` nao referencia mais as instancias antigas nem `https://evo.askgeni.us`
- [ ] O cleanup do teste continua removendo apenas os dados do `testRunId` da execucao

## Casos de Borda

- O que acontece se a fixture nova existir na Evolution mas nao no banco?
- O que acontece se a fixture existir no banco mas estiver desconectada?
- O que acontece se o webhook chegar com `instance` diferente das fixtures esperadas?
- O que acontece se o probe enviar mensagem para uma instancia nao conectada?
- O que acontece se o token da instancia nao estiver resolvivel no ambiente?

## Impacto e Riscos

- O E2E reativo depende de infraestrutura real compartilhada; fixture mal isolada pode reintroduzir flakiness.
- A troca do script operacional muda o caminho oficial de validacao local.
- Se as novas fixtures nao forem persistidas com o servidor/token corretos, o runner vai falhar durante a execucao real.
- O runbook precisa refletir exatamente o ambiente atual para nao induzir operacao errada.

## Estrutura de Testes Planejada

- `TestReactiveLoopRealE2E` — valida o loop reativo completo com as novas fixtures reais
- `TestSendInboundProbeContract` ou equivalente — se necessario, valida o contrato do script/endpoint esperado sem inventar comportamento
- rerun de `TestEvolutionFetchInstancesReal` — confirma que as fixtures novas existem no ambiente real

## Status: AGUARDANDO APROVACAO
