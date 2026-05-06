# Plano de acao

Este plano detalha as pendencias atuais e como cada uma deve ser executada seguindo a metodologia do projeto:

1. documentar a feature;
2. criar teste;
3. rodar e confirmar falha;
4. implementar;
5. ajustar ate passar;
6. validar com dados reais quando houver integracao externa;
7. limpar dados de teste.

## 1. DATABASE_URL real no backend

### Objetivo

Permitir que o backend Go acesse diretamente o Postgres do projeto Supabase `rxdophybnwoocsdyxyjm`, sem depender do MCP para runtime.

### Plano

1. Obter `DATABASE_URL` do projeto Zap Fire no painel Supabase ou secret manager.
2. Exportar localmente, sem gravar em arquivo versionavel:

```bash
export DATABASE_URL='postgres://...'
```

3. Rodar teste integration existente:

```bash
ENABLE_REAL_TESTS=true \
TEST_RUN_ID=manual-db-001 \
DATABASE_URL='postgres://...' \
go test -tags=integration ./internal/repository -v
```

4. Ajustar se houver erro de SSL, pool ou permissao.
5. Usar `DATABASE_URL` no `cmd/api` para wiring real.

### Criterios de pronto

- `database.Open` conecta no Postgres real.
- Teste integration dos repositories passa.
- Dados reais sao criados e removidos.
- Nenhuma credencial e gravada nos arquivos do repo.

### Status

Concluido. `DATABASE_URL` valido foi testado em runtime e o E2E `worker-local-e2e` passou contra o projeto `rxdophybnwoocsdyxyjm`.

## 2. Proxies reais

### Objetivo

Cadastrar proxies reais em `public.proxies` para que a criacao de instancia envie `proxyHost`, `proxyPort`, `proxyProtocol`, `proxyUsername` e `proxyPassword` para a Evolution.

### Plano

1. Definir formato das envs de proxy:

```env
PROXY_01_PASSWORD=...
```

2. Criar rota `POST /api/v1/proxies`.
3. Criar teste do handler com fake service.
4. Criar service/repository se faltar contrato.
5. Validar insert real via MCP ou integration.
6. Confirmar que `InstanceService` seleciona o proxy habilitado.

### Criterios de pronto

- Proxy cadastrado no banco sem senha em texto puro.
- `password_secret_name` aponta para env.
- Criacao de instancia usa proxy quando habilitado.
- Cleanup por `metadata.testRunId` funciona.

### Status

Implementado no backend para cadastro/listagem e selecao na criacao de instancia. Ainda faltam dados reais de proxy para validacao operacional com Evolution.

## 3. Rotas auxiliares

### Objetivo

Expor endpoints de CRUD/consulta para alimentar o banco sem SQL manual.

### Rotas prioritarias

```http
POST /api/v1/phone-numbers
GET /api/v1/phone-numbers
POST /api/v1/proxies
GET /api/v1/proxies
GET /api/v1/evolution-servers
```

### Plano

1. Documentar contrato no `04-backend-api.md`.
2. Criar testes HTTP com fakes.
3. Implementar interfaces no `httpserver`.
4. Implementar services finos ou usar repositories injetados.
5. Fazer wiring no `cmd/api`.
6. Rodar `go test ./...`.
7. Validar insert/list/delete real por MCP ou integration.

### Criterios de pronto

- Rotas retornam JSON consistente.
- Valida campos obrigatorios.
- Testes unitarios passam.
- Persistencia real validada.

### Status

Implementado para phone numbers, proxies, Evolution servers, message templates, conversation scripts, warming jobs e execution logs.

## 4. E2E real de criacao de instancia Evolution

### Objetivo

Criar uma instancia real na Evolution usando a rota do backend e persistir o resultado em Supabase.

### Plano

1. Confirmar endpoint de remocao/delecao de instancia da Evolution instalada.
2. Criar teste `e2e` protegido por build tag:

```bash
ENABLE_REAL_TESTS=true \
DATABASE_URL='postgres://...' \
AUTHENTICATION_API_KEY='...' \
SERVER_URL='https://evo.askgeni.us' \
go test -tags=e2e ./... -run TestCreateEvolutionInstanceE2E -v
```

3. Criar phone number de teste.
4. Criar instance via service/HTTP.
5. Consultar `fetchInstances`.
6. Persistir `instances`.
7. Remover dados do banco.
8. Remover instancia na Evolution, se endpoint confirmado.

### Criterios de pronto

- Instancia real criada.
- Registro salvo em `public.instances`.
- Cleanup no banco feito.
- Cleanup na Evolution confirmado ou documentado como bloqueio.

### Status

Concluido. A Evolution real respondeu a `POST /instance/create` e `DELETE /instance/delete/{instance}`. O E2E real de criacao, persistencia e cleanup passou.

## 5. RabbitMQ real

### Objetivo

Usar RabbitMQ real para publicar e consumir jobs.

### Plano

1. Confirmar conectividade externa:

```bash
nc -vz rabbitmq.askgeni.us 5672
nc -vz rabbitmq.askgeni.us 5671
```

2. Confirmar protocolo:

- `amqp://` porta `5672`;
- `amqps://` porta `5671`.

3. Rodar teste integration existente:

```bash
ENABLE_REAL_TESTS=true \
TEST_RUN_ID=manual-rabbit-001 \
RABBITMQ_URL='amqp://...' \
go test -tags=integration ./internal/queue -run TestRabbitMQPublishConsumeReal -v
```

4. Implementar consumer real de warming jobs.
5. Implementar retry/dead-letter.

### Criterios de pronto

- Topologia declarada no RabbitMQ real.
- Mensagem publicada e consumida.
- Ack funcionando.
- Fila temporaria de teste removida.

### Status

Parcialmente implementado. Topologia/publisher/scheduler local existem e passam em testes unitarios. Teste real segue bloqueado por timeout TCP em `rabbitmq.askgeni.us:5672`.

## 6. Features maiores do aquecedor

### Objetivo

Implementar o fluxo de aquecimento completo.

### Ordem recomendada

1. Templates:
   - `message_templates`;
   - CRUD HTTP;
   - testes unitarios e banco real.

2. Scripts:
   - `conversation_scripts`;
   - `conversation_steps`;
   - validacao de ordem e delays.

3. Jobs:
   - `warming_jobs`;
   - planner;
   - publisher RabbitMQ.

4. Executor:
   - consumer RabbitMQ;
   - chamada Evolution;
   - `execution_logs`;
   - idempotencia.

5. Webhooks:
   - `POST /api/v1/webhooks/evolution`;
   - salvar `evolution_events`;
   - publicar evento na fila.

6. Score:
   - regras por env;
   - calculo de bonus/penalidade;
   - transicao de status do numero.

7. Acoes WhatsApp:
   - texto;
   - presenca;
   - reply;
   - reaction;
   - sticker;
   - midia;
   - status.

### Criterios de pronto

- Cada subfeature tem doc propria.
- Cada subfeature tem teste falhando antes da implementacao.
- Toda persistencia real usa `testRunId`.
- Todo teste real limpa os dados.

### Status

Parcialmente implementado ate worker inicial e executor de steps para texto, presenca e reaction. Ainda falta ligar o worker ao consumidor RabbitMQ real, selecionar instancias por job e implementar idempotencia por step.
