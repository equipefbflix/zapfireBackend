# Actions send_media e send_sticker

Esta feature adiciona suporte inicial a `send_media` e `send_sticker`.

## send_media

Payload esperado:

```json
{
  "number": "5511999999999",
  "mediatype": "image",
  "mimetype": "image/png",
  "caption": "Bom dia",
  "media": "https://example.com/file.png",
  "fileName": "file.png",
  "delay": 1000
}
```

Endpoint Evolution:

```http
POST /message/sendMedia/{instance}
```

## send_sticker

Payload esperado:

```json
{
  "number": "5511999999999",
  "sticker": "https://example.com/sticker.webp",
  "delay": 1000
}
```

Endpoint Evolution:

```http
POST /message/sendSticker/{instance}
```

## Resultado esperado

- Client Evolution com `SendMedia` e `SendSticker`.
- `executor.StepExecutor` suportando as duas actions.
- Testes unitarios para client e executor.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `Client.SendMedia`, `Client.SendSticker`, `send_media` e `send_sticker` no executor.
2. `go test ./internal/evolution ./internal/executor` falhou por ausencia de tipos e metodos.
3. Client Evolution e executor foram implementados.
4. `go test ./...` passou.
5. Sem validacao real Evolution neste bloco; ainda dependemos de instancias WhatsApp conectadas para E2E externo.
