# Action send_reply

Esta feature adiciona suporte ao step `send_reply`.

## Payload esperado

```json
{
  "number": "5511999999999",
  "text": "Respondi aqui",
  "quoted": {
    "remoteJid": "5511999999999@s.whatsapp.net",
    "messageId": "message-id",
    "fromMe": false
  },
  "delay": 1000,
  "linkPreview": false
}
```

## Implementacao

Usar o endpoint `sendText` da Evolution com o campo `quoted`.

## Resultado esperado

- `executor.StepExecutor` suporta `send_reply`.
- Teste unitario valida montagem do `QuotedInfo`.

## Validacao realizada

Fluxo executado:

1. Teste criado antes da implementacao para `send_reply`.
2. `go test ./internal/executor` falhou com action nao suportada.
3. Implementacao adicionada usando `SendTextRequest.Quoted`.
4. `go test ./...` passou.
