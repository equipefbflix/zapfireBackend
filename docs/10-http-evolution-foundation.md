# Fundacao HTTP e Evolution

Esta feature cobre a base do backend que nao depende do Supabase:

- carregamento de configuracao do servidor;
- carregamento de multiplas Evolution APIs por `.env`;
- rotas de health;
- cliente HTTP da Evolution API v2;
- testes contract com servidor fake;
- teste real opcional contra Evolution de teste.

## Configuracao

Variaveis principais:

```env
APP_ENV=development
APP_PORT=8080
APP_PUBLIC_URL=http://localhost:8080

EVOLUTION_SERVERS=evo1,evo2
EVOLUTION_EVO1_NAME=evo1
EVOLUTION_EVO1_BASE_URL=https://evo1.example.com
EVOLUTION_EVO1_API_KEY=change-me
EVOLUTION_EVO1_ENABLED=true
EVOLUTION_EVO1_WEIGHT=1
EVOLUTION_EVO1_MAX_CONCURRENT_JOBS=5
```

O parser deve aceitar qualquer quantidade de servidores listados em `EVOLUTION_SERVERS`.

## Health

Rotas:

```http
GET /health
GET /api/v1/health
```

Resposta esperada enquanto Supabase nao esta implementado:

```json
{
  "status": "ok",
  "appEnv": "development",
  "evolutionServers": [
    {
      "name": "evo1",
      "baseUrl": "https://evo1.example.com",
      "enabled": true
    }
  ],
  "supabase": {
    "status": "not_configured"
  }
}
```

## Cliente Evolution

Operacoes iniciais:

- `FetchInstances`
- `ConnectionState`
- `ConnectInstance`
- `CreateInstance`
- `DeleteInstance`
- `SendText`
- `SendPresence`
- `SendReaction`

O cliente deve:

- enviar header `apikey`;
- preservar payloads v2;
- usar timeout configuravel;
- retornar erro com status HTTP e corpo quando a Evolution responder falha;
- nao logar API key.

## Testes

Unitarios/contract:

```bash
go test ./...
```

Teste real Evolution:

```bash
ENABLE_REAL_TESTS=true \
EVOLUTION_TEST_BASE_URL='https://evo.example.com' \
EVOLUTION_TEST_API_KEY='secret' \
go test -tags=integration ./internal/evolution -run TestEvolutionFetchInstancesReal -v
```

O teste real apenas consulta instancias. Criacao, conexao e envio de mensagens reais devem entrar em fases posteriores, com dados de teste e cleanup.

## Resultado real em 2026-05-04

- `go test -tags=integration ./internal/evolution -run TestEvolutionFetchInstancesReal -v`: `PASS`
- A Evolution real em `https://evo.askgeni.us` respondeu corretamente ao `FetchInstances`.
- O contrato real de `CreateInstance` observado mais tarde no E2E retornou `hash` como `string`, e o client foi ajustado para aceitar tanto `string` quanto objeto com `apikey`.
