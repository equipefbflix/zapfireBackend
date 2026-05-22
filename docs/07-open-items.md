# Bloqueios e proximos passos

## Estado atual do Supabase

O projeto Supabase alvo atual e `cqmxcsmpdshuncupcwaw`.

Verificacao consolidada ate 2026-05-13:

- o MCP resolve o projeto correto;
- o schema real esta documentado em `03-supabase-schema.md`;
- o backend usa `DATABASE_URL` para runtime real;
- os testes reais de banco seguem usando `testRunId` e cleanup ao final.

Chamadas MCP ja validadas contra `cqmxcsmpdshuncupcwaw`:

- `get_project_url`
- `list_tables`
- `list_migrations`
- `execute_sql`

Regra operacional:

- nao registrar chaves ou tokens retornados por MCP em arquivos versionaveis;
- antes de qualquer migration nova, reler o schema real e confirmar o `project_ref`.

## Confirmacoes pendentes

- Versao exata da Evolution API instalada em cada servidor.
- Se todas as 5 Evolution APIs terao a mesma versao.
- Se a criacao de instancias deve gerar token automaticamente ou usar token definido pelo backend.
- Se os proxies serao cadastrados no banco, no `.env`, ou em ambos.
- Politicas exatas de score de aquecimento.
- Quais acoes entram no MVP alem de texto, presenca, reply e reaction.
- Se havera painel frontend consumindo anon key do Supabase.
- Porta externa real do RabbitMQ. O teste com `rabbitmq.askgeni.us:5672` falhou por timeout TCP.

## Proximo passo tecnico recomendado

1. Validar webhook sync com eventos reais de instancia conectada em rotina repetivel, nao apenas execucao pontual.
2. Expandir planner real de conversas com frequencia por numero, janela de negocio e historico mais rico.
3. Validar integration real de dead-letter com cenario de falha controlada.
4. Expandir logs estruturados de publish/consume e correlacao entre `warming_jobs` e `execution_logs`.
5. Cobrir na UI os recursos de observabilidade e stale cleanup que ainda estao so no backend.

## Atualizacao em 2026-05-05

Concluido:

- warming score com persistencia em `phone_numbers`
- webhook sync recalculando score
- planner inicial com cooldown por par e janela
- retry e dead-letter policy no worker
- limite de concorrencia por par e por Evolution server

Ainda pendente para fechamento operacional:

- validacao real do dead-letter em cenario induzido
- correlacao mais rica de `send_status` por webhook, caso a Evolution passe a expor identificador retornavel
- automacao repetivel do loop reativo ponta a ponta com webhook publico e instancias dedicadas de QA
- validacao operacional externa com webhook publico continua manual; a cobertura automatizada agora usa a rota real via `httptest` e RabbitMQ local isolado

Atualizacao em 2026-05-05:

- webhook real da Evolution validado via ngrok
- rota `POST /api/v1/warming-jobs/{id}/run-now` implementada e validada
- fluxo real `route -> runner -> Evolution -> webhook -> Supabase` validado
- loop reativo `webhook inbound -> planner -> warming_job -> scheduler -> worker` validado com RabbitMQ local
- `cmd/scheduler` implementado e validado em runtime local
- `TestRabbitMQLocalRealFlow` passou com RabbitMQ local real em 2026-05-13
- `TestReactiveLoopRealE2E` passou em 2026-05-13 validando `webhook -> job -> queue -> worker -> runner -> execution_logs`
- placeholders `{{phoneA}}`, `{{phoneB}}`, `{{triggerMessageId}}` e `{{triggerRemoteJid}}` implementados no runner
- `send_typing` validado em job real
- `send_reply`, `send_reaction`, `send_audio` e `send_recording` validados em job real
- `send_status` validado em job real com aceite assíncrono controlado
- observabilidade minima e cleanup de jobs stale validados com banco real
- `Service.SyncState` validado contra Evolution real em `TestServiceSyncStateRealEvolutionInstanceE2E`

## Criterio para desbloquear

Estas chamadas MCP ja funcionam para `cqmxcsmpdshuncupcwaw`:

- `get_project_url`
- `list_tables`
- `list_migrations`
- `execute_sql` para leitura
- `list_extensions`
- `get_publishable_keys`

Para aplicar schema ou migrations pelo MCP, usar `apply_migration` com `project_id`.

## RabbitMQ de teste

Credenciais de teste devem ficar fora da documentacao versionavel. O backend espera receber `RABBITMQ_URL` por ambiente.

Validacao executada:

- Teste unitario de config/topologia/publisher passou.
- Teste real `go test -tags=integration ./internal/queue -run TestRabbitMQPublishConsumeReal -v` foi criado.
- Teste equivalente passou contra RabbitMQ local descartavel em Docker.
- `TestRabbitMQLocalRealFlow` passou contra `amqp://guest:guest@127.0.0.1:5672/` em 2026-05-13.

Se voltar a validar broker externo, confirmar:

- host e porta AMQP expostos;
- se usa `amqp://` na porta `5672` ou `amqps://` na porta `5671`;
- se o acesso esta liberado para a rede atual;
- se o vhost e `/`.

## Status implementado ate 2026-05-04

- Rotas auxiliares: phone numbers, proxies, Evolution servers.
- Templates de mensagens.
- Conversation scripts e steps.
- Warming jobs.
- Execution logs.
- Webhook Evolution persistindo `evolution_events`.
- Scheduler local que publica `warming.job.due`.
- Worker inicial que registra log `running`.
- Executor de steps para `send_presence`, `send_text` e `send_reaction`.
- Teste integrado protegido por build tag para fluxo minimo `phone_numbers -> warming_jobs -> execution_logs`.
- `DATABASE_URL` real validado no backend.
- Evolution real validada para `FetchInstances`.
- E2E real de criacao, persistencia e remocao de instancia validado com cleanup.
- Actions `send_audio`, `send_status`, `send_typing` e `send_recording` implementadas em client e executor, com testes unitarios.
- Primeiro bloco de `warming score` implementado e validado com banco real.
- Webhook sync agora atualiza `execution_logs` e recalcula `warming_score` com validacao integration real em banco.
- Primeiro bloco do planner real de conversas implementado e validado com banco real.
- Politica inicial de retry/dead-letter no worker implementada e coberta por testes unitarios.
