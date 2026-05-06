# Fundacao Supabase

Esta feature cria o schema operacional do aquecedor no Supabase real `rxdophybnwoocsdyxyjm`.

## Estado inicial confirmado

Tabelas existentes antes da migration no projeto correto:

- `public.profiles`
- `public.plans`
- `public.subscriptions`
- `public.messages`

Migrations existentes antes do aquecedor:

- `20260428231710`
- `20260428231743`

Essas tabelas pertencem ao projeto Zapfire e nao devem ser alteradas.

## Objetivo da migration

Criar tabelas e enums do dominio do aquecedor:

- servidores Evolution;
- proxies;
- numeros;
- instancias;
- templates;
- scripts e steps;
- jobs;
- logs;
- eventos da Evolution.

## Regras

- Usar nomes explicitos do dominio para evitar colisao.
- Habilitar RLS nas tabelas novas.
- O backend usara credencial server-side/service role para bypass de RLS.
- Nao armazenar API keys e senhas em texto puro quando o valor puder ficar em `.env` ou cofre.
- Toda entidade criada em teste real deve aceitar `metadata.testRunId`.

## Validacao real

Depois da migration:

1. `list_tables` deve listar as novas tabelas.
2. `execute_sql` deve inserir e apagar um registro de teste em `phone_numbers`.
3. `get_advisors(type=security)` deve ser executado.
4. `get_advisors(type=performance)` deve ser executado.

## Resultado em 2026-04-30

Migrations aplicadas:

- `20260430151945_create_aquecedor_core_schema`

Tabelas criadas:

- `public.evolution_servers`
- `public.proxies`
- `public.phone_numbers`
- `public.instances`
- `public.message_templates`
- `public.conversation_scripts`
- `public.conversation_steps`
- `public.warming_jobs`
- `public.execution_logs`
- `public.evolution_events`

Validacoes executadas:

- `list_tables` confirmou as tabelas novas com RLS habilitado.
- Insert real em `public.phone_numbers`, `public.evolution_servers` e `public.proxies` com `metadata.testRunId = codex-correct-project-validation` funcionou.
- Cleanup real removeu os registros criados.
- Verificacao pos-cleanup retornou `0` registros com o `testRunId`.
- `get_advisors(type=security)` foi executado.
- `get_advisors(type=performance)` foi executado.

Hardening incluido na migration:

- `public.aquecedor_set_updated_at()` usa `search_path = public`.
- FKs novas tem indices de cobertura.
- Grants das tabelas novas foram revogados de `anon` e `authenticated`.
- Foram criadas policies deny-all para `anon` e `authenticated`; o backend server-side continua acessando com role privilegiada.

Advisors restantes:

- Security advisor nao retornou lints especificos das tabelas do aquecedor.
- Performance advisor retornou apenas `unused_index` nas tabelas do aquecedor, esperado logo apos criar o schema, antes de trafego real.

## Codigo backend

Pacotes adicionados:

- `internal/config/database.go`
- `internal/database/database.go`

Testes adicionados:

- `internal/config/database_test.go`
- `internal/database/database_test.go`

Validacao local:

```bash
go test ./...
```
