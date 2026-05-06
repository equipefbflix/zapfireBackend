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
- duas instancias reais conectadas na Evolution
- `DATABASE_URL`
- `SERVER_URL`
- `AUTHENTICATION_API_KEY`

## 1. Subir a API

```bash
cd /Volumes/SSDExterno/aquecedor-evolution/backend
export DATABASE_URL='postgresql://postgres:***@db.rxdophybnwoocsdyxyjm.supabase.co:5432/postgres'
export SERVER_URL='https://evo.askgeni.us'
export AUTHENTICATION_API_KEY='***'
export API_AUTH_ENABLED='true'
export SUPABASE_URL='https://rxdophybnwoocsdyxyjm.supabase.co'
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

## 3. Reapontar o webhook das instancias

```bash
NGROK_URL='https://<subdominio>.ngrok-free.app/api/v1/webhooks/evolution'

for inst in connect_a_20260505 connect_b_20260505; do
  curl -X POST "https://evo.askgeni.us/webhook/set/$inst" \
    -H 'Content-Type: application/json' \
    -H "apikey: ${AUTHENTICATION_API_KEY}" \
    --data "{
      \"webhook\": {
        \"enabled\": true,
        \"url\": \"${NGROK_URL}\",
        \"events\": [\"MESSAGES_UPSERT\", \"MESSAGES_UPDATE\", \"CONNECTION_UPDATE\"],
        \"webhook_by_events\": true,
        \"webhook_base64\": false
      }
    }"
done
```

## 4. Subir worker e scheduler

```bash
export RABBITMQ_URL='amqp://guest:guest@127.0.0.1:5672/'
./scripts/run-worker.sh
```

Em outro terminal:

```bash
export RABBITMQ_URL='amqp://guest:guest@127.0.0.1:5672/'
./scripts/run-scheduler.sh
```

## 5. Semear scripts reativos

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

## 6. Disparar uma mensagem inbound de prova

```bash
./scripts/send-inbound-probe.sh connect_b_20260505 5519989411105
```

## 7. O que deve acontecer

1. Evolution envia `MESSAGES_UPSERT`
2. API responde `202`
3. backend cria `warming_job` com `metadata.autoReactive = true`
4. scheduler publica o job devido na fila
5. worker consome
6. runner executa os steps do script reativo
7. `execution_logs` e `warming_jobs` refletem a execucao

## 8. Consultas uteis

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

## 9. Observacoes

- o planner calcula `scheduled_at` automaticamente; para depuracao rapida, voce pode adiantar o job para `now()`
- o loop atual responde apenas a `MESSAGES_UPSERT`
- o cooldown de inbound atual vem de `WARMING_INBOUND_TRIGGER_COOLDOWN_SECONDS`
