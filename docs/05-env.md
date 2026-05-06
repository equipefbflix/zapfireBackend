# Variaveis de ambiente

## Servidor

```env
APP_ENV=development
APP_PORT=8080
APP_PUBLIC_URL=http://localhost:8080
EVOLUTION_TIMEOUT_SECONDS=30
OBSERVABILITY_LOOKBACK_MINUTES=60
WARMING_STALE_RUNNING_MINUTES=20
WARMING_STALE_CLEANUP_REASON=stale running job cleanup
API_AUTH_ENABLED=true
WEBHOOK_EVOLUTION_SECRET=change-me
```

## Supabase

```env
SUPABASE_PROJECT_REF=rxdophybnwoocsdyxyjm
SUPABASE_URL=https://rxdophybnwoocsdyxyjm.supabase.co
SUPABASE_SERVICE_ROLE_KEY=change-me
DATABASE_URL=postgres://postgres:password@db.rxdophybnwoocsdyxyjm.supabase.co:5432/postgres?sslmode=require
```

O backend Go deve preferir `DATABASE_URL` para operacoes relacionais e pode usar `SUPABASE_URL` + service role para APIs auxiliares.

Quando `API_AUTH_ENABLED=true`, o backend valida `Authorization: Bearer <access_token>` usando o JWKS remoto em `SUPABASE_URL/auth/v1/.well-known/jwks.json`.

## Evolution APIs

Quantidade configuravel:

```env
EVOLUTION_SERVERS=evo1,evo2,evo3,evo4,evo5

EVOLUTION_EVO1_NAME=evo1
EVOLUTION_EVO1_BASE_URL=https://evo1.example.com
EVOLUTION_EVO1_API_KEY=change-me
EVOLUTION_EVO1_ENABLED=true
EVOLUTION_EVO1_WEIGHT=1
EVOLUTION_EVO1_MAX_CONCURRENT_JOBS=5

EVOLUTION_EVO2_NAME=evo2
EVOLUTION_EVO2_BASE_URL=https://evo2.example.com
EVOLUTION_EVO2_API_KEY=change-me
EVOLUTION_EVO2_ENABLED=true
EVOLUTION_EVO2_WEIGHT=1
EVOLUTION_EVO2_MAX_CONCURRENT_JOBS=5
```

O parser deve aceitar qualquer quantidade listada em `EVOLUTION_SERVERS`.

Tambem e aceito o formato simples usado pela Evolution em docker/teste:

```env
SERVER_URL=https://evo.askgeni.us
AUTHENTICATION_API_KEY=change-me
```

Quando `EVOLUTION_SERVERS` nao estiver definido, o backend cria uma configuracao interna chamada `default` usando essas duas variaveis.

## Proxies

Proxies podem vir do banco ou do `.env`. Para MVP, usar `.env` como fonte de segredo e banco como fonte operacional.

```env
PROXY_DEFAULT_ENABLED=true
PROXY_ASSIGNMENT_STRATEGY=least_used

PROXY_01_PASSWORD=change-me
PROXY_02_PASSWORD=change-me
```

## Regras de aquecimento

```env
WARMING_ENABLED=true
WARMING_TICK_SECONDS=30
WARMING_JOB_LOOKAHEAD_MINUTES=30
WARMING_MAX_JOBS_PER_TICK=20

WARMING_MIN_DELAY_SECONDS=20
WARMING_MAX_DELAY_SECONDS=240
WARMING_PAIR_COOLDOWN_MINUTES=30
WARMING_INBOUND_TRIGGER_COOLDOWN_SECONDS=90
WARMING_WINDOW_START_HOUR=8
WARMING_WINDOW_END_HOUR=22
WARMING_MAX_RUNNING_JOBS_PER_PAIR=1
WARMING_MAX_RUNNING_JOBS_PER_EVOLUTION_SERVER=5
WARMING_MAX_DAILY_MESSAGES_PER_NUMBER=30
WARMING_MAX_PAIR_DAILY_MESSAGES=8
WARMING_MIN_SCORE_TO_MARK_WARM=80

WARMING_SCORE_MESSAGE_SUCCESS=1.5
WARMING_SCORE_REPLY_SUCCESS=2.0
WARMING_SCORE_REACTION_SUCCESS=0.5
WARMING_SCORE_DAILY_ACTIVE_BONUS=3.0
WARMING_SCORE_FAILURE_PENALTY=2.0
WARMING_SCORE_DISCONNECTED_PENALTY=5.0
```

## Politicas de recuperacao

```env
INSTANCE_HEALTH_CHECK_SECONDS=60
INSTANCE_RESTART_AFTER_FAILURES=3
INSTANCE_MAX_RESTARTS_PER_DAY=3
EVOLUTION_HEALTH_CHECK_SECONDS=30
EVOLUTION_MARK_DOWN_AFTER_FAILURES=3
```

## HTTP client

```env
HTTP_TIMEOUT_SECONDS=30
HTTP_RETRY_MAX=3
HTTP_RETRY_BASE_DELAY_MS=300
```

## RabbitMQ

Nao gravar credenciais reais em arquivos versionaveis. Para desenvolvimento, usar `.env.local` ou variaveis exportadas no shell.

```env
RABBITMQ_URL=amqp://user:password@rabbitmq.example.com:5672/
RABBITMQ_EXCHANGE=aquecedor.events
RABBITMQ_QUEUE_WARMING_JOBS=aquecedor.warming.jobs
RABBITMQ_QUEUE_EVOLUTION_EVENTS=aquecedor.evolution.events
RABBITMQ_QUEUE_DEAD_LETTER=aquecedor.dead_letter
RABBITMQ_PREFETCH=10
RABBITMQ_MAX_RETRIES=3
RABBITMQ_PUBLISH_TIMEOUT_SECONDS=5
RABBITMQ_CONSUMER_ENABLED=true
SCHEDULER_ENABLED=true
SCHEDULER_TICK_SECONDS=5
```

O backend deve aceitar `RABBITMQ_URL` completo para evitar expor usuario/senha em logs ou documentacao.

## Observacao de runtime

`EVOLUTION_TIMEOUT_SECONDS` controla o timeout HTTP usado pelo backend para chamadas na Evolution. O valor padrao continua `30`, mas `sendStatus` no servidor real mostrou comportamento assíncrono e pode exigir janela maior para diagnóstico operacional.
