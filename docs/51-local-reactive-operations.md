# Operacao local do loop reativo

Este documento descreve o menor caminho para operar localmente:

- API
- RabbitMQ local
- worker
- scheduler
- webhook publico via ngrok
- scripts reativos no banco
- mensagem semente para disparar o loop

## Pre-requisitos

- RabbitMQ local ativo
- duas instancias reais conectadas na Evolution Go
- `DATABASE_URL`
- `SERVER_URL`
- `AUTHENTICATION_API_KEY`

## 1. Subir a API

```bash
cd /Volumes/SSDExterno/aquecedor-evolution/backend
export DATABASE_URL='postgresql://postgres:***@db.cqmxcsmpdshuncupcwaw.supabase.co:5432/postgres'
export SERVER_URL='https://go.zaapfire.com.br'
export AUTHENTICATION_API_KEY='***'
export API_AUTH_ENABLED='true'
export SUPABASE_URL='https://cqmxcsmpdshuncupcwaw.supabase.co'
./scripts/run-api.sh
```

Padrao local:

- `APP_PORT=8081`
- `APP_PUBLIC_URL=http://127.0.0.1:8081`

## 2. Abrir o webhook publico

```bash
ngrok http 8081
```

Copiar a URL publica e usar:

`https://<subdominio>.ngrok-free.app/api/v1/webhooks/evolution`

## 3. Garantir as fixtures reativas de QA

```bash
export REACTIVE_FIXTURE_A_NAME='reactive_a_go_20260515'
export REACTIVE_FIXTURE_B_NAME='reactive_b_go_20260515'
export REACTIVE_FIXTURE_A_PHONE='5519989411105'
export REACTIVE_FIXTURE_B_PHONE='5519995081355'
```

Essas sao as fixtures QA atuais usadas pelo E2E real do loop reativo.

Elas precisam existir:

- na `evolution-go`
- em `public.phone_numbers`
- em `public.instances`

## 4. Parear as duas fixtures

Use o fluxo de pareamento ja existente no frontend ou a rota de `connect` do backend para obter o QR/pairing code das duas instancias:

- `reactive_a_go_20260515`
- `reactive_b_go_20260515`

O E2E real do loop reativo exige que ambas estejam:

- `state = open`
- `loggedIn = true`
- `jid != ''`

## 5. Subir worker e scheduler

```bash
export RABBITMQ_URL='amqp://guest:guest@127.0.0.1:5672/'
./scripts/run-worker.sh
```

Em outro terminal:

```bash
export RABBITMQ_URL='amqp://guest:guest@127.0.0.1:5672/'
./scripts/run-scheduler.sh
```

## 6. Semear scripts reativos

```bash
./scripts/seed-reactive-scripts.sh
```

Isso cria:

- `reactive_short_v1`
- `reactive_medium_v1`
- `reactive_long_v1`

Se a API estiver com auth habilitada:

```bash
export API_BEARER_TOKEN='<supabase_access_token>'
./scripts/seed-reactive-scripts.sh
```

## 7. Disparar uma mensagem inbound de prova

```bash
./scripts/send-inbound-probe.sh reactive_b_go_20260515 5519989411105
```

## 8. O que deve acontecer

1. Evolution envia `MESSAGES_UPSERT`
2. API responde `202`
3. backend cria `warming_job` com `metadata.autoReactive = true`
4. scheduler publica o job devido na fila
5. worker consome
6. runner executa os steps do script reativo
7. `execution_logs` e `warming_jobs` refletem a execucao

## 9. Consultas uteis

Job reativo mais recente:

```sql
select id, status, scheduled_at, started_at, finished_at, metadata::text
from public.warming_jobs
where metadata->>'autoReactive' = 'true'
order by updated_at desc
limit 5;
```

Logs do job:

```sql
select id, status, action_type, request_payload::text, response_payload::text, error
from public.execution_logs
where warming_job_id = '<job-id>'
order by created_at asc;
```

## 10. Observacoes

- o planner calcula `scheduled_at` automaticamente; para depuracao rapida, voce pode adiantar o job para `now()`
- o loop atual responde apenas a `MESSAGES_UPSERT`
- o cooldown de inbound atual vem de `WARMING_INBOUND_TRIGGER_COOLDOWN_SECONDS`

## 11. Validacao minima de runtime local

Antes de abrir ngrok e testar o loop inteiro, valide a malha local:

```bash
ENABLE_REAL_TESTS=true \
RABBITMQ_URL='amqp://guest:guest@127.0.0.1:5672/' \
go test -tags=integration ./internal/workerapp -run TestRabbitMQLocalRealFlow -v
```

Em 2026-05-13, esse teste passou com:

- RabbitMQ local real em `127.0.0.1:5672`
- banco real do projeto `rxdophybnwoocsdyxyjm`
- `execution_logs` persistidos
- cleanup completo ao final

## 12. E2E real do webhook reativo

Existe tambem um teste real cobrindo:

`webhook -> evolutionsync -> conversationloop -> warming_job -> queue -> worker -> runner -> execution_logs`

Execucao:

```bash
ENABLE_REAL_TESTS=true \
RABBITMQ_URL='amqp://guest:guest@127.0.0.1:5672/' \
go test -tags=integration ./internal/httpserver -run TestReactiveLoopRealE2E -v
```

Esse teste:

- usa a rota real `POST /api/v1/webhooks/evolution` via `httptest`
- persiste `evolution_events` no banco real
- propaga `testRunId` do payload do webhook para o `warming_job`
- exige que as fixtures `reactive_a_go_20260515` e `reactive_b_go_20260515` existam, estejam persistidas no banco, estejam `open` e com sessao real ativa na `evolution-go`
- publica em topologia RabbitMQ isolada por `testRunId`
- consome no worker real
- executa o runner real
- verifica `execution_logs` de sucesso
- limpa `execution_logs`, `warming_jobs` e `evolution_events` ao final
