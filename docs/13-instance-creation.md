# Criacao de instancia

Esta feature implementa a camada de negocio para criar uma instancia WhatsApp na Evolution API com suporte a proxy e persistencia em Supabase.

## Escopo desta etapa

- Criar `InstanceRepository`.
- Criar `InstanceService`.
- Selecionar um `evolution_server` habilitado.
- Selecionar um proxy habilitado quando existir.
- Chamar Evolution API `POST /instance/create`.
- Persistir a instancia em `public.instances`.
- Manter teste real de banco com cleanup por `metadata.testRunId`.

## Fora desta etapa

- Conectar QR/pairing.
- Enviar mensagens reais.

Esses pontos entram nas proximas features para manter o ciclo TDD pequeno. O E2E real de criacao de instancia foi fechado depois em `36-real-instance-e2e.md`.

## Contrato do service

Entrada:

- `phoneNumberID`
- `phoneE164`
- `instanceName`
- `testRunID`
- configuracoes Evolution opcionais

Fluxo:

1. Buscar servidores Evolution habilitados no Supabase.
2. Escolher o primeiro servidor habilitado por enquanto.
3. Buscar proxies habilitados.
4. Escolher o proxy menos usado quando disponivel.
5. Criar a instancia na Evolution API com `proxyHost`, `proxyPort`, `proxyProtocol`, `proxyUsername`, `proxyPassword` quando proxy existir.
6. Persistir em `public.instances`.

## Contrato HTTP

Endpoint:

```http
POST /api/v1/instances
```

Request:

```json
{
  "phoneNumberId": "uuid",
  "phoneE164": "5511999999999",
  "instanceName": "chip_5511999999999",
  "testRunId": "optional-test-run-id"
}
```

Response `201`:

```json
{
  "id": "uuid",
  "phoneNumberId": "uuid",
  "evolutionServerId": "uuid",
  "proxyId": "uuid opcional",
  "instanceName": "chip_5511999999999",
  "status": "created"
}
```

Erros:

- `400` para JSON invalido ou campos obrigatorios ausentes.
- `409` para ausencia de Evolution server habilitado.
- `500` para erro inesperado do service.

## Testes

Unitarios:

- service monta payload correto com proxy;
- service persiste instancia apos sucesso da Evolution;
- service falha se nao houver servidor Evolution habilitado;
- repository insere e limpa instancia por `testRunId`.

Integration:

```bash
ENABLE_REAL_TESTS=true \
DATABASE_URL='postgres://...' \
go test -tags=integration ./internal/repository -run TestInstanceRepositoryRealDatabase -v
```

Validacao real via MCP:

- inserir telefone, servidor, proxy e instancia;
- limpar em ordem segura;
- confirmar zero sobras.

## Resultado em 2026-05-04

Implementado:

- `InstanceRepository`
- `InstanceService`
- `POST /api/v1/instances`
- wiring em `cmd/api` quando `DATABASE_URL` estiver configurado;
- `EvolutionClientFactory` e `EnvSecretResolver`;
- selecao inicial do primeiro Evolution server habilitado;
- selecao inicial do primeiro proxy habilitado;
- montagem do payload Evolution com proxy;
- persistencia em `public.instances`.

Testes locais:

```bash
go test ./...
```

Resultado: passou.

Validacao real via MCP global no projeto `rxdophybnwoocsdyxyjm`:

- inseriu telefone, servidor, proxy e instancia com `metadata.testRunId = codex-instance-validation`;
- removeu instancia primeiro, depois telefone, servidor e proxy;
- verificacao final retornou zero sobras.

Observacao:

- Os testes unitarios usam fakes para a chamada Evolution.
- Em ambiente real, `cmd/api` monta repositories e service quando `DATABASE_URL` estiver presente.
- A rota real exige que existam registros habilitados em `public.evolution_servers` e, opcionalmente, `public.proxies`.
- O teste E2E real de criacao na Evolution foi fechado em `36-real-instance-e2e.md`, com cleanup confirmado na Evolution instalada.

## Configuracao real atual

Em 2026-05-04, foi cadastrado no Supabase correto `rxdophybnwoocsdyxyjm`:

- `public.evolution_servers.name = default`
- `base_url = https://evo.askgeni.us`
- `api_key_secret_name = AUTHENTICATION_API_KEY`
- `enabled = true`

A chave real permanece apenas no ambiente, nao no banco.
