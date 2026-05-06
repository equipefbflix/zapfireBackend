# Supabase Auth no backend

O frontend pode continuar autenticando com Supabase Auth. O backend agora aceita o `access_token` do frontend e valida o JWT localmente antes de permitir acesso às rotas protegidas.

## Rotas protegidas

Tudo sob `/api/v1/*` exige:

```http
Authorization: Bearer <supabase_access_token>
```

Excecoes:

- `GET /health`
- `GET /api/v1/health`
- `POST /api/v1/webhooks/evolution`
- `POST /api/v1/webhooks/evolution/*`

## Variaveis de ambiente

```env
API_AUTH_ENABLED=true
SUPABASE_URL=https://rxdophybnwoocsdyxyjm.supabase.co
```

O backend deriva:

- `issuer = SUPABASE_URL/auth/v1`
- `jwks = SUPABASE_URL/auth/v1/.well-known/jwks.json`

## Validacao

O middleware exige:

- header `Authorization: Bearer ...`
- assinatura JWT valida por JWKS
- `iss` igual a `SUPABASE_URL/auth/v1`
- `aud` contendo `authenticated`
- `sub` presente

## Fluxo frontend

1. o frontend faz login no Supabase Auth
2. recebe `session.access_token`
3. envia esse token para o backend em `Authorization: Bearer ...`
4. o backend valida o JWT e injeta o usuario autenticado no contexto da request

## Observacoes operacionais

- se `API_AUTH_ENABLED=true` e `SUPABASE_URL` estiver ausente, a API nao sobe
- o webhook da Evolution continua fora desse fluxo e usa `WEBHOOK_EVOLUTION_SECRET`
- o backend ainda opera com `DATABASE_URL` server-side; o JWT do frontend protege a API HTTP, nao o acesso interno ao banco
