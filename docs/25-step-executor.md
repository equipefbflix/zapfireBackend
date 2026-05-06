# Executor de steps

Esta feature implementa a execucao isolada de actions de um `conversation_step` contra a Evolution API.

## Actions suportadas neste bloco

- `send_presence`
- `send_text`
- `send_reaction`

## Payloads esperados

### send_presence

```json
{
  "number": "5511999999999",
  "presence": "composing",
  "delay": 1000
}
```

### send_text

```json
{
  "number": "5511999999999",
  "text": "Bom dia, tudo certo?",
  "delay": 1000,
  "linkPreview": false
}
```

### send_reaction

```json
{
  "remoteJid": "5511999999999@s.whatsapp.net",
  "messageId": "message-id",
  "fromMe": true,
  "reaction": "👍"
}
```

## Resultado esperado

- `executor.StepExecutor`.
- Testes unitarios com Evolution fake.
- Sem chamada real Evolution neste bloco; chamadas reais entram quando houver instancias de teste conectadas.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `send_text`, `send_presence`, `send_reaction` e action nao suportada.
2. `go test ./internal/executor` falhou por ausencia de `NewStepExecutor`.
3. `executor.StepExecutor` implementado com conversao de payload para tipos da Evolution.
4. `go test ./...` passou.
5. Sem validacao real Evolution neste bloco, pois depende de instancias WhatsApp conectadas.
