# Evolution API

Base usada agora nesta especificacao: `evolution-go`, publicada em `https://go.zaapfire.com.br/swagger/index.html`.

## Autenticacao

A `evolution-go` usa header:

```http
apikey: <api-key>
```

Mas existem dois contextos de autenticacao:

1. **chave global da Evolution**
   - usada para operacoes administrativas como criar instancia e listar todas as instancias;
2. **token da propria instancia**
   - usado para conectar, consultar status, obter QR e enviar mensagens.

No backend do aquecedor isso ficou assim:

- `CreateInstance` usa a chave global do `evolution_server`;
- o backend gera um token por instancia e persiste em `instances.instance_api_key_secret_name`;
- `connect`, `sync-state` e `send/*` usam esse token da instancia.

## Instancias

### Criar instancia

Endpoint:

```http
POST /instance/create
```

Payload relevante:

```json
{
  "instanceId": "chip_5511999999999",
  "name": "chip_5511999999999",
  "token": "inst_chip_5511999999999_xxx",
  "proxy": {
    "protocol": "http",
    "host": "proxy.example.com",
    "port": "8000",
    "username": "user",
    "password": "pass"
  },
  "advancedSettings": {
    "alwaysOnline": true,
    "ignoreGroups": true,
    "ignoreStatus": false,
    "msgRejectCall": "Nao posso atender agora.",
    "readMessages": true,
    "rejectCall": true
  }
}
```

Resposta esperada:

```json
{
  "message": "success",
  "data": {
    "instanceId": "uuid-gerado",
    "name": "chip_5511999999999",
    "token": "inst_chip_5511999999999_xxx",
    "status": "created"
  }
}
```

### Conectar instancia

Endpoint:

```http
POST /instance/connect
```

Payload operacional minimo:

```json
{
  "webhookUrl": "https://backend.example.com/api/v1/webhooks/evolution",
  "subscribe": [
    "messages.upsert",
    "connection.update"
  ]
}
```

Resposta da conexao:

```json
{
  "message": "success",
  "data": {
    "jid": "5511999999999@s.whatsapp.net",
    "webhookUrl": "https://backend.example.com/api/v1/webhooks/evolution",
    "eventString": "messages.upsert,connection.update"
  }
}
```

Para exibir pareamento no frontend, o backend faz em seguida:

```http
GET /instance/qr
```

e mapeia a resposta para o contrato interno atual:

- `pairingCode` <- `data.qrcode`
- `code` <- `data.code`

### Estado da conexao

Endpoint:

```http
GET /instance/status
```

Resposta tipica:

```json
{
  "message": "success",
  "data": {
    "instanceId": "uuid-da-instancia",
    "name": "chip_5511999999999",
    "status": "open",
    "profileName": "Chip 1"
  }
}
```

Mapeamento interno atual:

- `open` -> `open`
- `close` -> `close`
- vazio + `connected=true` -> `open`
- vazio + `connected=false` -> `close`

### Buscar instancias

Endpoint administrativo:

```http
GET /instance/all
```

Resposta tipica:

```json
{
  "message": "success",
  "data": [
    {
      "id": "abc123",
      "name": "chip_5511999999999",
      "connected": true,
      "jid": "5511999999999@s.whatsapp.net"
    }
  ]
}
```

### Reiniciar instancia

Endpoint operacional:

```http
POST /instance/reconnect
```

No backend atual, `RestartInstance` passou a usar esse endpoint com o token da instancia.

### Deletar instancia

Endpoint administrativo:

```http
DELETE /instance/delete/{instanceId}
```

Esse caminho continua exigindo chave global.

## Mensagens e acoes

### Texto

```http
POST /send/text
```

Payload:

```json
{
  "number": "5511888888888",
  "text": "Oi, tudo bem?",
  "delay": 1200,
  "quoted": {
    "messageId": "MESSAGE_ID",
    "participant": "5511888888888@s.whatsapp.net"
  }
}
```

### Presenca

```http
POST /message/presence
```

Payloads usados pelo aquecedor:

```json
{
  "number": "5511888888888",
  "state": "composing",
  "isAudio": false
}
```

```json
{
  "number": "5511888888888",
  "state": "composing",
  "isAudio": true
}
```

Regra atual no backend:

- `send_typing` -> `state=composing`, `isAudio=false`
- `send_recording` -> `state=composing`, `isAudio=true`

### Reacao

```http
POST /message/react
```

Payload:

```json
{
  "number": "5511888888888@s.whatsapp.net",
  "id": "MESSAGE_ID",
  "fromMe": false,
  "reaction": "👍"
}
```

### Midia e audio

```http
POST /send/media
```

Payload de imagem/video/documento:

```json
{
  "number": "5511888888888",
  "url": "https://example.com/file.png",
  "type": "image",
  "caption": "Bom dia",
  "filename": "file.png",
  "delay": 1000
}
```

Payload de audio:

```json
{
  "number": "5511888888888",
  "url": "https://example.com/audio.ogg",
  "type": "audio",
  "delay": 1500
}
```

### Sticker

```http
POST /send/sticker
```

### Status

Texto:

```http
POST /send/status/text
```

Payload:

```json
{
  "text": "Bom dia"
}
```

Midia:

```http
POST /send/status/media
```

Payload:

```json
{
  "type": "image",
  "url": "https://example.com/status.png",
  "caption": "Legenda"
}
```

Observacao importante:

- `POST /message/status` na `evolution-go` e consulta de status de mensagem, nao envio de story/status.

## Webhooks

Eventos minimos para habilitar no loop reativo:

- `messages.upsert`
- `connection.update`

O backend continua normalizando esses eventos para o contrato interno do aquecedor antes de passar por `evolutionsync`, `conversationloop`, `scheduler`, `queue` e `worker`.
