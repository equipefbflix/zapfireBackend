# Rotas de numeros

Esta feature implementa endpoints HTTP para cadastrar e listar numeros no aquecedor.

## Endpoints

```http
POST /api/v1/phone-numbers
GET /api/v1/phone-numbers
```

## POST /api/v1/phone-numbers

Request:

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

Response `201`:

```json
{
  "id": "uuid",
  "phoneE164": "5511999999999",
  "label": "chip-sp-01",
  "status": "new",
  "warmingScore": 0,
  "metadata": {
    "carrier": "vivo",
    "testRunId": "optional-test-run-id"
  }
}
```

## GET /api/v1/phone-numbers

Response `200`:

```json
{
  "items": []
}
```

## Resultado esperado

- Testes HTTP com fake store.
- Repository com `List`.
- Wiring no `cmd/api`.
- Validacao real via MCP ou integration.

## Resultado em 2026-05-04

Implementado:

- `POST /api/v1/phone-numbers`
- `GET /api/v1/phone-numbers`
- `PhoneNumberRepository.List`
- wiring em `cmd/api` quando `DATABASE_URL` estiver configurado

Testes:

```bash
go test ./...
```

Resultado: passou.

Validacao MCP:

- `public.phone_numbers` no projeto `rxdophybnwoocsdyxyjm` esta com `0` registros apos as validacoes anteriores.

