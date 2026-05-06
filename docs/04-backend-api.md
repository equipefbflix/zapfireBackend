# Contrato REST do backend

Prefixo recomendado:

```http
/api/v1
```

## Autenticacao

Quando `API_AUTH_ENABLED=true`, todas as rotas `/api/v1/*` exigem:

```http
Authorization: Bearer <supabase_access_token>
```

Excecoes:

- `GET /health`
- `GET /api/v1/health`
- `POST /api/v1/webhooks/evolution`
- `POST /api/v1/webhooks/evolution/*`

## Health

```http
GET /health
GET /api/v1/health
GET /api/v1/metrics
```

Retorna status do backend, Supabase e Evolution APIs configuradas.

`GET /api/v1/metrics` expõe contadores operacionais basicos.

## Evolution servers

```http
GET /api/v1/evolution-servers
GET /api/v1/evolution-servers/{id}
POST /api/v1/evolution-servers/{id}/health-check
```

Na primeira versao, servidores serao definidos por `.env`; essas rotas apenas leem status persistido e executam checks.

## Proxies

```http
GET /api/v1/proxies
POST /api/v1/proxies
PATCH /api/v1/proxies/{id}
POST /api/v1/proxies/{id}/health-check
```

Campos principais:

```json
{
  "name": "proxy-01",
  "host": "proxy.example.com",
  "port": 8000,
  "protocol": "http",
  "username": "user",
  "passwordSecretName": "PROXY_01_PASSWORD",
  "enabled": true,
  "maxInstances": 20
}
```

## Numeros

```http
GET /api/v1/phone-numbers
POST /api/v1/phone-numbers
GET /api/v1/phone-numbers/{id}
PATCH /api/v1/phone-numbers/{id}
POST /api/v1/phone-numbers/{id}/pause
POST /api/v1/phone-numbers/{id}/resume
```

Criacao:

```json
{
  "phoneE164": "5511999999999",
  "label": "chip-sp-01",
  "testRunId": "optional-test-run-id",
  "metadata": {
    "carrier": "vivo"
  }
}
```

## Instancias

```http
GET /api/v1/instances
POST /api/v1/instances
GET /api/v1/instances/{id}
POST /api/v1/instances/{id}/connect
POST /api/v1/instances/{id}/restart
POST /api/v1/instances/{id}/sync-state
POST /api/v1/instances/{id}/settings
```

Criacao:

```json
{
  "phoneNumberId": "uuid",
  "evolutionServerId": "uuid opcional",
  "proxyId": "uuid opcional",
  "qrcode": true,
  "settings": {
    "rejectCall": true,
    "msgCall": "Nao posso atender agora.",
    "groupsIgnore": true,
    "alwaysOnline": true,
    "readMessages": true,
    "readStatus": true,
    "syncFullHistory": false
  }
}
```

Se `evolutionServerId` nao for informado, o backend escolhe uma Evolution saudavel por peso e capacidade. Se `proxyId` nao for informado, o backend escolhe um proxy disponivel ou cria sem proxy conforme politica.

## Templates e scripts

```http
GET /api/v1/message-templates
POST /api/v1/message-templates
PATCH /api/v1/message-templates/{id}

GET /api/v1/conversation-scripts
POST /api/v1/conversation-scripts
GET /api/v1/conversation-scripts/{id}
PATCH /api/v1/conversation-scripts/{id}
POST /api/v1/conversation-scripts/{id}/steps
PATCH /api/v1/conversation-steps/{id}
DELETE /api/v1/conversation-steps/{id}
```

Exemplo de script:

```json
{
  "name": "conversa_basica_manha",
  "category": "casual",
  "weight": 10,
  "steps": [
    {
      "stepOrder": 1,
      "senderRole": "a",
      "actionType": "send_presence",
      "payload": { "presence": "composing" },
      "minDelaySeconds": 1,
      "maxDelaySeconds": 3
    },
    {
      "stepOrder": 2,
      "senderRole": "a",
      "actionType": "send_text",
      "templateId": "uuid",
      "minDelaySeconds": 5,
      "maxDelaySeconds": 30
    },
    {
      "stepOrder": 3,
      "senderRole": "b",
      "actionType": "send_reply",
      "templateId": "uuid",
      "minDelaySeconds": 20,
      "maxDelaySeconds": 180
    }
  ]
}
```

## Jobs de aquecimento

```http
GET /api/v1/warming-jobs
POST /api/v1/warming-jobs
GET /api/v1/warming-jobs/{id}
POST /api/v1/warming-jobs/{id}/cancel
POST /api/v1/warming-jobs/{id}/run-now
POST /api/v1/warming-jobs/stale-cleanup
```

Criacao manual:

```json
{
  "phoneAId": "uuid",
  "phoneBId": "uuid",
  "scriptId": "uuid",
  "scheduledAt": "2026-04-29T14:00:00Z"
}
```

## Execucoes e logs

```http
GET /api/v1/execution-logs
GET /api/v1/execution-logs/{id}
```

Filtros:

- `warmingJobId`
- `phoneNumberId`
- `instanceId`
- `status`
- `from`
- `to`

## Webhook Evolution

```http
POST /api/v1/webhooks/evolution
```

Headers:

```http
X-Webhook-Secret: <secret>
```

O handler deve:

1. validar segredo;
2. persistir payload bruto em `evolution_events`;
3. responder rapido;
4. processar evento em worker assíncrono ou cron curto.
