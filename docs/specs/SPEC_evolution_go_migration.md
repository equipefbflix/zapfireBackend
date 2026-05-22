# SPEC: Migracao para Evolution Go

## Contexto

O backend e os testes reais do aquecedor estao acoplados hoje ao contrato da Evolution API v2 descrito em `backend/docs/02-evolution-api.md`, com endpoints no formato:

- `GET /instance/fetchInstances`
- `GET /instance/connect/{instance}`
- `GET /instance/connectionState/{instance}`
- `POST /message/sendText/{instance}`
- `POST /chat/sendPresence/{instance}`
- `POST /message/sendReaction/{instance}`
- `POST /message/sendMedia/{instance}`
- `POST /message/sendSticker/{instance}`
- `POST /message/sendStatus/{instance}`

Agora o sistema precisa usar a versao `evolution-go`, publicada em:

- `https://go.zaapfire.com.br/swagger/index.html`

O Swagger da `evolution-go` exibe um contrato diferente, com endpoints e payloads novos, por exemplo:

- `POST /instance/create`
- `POST /instance/connect`
- `GET /instance/status`
- `GET /instance/all`
- `POST /instance/reconnect`
- `GET /instance/qr`
- `POST /instance/pair`
- `POST /send/text`
- `POST /message/presence`
- `POST /send/media`
- `POST /send/sticker`
- `POST /message/react`

Essa diferenca nao e apenas configuracao. Ela afeta o client HTTP, a camada de execucao de steps, a criacao/conexao/sync de instancias, os testes reais e a documentacao operacional.

## Solucao Proposta

Migrar a integracao do backend para o padrao da `evolution-go`, preservando o contrato interno do backend sempre que possivel.

Abordagem:

1. adaptar `backend/internal/evolution/client.go` para falar com a `evolution-go`;
2. manter a interface consumida por `instance.Service` e `executor.StepExecutor` o mais estavel possivel;
3. mapear os requests internos atuais para os payloads novos da `evolution-go`;
4. atualizar os testes unitarios do client e executor;
5. atualizar os testes reais/E2E para validar a nova Evolution;
6. atualizar a documentacao tecnica da integracao.

Diretriz importante:

- o frontend continua consumindo o backend do aquecedor, nao a Evolution diretamente;
- portanto, a primeira meta e manter estavel o contrato HTTP do backend;
- o frontend so deve ser alterado se a resposta de pareamento/estado real exigir ajuste para continuar exibindo QR, pairing code ou status corretamente.

## Arquivos que Serao Criados/Modificados

- `backend/docs/specs/SPEC_evolution_go_migration.md` — especificacao da migracao
- `backend/docs/02-evolution-api.md` — atualizar para o contrato da `evolution-go`
- `backend/internal/evolution/client.go` — migrar endpoints, payloads e parsing
- `backend/internal/evolution/client_test.go` — reescrever testes do client contra o contrato novo
- `backend/internal/evolution/client_integration_test.go` — retestar leitura real de instancias/status na `evolution-go`
- `backend/internal/executor/step_executor.go` — ajustar mapeamento de actions para endpoints/payloads novos
- `backend/internal/executor/step_executor_test.go` — cobrir os novos mapeamentos
- `backend/internal/instance/service.go` — ajustar create/connect/sync/restart conforme o client novo
- `backend/internal/instance/service_test.go` — validar contratos atualizados do service
- `backend/internal/instance/service_e2e_test.go` — revalidar criacao/conexao/sync/restart reais
- `backend/internal/instance/service_onboarding_e2e_test.go` — revalidar onboarding real com nome da instancia
- `frontend/src/pages/ConectarAparelho.tsx` — ajustar apenas se o backend precisar devolver formato diferente para pareamento
- `frontend/src/features/backend/services/management.real.test.ts` — revalidar fluxo real do wizard/pareamento via backend

## Criterios de Aceite

- [ ] `CreateInstance` passa a usar o contrato da `evolution-go` e continua permitindo criacao operacional do backend
- [ ] `ConnectInstance` passa a usar o contrato da `evolution-go` e o backend continua expondo dados reais de pareamento
- [ ] `ConnectionState` ou equivalente passa a ler o status real da `evolution-go` e atualiza `instances.status`
- [ ] `FetchInstances` ou equivalente continua permitindo reconciliacao real de instancias
- [ ] `RestartInstance` ou equivalente funciona no novo contrato
- [ ] `send_text`, `send_reply`, `send_typing`, `send_recording`, `send_reaction`, `send_media`, `send_sticker` e `send_status` ficam mapeados para o padrao novo conforme suportado pela `evolution-go`
- [ ] os testes unitarios do client/executor falham primeiro contra o contrato antigo e passam apos a migracao
- [ ] os testes reais de instancia e pelo menos um fluxo real de execucao de mensagem passam com a `evolution-go`
- [ ] a documentacao do backend deixa de descrever o contrato antigo como fonte principal

## Casos de Borda

- `instance/connect` da `evolution-go` pode nao devolver o mesmo shape de `pairingCode`, `code` e `count`
- `instance/status` pode substituir `connectionState` com payload diferente
- `instance/all` pode substituir `fetchInstances` com shape diferente de listagem
- `message/status` no Swagger da `evolution-go` parece ser consulta de status de mensagem, nao envio de status/story; isso precisa ser confirmado antes de migrar `send_status`
- `message/presence` na `evolution-go` usa `state` e `isAudio`, diferente do modelo atual com `presence`
- `send/media` e `send/sticker` usam payloads novos com `url`, `type`, `filename`, `quoted` e `id`
- a autenticacao pode continuar em `apikey`, mas isso deve ser confirmado em teste real
- algum recurso hoje usado pelo aquecedor pode nao existir com a mesma semantica na `evolution-go`; nesses casos a SPEC nao autoriza inventar mock ou fallback silencioso

## Impacto e Riscos

- alto impacto no backend, porque a Evolution e a espinha dorsal de instancia, mensagens, score e loop reativo
- risco de regressao no wizard existente se o pareamento mudar
- risco de `send_status` nao ter equivalente direto para story/status post no contrato novo
- risco de diferencas de payload em quoted/reaction quebrarem scripts existentes
- exige nova rodada de validacao real; nao basta teste unitario

## Estrutura de Testes Planejada

- `TestClientCreateInstanceEvolutionGo`
- `TestClientConnectInstanceEvolutionGo`
- `TestClientConnectionStateEvolutionGo`
- `TestClientFetchInstancesEvolutionGo`
- `TestClientSendTextEvolutionGo`
- `TestClientSendPresenceEvolutionGo`
- `TestClientSendMediaEvolutionGo`
- `TestClientSendStickerEvolutionGo`
- `TestClientSendReactionEvolutionGo`
- `TestStepExecutorSendTextEvolutionGo`
- `TestStepExecutorSendReactionEvolutionGo`
- `TestStepExecutorSendMediaEvolutionGo`
- `TestStepExecutorSendStickerEvolutionGo`
- `TestServiceCreateRealEvolutionGoInstanceE2E`
- `TestServiceCreateRealEvolutionGoOnboardingInstanceWithoutPhoneE2E`
- `TestEvolutionGoFetchInstancesReal`
- `management.real test` do fluxo de onboarding/pareamento via backend local

## Status: AGUARDANDO APROVACAO
