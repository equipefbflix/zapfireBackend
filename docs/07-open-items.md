# Bloqueios e proximos passos

## Estado atual do Supabase

O projeto Supabase alvo atual e `rxdophybnwoocsdyxyjm`.

Verificacao em 2026-04-30:

- Em 2026-05-04, o MCP foi configurado como global, sem `project_ref`.
- As operacoes agora precisam receber `project_id = rxdophybnwoocsdyxyjm` quando a ferramenta exigir.
- `list_projects` lista todos os projetos da conta, incluindo `Zap Fire` (`rxdophybnwoocsdyxyjm`).
- O schema do aquecedor foi aplicado no projeto correto.
- O schema aplicado anteriormente no projeto errado `cqmxcsmpdshuncupcwaw` foi removido; la sobraram apenas as tabelas preexistentes.

Chamadas testadas contra `rxdophybnwoocsdyxyjm`:

- `get_project_url`
- `list_tables`
- `list_migrations`
- `execute_sql`
- `get_logs(service=postgres)`
- `get_advisors(type=security)`
- `list_extensions`
- `get_publishable_keys`

Resultado:

- MCP funcional.
- Schema real listado em `03-supabase-schema.md`.
- `public` continha tabelas existentes `profiles`, `plans`, `subscriptions` e `messages` antes do aquecedor.
- Nao registrar chaves retornadas por `get_publishable_keys` em arquivos versionaveis.

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

1. Validar RabbitMQ real com `RABBITMQ_URL`, publicacao, consumo e `ack`.
2. Fechar `send_status` como operacao confiavel no backend. As demais actions novas do MVP ja foram validadas em instancia conectada real.
3. Criar E2E real do scheduler/worker consumindo fila e gravando `execution_logs`.
4. Validar webhook sync com eventos reais de instancia conectada.
5. Expandir planner real de conversas com frequencia por numero, janela de negocio e historico mais rico.
6. Validar integration real de dead-letter quando houver broker acessivel nesta sessao.
7. Expandir logs estruturados de publish/consume quando o broker externo estiver operando.

## Atualizacao em 2026-05-05

Concluido:

- warming score com persistencia em `phone_numbers`
- webhook sync recalculando score
- planner inicial com cooldown por par e janela
- retry e dead-letter policy no worker
- limite de concorrencia por par e por Evolution server

Ainda pendente para fechamento operacional:

- validacao real do dead-letter em broker acessivel
- RabbitMQ externo real
- E2E externo completo `scheduler -> queue -> worker -> runner`
- correlacao mais rica de `send_status` por webhook, caso a Evolution passe a expor identificador retornavel
- RabbitMQ externo real

Atualizacao em 2026-05-05:

- webhook real da Evolution validado via ngrok
- rota `POST /api/v1/warming-jobs/{id}/run-now` implementada e validada
- fluxo real `route -> runner -> Evolution -> webhook -> Supabase` validado
- loop reativo `webhook inbound -> planner -> warming_job -> scheduler -> worker` validado com RabbitMQ local
- `cmd/scheduler` implementado e validado em runtime local
- placeholders `{{phoneA}}`, `{{phoneB}}`, `{{triggerMessageId}}` e `{{triggerRemoteJid}}` implementados no runner
- `send_typing` validado em job real
- `send_reply`, `send_reaction`, `send_audio` e `send_recording` validados em job real
- `send_status` validado em job real com aceite assíncrono controlado
- observabilidade minima e cleanup de jobs stale validados com banco real

## Criterio para desbloquear

Estas chamadas MCP ja funcionam para `rxdophybnwoocsdyxyjm`:

- `get_project_url`
- `list_tables`
- `list_migrations`
- `execute_sql` para leitura
- `list_extensions`
- `get_publishable_keys`

Para aplicar schema ou migrations pelo MCP, usar `apply_migration` com `project_id`.

## RabbitMQ de teste

Credenciais de teste foram fornecidas fora da documentacao versionavel. O backend espera receber `RABBITMQ_URL` por ambiente.

Validacao executada:

- Teste unitario de config/topologia/publisher passou.
- Teste real `go test -tags=integration ./internal/queue -run TestRabbitMQPublishConsumeReal -v` foi criado.
- Teste equivalente passou contra RabbitMQ local descartavel em Docker.
- A tentativa real falhou por timeout TCP em `rabbitmq.askgeni.us:5672`.

Para desbloquear o teste real, confirmar:

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
