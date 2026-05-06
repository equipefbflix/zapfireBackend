# Actions de audio, status, typing e recording

Esta feature expande o executor de steps com quatro actions novas:

- `send_audio`
- `send_status`
- `send_typing`
- `send_recording`

## Escopo

- Client Evolution com:
  - `POST /message/sendWhatsAppAudio/{instance}`
  - `POST /message/sendStatus/{instance}`
- Executor com:
  - `send_audio`
  - `send_status`
  - `send_typing`
  - `send_recording`

## TDD aplicado

1. Testes adicionados em `internal/evolution/client_test.go`:
   - `TestClientSendWhatsAppAudio`
   - `TestClientSendStatus`
2. Testes adicionados em `internal/executor/step_executor_test.go`:
   - `TestStepExecutorSendAudio`
   - `TestStepExecutorSendStatus`
   - `TestStepExecutorSendTyping`
   - `TestStepExecutorSendRecording`
3. `go test ./internal/evolution ./internal/executor` falhou pelos contratos ausentes.
4. Implementados novos requests e metodos no client.
5. Implementadas novas actions no executor.
6. `go test ./internal/evolution ./internal/executor` passou.

## Decisoes

### `send_typing`

`send_typing` e um alias de `sendPresence` com:

```json
{
  "presence": "composing"
}
```

### `send_recording`

`send_recording` e um alias de `sendPresence` com:

```json
{
  "presence": "recording"
}
```

### `send_audio`

Usa payload:

```json
{
  "number": "5511999999999",
  "audio": "https://example.com/audio.ogg",
  "delay": 1200
}
```

### `send_status`

Implementado como status/story do WhatsApp via Evolution. Payload atual:

```json
{
  "type": "text",
  "content": "Bom dia",
  "backgroundColor": "#112233",
  "font": 2,
  "media": "",
  "caption": "",
  "delay": 700,
  "linkPreview": false
}
```

## Observacao

Ainda nao houve validacao real dessas actions porque isso exige uma instancia conectada de teste. Nesta fase, o contrato ficou coberto por testes unitarios do client/executor.
