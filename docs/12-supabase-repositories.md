# Repositories Supabase

Esta feature implementa a primeira camada de persistencia Go sobre o schema real do Supabase.

## Escopo inicial

Entidades:

- `phone_numbers`
- `evolution_servers`
- `proxies`

Essas entidades sao pre-requisitos para a proxima feature: criacao de instancia Evolution com proxy.

## Metodologia TDD

1. Criar testes unitarios do contrato de repository.
2. Confirmar falha por tipos/metodos inexistentes.
3. Implementar repositories usando `pgx`.
4. Rodar `go test ./...`.
5. Criar teste integration protegido por build tag para banco real.
6. Validar com dados reais via MCP ou `DATABASE_URL`.
7. Limpar dados de teste por `metadata.testRunId`.

## Contratos

### Phone numbers

- `CreatePhoneNumber`
- `GetPhoneNumberByID`
- `DeletePhoneNumbersByTestRunID`

### Evolution servers

- `CreateEvolutionServer`
- `ListEnabledEvolutionServers`
- `DeleteEvolutionServersByTestRunID`

### Proxies

- `CreateProxy`
- `ListEnabledProxies`
- `DeleteProxiesByTestRunID`

## Regras

- Repositories nao devem conhecer HTTP.
- Repositories nao devem conhecer Evolution API.
- Repositories devem aceitar `context.Context`.
- Dados reais de teste precisam carregar `metadata.testRunId`.
- Deletes de cleanup devem filtrar por `metadata ->> 'testRunId'`.

## Teste real esperado

Com `DATABASE_URL`:

```bash
ENABLE_REAL_TESTS=true \
TEST_RUN_ID=manual-20260430-001 \
DATABASE_URL='postgres://...' \
go test -tags=integration ./internal/repository -v
```

Sem `DATABASE_URL`, a validacao real pode ser feita via MCP com insert/delete SQL, mas isso nao substitui o teste integration do backend.

## Resultado em 2026-04-30

Implementado:

- `PhoneNumberRepository`
- `EvolutionServerRepository`
- `ProxyRepository`
- adaptador `NewPgxExecutor`
- teste integration protegido por `//go:build integration`

Testes locais:

```bash
go test ./...
```

Resultado: passou.

Validacao real via MCP:

- inseriu 1 registro em `public.phone_numbers`;
- inseriu 1 registro em `public.evolution_servers`;
- inseriu 1 registro em `public.proxies`;
- todos com `metadata.testRunId = codex-correct-project-validation`;
- removeu os 3 registros;
- verificacao final retornou zero registros restantes.

Observacao:

- O teste integration Go real ainda depende de `DATABASE_URL` disponivel no ambiente local.
- Enquanto nao houver `DATABASE_URL`, o MCP valida a persistencia real e a limpeza, mas o backend ainda nao executa esse teste contra o banco diretamente.
