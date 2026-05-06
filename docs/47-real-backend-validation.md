# Validacao real via backend

Data: 2026-05-05

## Objetivo

Validar o fluxo real do sistema usando:

- backend Go local
- ngrok publico para webhook
- Evolution real
- Supabase real
- duas instancias WhatsApp conectadas

## Problemas encontrados e corrigidos

### 1. Webhook `byEvents=true` batendo em subpaths

A Evolution enviou eventos para:

- `/api/v1/webhooks/evolution/connection-update`
- `/api/v1/webhooks/evolution/messages-update`
- `/api/v1/webhooks/evolution/messages-upsert`

O backend aceitava apenas `POST /api/v1/webhooks/evolution`.

Correcao:

- aceitar `POST /api/v1/webhooks/evolution/`
- inferir o tipo de evento pelo path quando necessario
- normalizar `connection.update`, `messages.update` e `messages.upsert`

### 2. `sendPresence` com payload incorreto

O executor enviava:

```json
{
  "number": "...",
  "options": {
    "delay": 1200,
    "presence": "composing"
  }
}
```

No servidor real a Evolution exigiu:

```json
{
  "number": "...",
  "delay": 1200,
  "presence": "composing"
}
```

Correcao aplicada no client e no executor.

### 3. `execution_logs` sem `evolution_message_key`

O runner gravava `messageKey` apenas em `response_payload`.
O sync por webhook procura por `evolution_message_key`.

Correcao:

- runner passou a persistir `evolution_message_key`
- isso permitiu correlacionar webhook real com `execution_logs`

### 4. Score recalculado pelo lado errado da conversa

O sync recalculava score pela `instanceName` do evento recebido.
No trafego real, o evento de mensagem nem sempre representa a instancia que originou o `execution_log`.

Correcao:

- sync agora resolve `phone_number_id` a partir do `message_id` encontrado no `execution_log`
- score passa a ser recalculado para o numero que realmente executou o step

### 5. Enum do banco incompleto

O codigo ja suportava:

- `send_typing`
- `send_recording`

Mas `public.warming_action_type` no Supabase ainda nao tinha esses valores.

Correcao aplicada no projeto `rxdophybnwoocsdyxyjm`:

- `alter type public.warming_action_type add value if not exists 'send_typing'`
- `alter type public.warming_action_type add value if not exists 'send_recording'`

### 6. Timeout da Evolution configuravel

Na validacao real de `send_status`, o endpoint `/message/sendStatus/{instance}` nao retornou resposta HTTP dentro da janela padrao do client.

Correcao aplicada:

- `EVOLUTION_TIMEOUT_SECONDS` adicionado ao carregamento de config da API
- timeout repassado para:
  - criacao de instancias
  - executores do runner

Observacao:

- mesmo com timeout maior, o endpoint real continua sem concluir de forma sincrona em tempo razoavel
- isso diferencia `send_status` das outras actions validadas

## Rotas validadas em pratica

- `POST /api/v1/conversation-scripts`
- `POST /api/v1/warming-jobs`
- `POST /api/v1/warming-jobs/{id}/run-now`
- `POST /api/v1/webhooks/evolution`
- `POST /api/v1/webhooks/evolution/{event}`

## Validacao real executada

### Ambiente

- API local: `http://127.0.0.1:8081`
- webhook publico:
  `https://c40f-2804-a4c-f15f-0-586b-1707-1961-1c4a.ngrok-free.app/api/v1/webhooks/evolution`

### Instancias reais

- `connect_a_20260505`
- `connect_b_20260505`

### Numeros reais corrigidos no banco

- `5519989411105`
- `5519995081355`

### Job real 1

Script com 4 steps:

1. `send_typing` A -> B
2. `send_text` A -> B
3. `send_typing` B -> A
4. `send_text` B -> A

Resultado:

- `run-now` executou `4` steps
- job finalizou com `status = success`
- `execution_logs`:
  - `send_typing` success
  - `send_text` success
  - `send_typing` success
  - `send_text` success
- `evolution_events` reais:
  - `MESSAGES_UPSERT`
  - `MESSAGES_UPDATE`

### Job real 2

Script com 2 steps:

1. `send_text` A -> B
2. `send_text` B -> A

Resultado:

- `run-now` executou `2` steps
- job finalizou com `status = success`
- `execution_logs` gravaram `message_id`
- `evolution_events` reais foram correlacionados por `message_id`
- `warming_score` dos dois numeros foi atualizado no banco

Estado observado apos o segundo job:

- `5519989411105` -> `status = warming`, `warming_score = 2.00`
- `5519995081355` -> `status = warming`, `warming_score = 6.00`

## Conclusao

O caminho abaixo foi provado em ambiente real:

`backend route -> runner -> Evolution real -> webhook real -> Supabase`

Mais especificamente:

- criacao de script pelo backend
- criacao de warming job pelo backend
- disparo de job pela rota `run-now`
- execucao real dos steps
- gravacao de `execution_logs`
- recebimento de webhook real da Evolution
- persistencia de `evolution_events`
- atualizacao de estado e score no Supabase

## O que ainda nao foi validado nesta rodada

- conclusao limpa de `send_status` via resposta HTTP do backend
- scheduler + RabbitMQ externo + worker externo

## Validacoes adicionais concluidas depois desta rodada inicial

- `send_reply` real: validado via `run-now`
- `send_reaction` real: validado via `run-now`
- `send_audio` real: validado via `run-now`
- `send_recording` real: validado via `run-now`
- `send_status` real: validado via `run-now` no modo assíncrono

## Estado real de `send_status`

O endpoint foi validado parcialmente com dados reais:

- o backend monta e envia o payload correto
- a Evolution deixou de responder `400` por `StatusJidList is required`
- o endpoint real permanece pendurado por muito tempo, sem resposta HTTP final utilizavel
- em paralelo, a Evolution emite eventos reais de `status@broadcast`, o que indica processamento assincrono do envio

Conclusao pratica:

- `send_status` esta funcional do ponto de vista de disparo na Evolution
- nao e uma operacao sincrona confiavel na Evolution real
- o backend passou a tratar esse endpoint como aceite assíncrono controlado
- a validacao real final ocorreu com:
  - `run-now` retornando `200`
  - `executedSteps = 1`
  - tempo total ~ `10.6s`
  - `execution_logs.response_payload = {"messageKey": null, "acceptedAsync": true}`
  - `warming_jobs.status = success`

Atualizacao posterior:

- o backend passou a tratar `send_status` como aceite assincrono
- quando o endpoint da Evolution consome a requisicao e fica sem resposta util ate o timeout curto do status, o step e marcado como `success` com `response_payload.acceptedAsync = true`
- detalhes em [48-send-status-async.md](/Volumes/SSDExterno/aquecedor-evolution/backend/docs/48-send-status-async.md:1)

## Validacao real do loop reativo em 2026-05-06

Ambiente usado:

- API local em `:8081`
- `ngrok` publico reapontado via `POST /webhook/set/{instance}`
- `cmd/worker` com RabbitMQ local
- `cmd/scheduler` com RabbitMQ local

Scripts reativos semeados no banco:

- `reactive_short_v1`
- `reactive_medium_v1`
- `reactive_long_v1`

Todos usam `category = reactive` e placeholders:

- `{{phoneA}}`
- `{{phoneB}}`
- `{{triggerMessageId}}`
- `{{triggerRemoteJid}}`

Gatilho real:

- `connect_b_20260505` enviou mensagem real para `connect_a_20260505`
- webhook inbound entrou como `MESSAGES_UPSERT`
- o backend criou automaticamente o `warming_job` reativo `3ed52493-ea23-43bc-9485-2858d15ea892`

Job reativo criado:

- `script_id = b12138db-e6f3-4f13-8dff-10356d548965` (`reactive_short_v1`)
- `status` inicial `pending`
- metadata:
  - `autoReactive = true`
  - `triggerMessageId = 3EB0E17B36A41AD8DD98A1`
  - `phoneAE164 = 5519989411105`
  - `phoneBE164 = 5519995081355`

Para acelerar a validacao, o `scheduled_at` desse job foi ajustado para `now()` apos a criacao.

Execucao observada no worker:

1. `send_reaction` em `connect_a_20260505`
2. `send_typing` em `connect_a_20260505`
3. `send_reply` em `connect_a_20260505`
4. `send_typing` em `connect_b_20260505`
5. `send_text` em `connect_b_20260505`

Resultado:

- `warming_jobs.status = success`
- `started_at` e `finished_at` preenchidos
- `5` `execution_logs` com `status = success`
- nenhum novo job reativo adicional foi criado para o mesmo par dentro da janela imediata, confirmando o bloqueio de recursao
