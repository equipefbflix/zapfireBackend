# Evolution API

Base usada nesta especificacao: documentacao publica da Evolution API v2.

## Autenticacao

A Evolution API usa header:

```http
apikey: <api-key>
```

Cada Evolution API configurada no `.env` tera:

- `name`
- `base_url`
- `api_key`
- `weight`
- `enabled`
- limites locais de concorrencia e rate.

## Instancias

### Criar instancia

Endpoint:

```http
POST /instance/create
```

Payload v2 relevante:

```json
{
  "instanceName": "chip_5511999999999",
  "integration": "WHATSAPP-BAILEYS",
  "token": "",
  "qrcode": true,
  "number": "5511999999999",
  "rejectCall": true,
  "msgCall": "Nao posso atender agora.",
  "groupsIgnore": true,
  "alwaysOnline": true,
  "readMessages": true,
  "readStatus": true,
  "syncFullHistory": false,
  "proxyHost": "proxy.example.com",
  "proxyPort": "8000",
  "proxyProtocol": "http",
  "proxyUsername": "user",
  "proxyPassword": "pass",
  "webhook": {
    "url": "https://backend.example.com/webhooks/evolution",
    "byEvents": true,
    "base64": false,
    "events": [
      "MESSAGES_UPSERT",
      "MESSAGES_UPDATE",
      "CONNECTION_UPDATE",
      "QRCODE_UPDATED"
    ]
  }
}
```

Notas:

- Em v2, proxy e enviado como campos planos `proxyHost`, `proxyPort`, `proxyProtocol`, `proxyUsername`, `proxyPassword`.
- Em v1, proxy aparece como objeto `proxy`. O cliente Go deve isolar essa diferenca em um adaptador, caso seja necessario suportar v1.
- `token` pode ficar vazio para a Evolution gerar uma chave por instancia, mas o backend deve persistir a chave retornada em `instances.instance_api_key`.

### Conectar instancia

Endpoint:

```http
GET /instance/connect/{instance}?number=5511999999999
```

Retorna `pairingCode`, `code` e `count`. O backend deve persistir tentativas de conexao e expor esse retorno para o painel/cliente.

### Estado da conexao

Endpoint:

```http
GET /instance/connectionState/{instance}
```

Estados esperados incluem `open`, `close` e estados intermediarios retornados pela Evolution.

### Buscar instancias

Endpoint:

```http
GET /instance/fetchInstances
GET /instance/fetchInstances?instanceName=<name>
```

Usado pelos crons de reconciliacao.

### Reiniciar instancia

Endpoint:

```http
PUT /instance/restart/{instance}
```

Usado apenas por politica de recuperacao, com limite de tentativas.

### Configuracoes

```http
POST /settings/set/{instance}
GET /settings/find/{instance}
```

Configuracoes iniciais recomendadas para aquecimento:

- `rejectCall=true`
- `groupsIgnore=true`
- `alwaysOnline=true`
- `readMessages=true`
- `readStatus=true`
- `syncFullHistory=false`

## Mensagens e acoes

### Texto

```http
POST /message/sendText/{instance}
```

Payload v2:

```json
{
  "number": "5511888888888",
  "text": "Oi, tudo bem?",
  "delay": 1200,
  "linkPreview": false,
  "quoted": {
    "key": {
      "id": "MESSAGE_ID"
    },
    "message": {
      "conversation": "Mensagem anterior"
    }
  }
}
```

### Presenca digitando

```http
POST /chat/sendPresence/{instance}
```

Usar antes de texto para simular conversa mais natural:

```json
{
  "number": "5511888888888",
  "options": {
    "delay": 1500,
    "presence": "composing"
  }
}
```

### Reacao

```http
POST /message/sendReaction/{instance}
```

Exige chave da mensagem anterior:

```json
{
  "key": {
    "remoteJid": "5511888888888@s.whatsapp.net",
    "fromMe": false,
    "id": "MESSAGE_ID"
  },
  "reaction": "👍"
}
```

### Resposta citada

Resposta citada e uma variacao do envio de texto com `quoted`. O backend deve guardar `remoteJid`, `fromMe` e `id` retornados pela Evolution para permitir respostas e reacoes futuras.

### Midia

Endpoints esperados na Evolution v2:

- `POST /message/sendMedia/{instance}`
- `POST /message/sendWhatsAppAudio/{instance}`
- `POST /message/sendSticker/{instance}`
- `POST /message/sendLocation/{instance}`
- `POST /message/sendContact/{instance}`
- `POST /message/sendPoll/{instance}`
- `POST /message/sendStatus/{instance}`

Nem todos devem entrar na primeira versao. Prioridade:

1. texto
2. presenca digitando
3. resposta citada
4. reacao
5. sticker
6. imagem/audio
7. status
8. contato/localizacao/poll

### Typing e recording

Os modos de digitacao e gravacao usam o mesmo endpoint de presenca:

```http
POST /chat/sendPresence/{instance}
```

Payloads:

```json
{
  "number": "5511888888888",
  "options": {
    "delay": 1200,
    "presence": "composing"
  }
}
```

```json
{
  "number": "5511888888888",
  "options": {
    "delay": 900,
    "presence": "recording"
  }
}
```

### Ligacoes

A documentacao publica evidencia configuracao para rejeitar chamadas (`rejectCall` e `msgCall`). Nao considerar "fazer ligacao" como capacidade garantida ate confirmar endpoint oficial da versao instalada.

## Webhooks

Eventos minimos para habilitar:

- `CONNECTION_UPDATE`
- `QRCODE_UPDATED`
- `MESSAGES_UPSERT`
- `MESSAGES_UPDATE`
- `SEND_MESSAGE`

O endpoint do backend:

```http
POST /webhooks/evolution
```

Deve validar assinatura/header compartilhado definido em `.env`, normalizar evento e persistir em `evolution_events`.

## Redundancia

Cada instancia pertence a uma Evolution API por vez:

- `instances.evolution_server_id` define a API ativa.
- `evolution_servers.health_status` define se a API pode receber novas execucoes.
- Se uma API cair, o backend pausa execucoes das instancias afetadas.
- Recriacao/migracao automatica de instancia entre Evolution APIs deve ser uma politica explicita, pois pode exigir novo QR/pairing.
