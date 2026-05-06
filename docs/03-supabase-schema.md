# Supabase e schema

Projeto alvo atual: `rxdophybnwoocsdyxyjm`.

## Estado de acesso

Verificacao em 2026-04-30:

- `get_project_url` retornou `https://rxdophybnwoocsdyxyjm.supabase.co`.
- `list_tables(schemas=["public"], verbose=false)` funcionou.
- `list_migrations` funcionou.
- `execute_sql` funcionou para insert/delete de validacao.

Conclusao: o MCP esta funcional para o projeto correto.

## Schema real atual

Tabelas preexistentes em `public` antes do schema do aquecedor:

- `public.profiles`
- `public.plans`
- `public.subscriptions`
- `public.messages`

Migrations existentes antes do aquecedor:

- `20260428231710`
- `20260428231743`

As tabelas existentes pertencem ao projeto Zapfire e nao devem ser alteradas pelo backend do aquecedor.

Migration do aquecedor aplicada:

- `20260430151945_create_aquecedor_core_schema`

## Extensoes recomendadas

```sql
create extension if not exists pgcrypto;
create extension if not exists pg_cron;
```

`pg_cron` so sera necessario se parte dos agendamentos rodar dentro do Postgres. A primeira versao pode manter crons no processo Go.

## Schema proposto

```sql
create type evolution_health_status as enum ('healthy', 'degraded', 'down', 'disabled');
create type instance_status as enum ('created', 'connecting', 'open', 'close', 'failed', 'paused');
create type phone_status as enum ('new', 'warming', 'warm', 'paused', 'blocked', 'lost');
create type execution_status as enum ('pending', 'running', 'success', 'failed', 'cancelled', 'skipped');
create type warming_action_type as enum (
  'send_text',
  'send_presence',
  'send_typing',
  'send_recording',
  'send_reaction',
  'send_reply',
  'send_sticker',
  'send_media',
  'send_audio',
  'send_status',
  'update_profile_status'
);

create table public.evolution_servers (
  id uuid primary key default gen_random_uuid(),
  name text not null unique,
  base_url text not null,
  api_key_secret_name text not null,
  enabled boolean not null default true,
  weight integer not null default 1,
  max_concurrent_jobs integer not null default 5,
  health_status evolution_health_status not null default 'healthy',
  last_health_check_at timestamptz,
  last_error text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table public.proxies (
  id uuid primary key default gen_random_uuid(),
  name text not null unique,
  host text not null,
  port integer not null,
  protocol text not null check (protocol in ('http', 'https', 'socks4', 'socks5')),
  username text,
  password_secret_name text,
  enabled boolean not null default true,
  max_instances integer,
  current_instances integer not null default 0,
  last_check_at timestamptz,
  last_error text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table public.phone_numbers (
  id uuid primary key default gen_random_uuid(),
  phone_e164 text not null unique,
  label text,
  status phone_status not null default 'new',
  warming_score numeric(5,2) not null default 0,
  daily_message_count integer not null default 0,
  total_message_count integer not null default 0,
  last_activity_at timestamptz,
  metadata jsonb not null default '{}',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table public.instances (
  id uuid primary key default gen_random_uuid(),
  phone_number_id uuid not null references public.phone_numbers(id) on delete cascade,
  evolution_server_id uuid not null references public.evolution_servers(id),
  proxy_id uuid references public.proxies(id),
  instance_name text not null unique,
  evolution_instance_id text,
  instance_api_key_secret_name text,
  status instance_status not null default 'created',
  owner_jid text,
  profile_name text,
  profile_picture_url text,
  last_connection_check_at timestamptz,
  last_connected_at timestamptz,
  last_disconnected_at timestamptz,
  last_error text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table public.message_templates (
  id uuid primary key default gen_random_uuid(),
  category text not null,
  title text not null,
  body text not null,
  weight integer not null default 1,
  enabled boolean not null default true,
  min_warming_score numeric(5,2) not null default 0,
  max_warming_score numeric(5,2) not null default 100,
  metadata jsonb not null default '{}',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table public.conversation_scripts (
  id uuid primary key default gen_random_uuid(),
  name text not null unique,
  category text not null,
  enabled boolean not null default true,
  weight integer not null default 1,
  min_warming_score numeric(5,2) not null default 0,
  max_warming_score numeric(5,2) not null default 100,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table public.conversation_steps (
  id uuid primary key default gen_random_uuid(),
  script_id uuid not null references public.conversation_scripts(id) on delete cascade,
  step_order integer not null,
  sender_role text not null check (sender_role in ('a', 'b')),
  action_type warming_action_type not null,
  template_id uuid references public.message_templates(id),
  payload jsonb not null default '{}',
  min_delay_seconds integer not null default 10,
  max_delay_seconds integer not null default 120,
  created_at timestamptz not null default now(),
  unique (script_id, step_order)
);

create table public.warming_jobs (
  id uuid primary key default gen_random_uuid(),
  script_id uuid references public.conversation_scripts(id),
  phone_a_id uuid not null references public.phone_numbers(id),
  phone_b_id uuid not null references public.phone_numbers(id),
  status execution_status not null default 'pending',
  scheduled_at timestamptz not null,
  started_at timestamptz,
  finished_at timestamptz,
  current_step_order integer not null default 0,
  error text,
  metadata jsonb not null default '{}',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (phone_a_id <> phone_b_id)
);

create table public.execution_logs (
  id uuid primary key default gen_random_uuid(),
  warming_job_id uuid references public.warming_jobs(id) on delete set null,
  instance_id uuid references public.instances(id) on delete set null,
  action_type warming_action_type,
  status execution_status not null,
  request_payload jsonb,
  response_payload jsonb,
  evolution_message_key jsonb,
  remote_jid text,
  error text,
  duration_ms integer,
  created_at timestamptz not null default now()
);

create table public.evolution_events (
  id uuid primary key default gen_random_uuid(),
  evolution_server_id uuid references public.evolution_servers(id),
  instance_name text,
  event_type text not null,
  payload jsonb not null,
  received_at timestamptz not null default now(),
  processed_at timestamptz,
  processing_error text
);

create index idx_instances_status on public.instances(status);
create index idx_instances_evolution_server on public.instances(evolution_server_id);
create index idx_phone_numbers_status_score on public.phone_numbers(status, warming_score);
create index idx_warming_jobs_due on public.warming_jobs(status, scheduled_at);
create index idx_execution_logs_job on public.execution_logs(warming_job_id, created_at);
create index idx_evolution_events_unprocessed on public.evolution_events(processed_at) where processed_at is null;
```

## Segredos

Nao salvar chaves sensiveis em texto puro nas tabelas. As colunas `*_secret_name` apontam para nomes no ambiente do backend ou em um cofre futuro.

Para MVP, o backend pode carregar todos os segredos pelo `.env`:

- API keys das Evolution APIs.
- Senhas de proxy.
- Service role key do Supabase.

## RLS

Como o backend usara service role, as tabelas operacionais podem iniciar sem politicas publicas. Se houver painel frontend usando anon key, criar views/RLS separadas para leitura controlada.
