# Plano de execucao TDD

Metodologia definida para este projeto:

1. Documentar a feature.
2. Criar teste automatizado para a feature.
3. Rodar o teste e confirmar falha.
4. Implementar a feature.
5. Ajustar ate o teste passar.
6. Validar com dados reais quando a feature integra Evolution API ou Supabase.
7. Registrar execucao, verificar persistencia no banco e remover dados de teste.

## Regras obrigatorias

- Nenhuma feature de integracao sera considerada pronta sem teste usando Supabase real.
- Toda feature que chama Evolution API tera ao menos um teste com instancia real de teste.
- Todo teste real deve criar dados com prefixo identificavel `test_` ou metadata `{"testRunId": "..."}`.
- Todo teste real deve limpar os dados criados no final.
- Se a limpeza falhar, o teste deve reportar os IDs pendentes.
- Testes reais devem ser separados de testes unitarios para evitar chamadas externas por acidente.
- O backend deve permitir dry-run apenas para planejamento; dry-run nao substitui teste real.

## Organizacao dos testes

Categorias:

- `unit`: sem rede, sem Supabase, sem Evolution.
- `integration`: usa Supabase real e pode usar Evolution real.
- `contract`: valida payloads enviados para Evolution contra fixtures e schemas esperados.
- `e2e`: cria instancia real, conecta/consulta, envia acao real entre numeros de teste e limpa dados.

Comandos previstos:

```bash
go test ./...
go test -tags=integration ./...
go test -tags=e2e ./...
```

Variaveis para testes reais:

```env
ENABLE_REAL_TESTS=false
TEST_RUN_ID=manual-20260429-001
TEST_CLEANUP_ENABLED=true

TEST_PHONE_A=5511999999999
TEST_PHONE_B=5511888888888
TEST_EVOLUTION_SERVER=evo1
TEST_PROXY_ID=
```

`ENABLE_REAL_TESTS` deve ser `true` para qualquer teste que toque Supabase real ou Evolution real.

## Sequencia de implementacao

### Fase 0: desbloqueio Supabase

Documentacao:

- Atualizar `03-supabase-schema.md` com tabelas reais.
- Registrar divergencias entre schema existente e schema proposto.

Teste primeiro:

- Teste de conectividade Supabase.
- Teste de leitura de migrations/tabelas.

Falha esperada inicial:

- Enquanto a conexao Postgres do projeto `rxdophybnwoocsdyxyjm` estiver indisponivel, os testes de banco real devem falhar com erro claro de conectividade/configuracao.

Implementacao:

- Configurar `DATABASE_URL` e `SUPABASE_SERVICE_ROLE_KEY`.
- Criar pacote `internal/config`.
- Criar pacote `internal/db`.

Validacao real:

- Abrir conexao no projeto correto.
- Inserir uma linha em tabela temporaria ou tabela de teste.
- Remover a linha apos verificar leitura.

### Fase 1: base do backend Go

Documentacao:

- Estrutura de pastas.
- Padrao de config, logger, erros e respostas HTTP.

Teste primeiro:

- `GET /health` retorna status do processo.
- `GET /api/v1/health` inclui Supabase e servidores Evolution configurados.

Falha esperada inicial:

- Rotas nao existem.

Implementacao:

- `cmd/api/main.go`
- `internal/httpserver`
- `internal/config`
- `internal/health`

Validacao real:

- Subir servidor local.
- Chamar health.
- Verificar conexao real com Supabase quando credenciais estiverem disponiveis.

### Fase 2: cliente Evolution API

Documentacao:

- Contratos de create instance, connect, connection state, settings, text, presence e reaction.

Teste primeiro:

- Testes contract com fixtures para payload v2.
- Teste real de health/fetch instances em uma Evolution de teste.

Falha esperada inicial:

- Cliente nao implementado.

Implementacao:

- `internal/evolution/client.go`
- Tipos de request/response.
- Timeout, retry e tratamento de erro.

Validacao real:

- Chamar `GET /instance/fetchInstances` em Evolution real.
- Persistir log da chamada quando usado pelo backend.

### Fase 3: schema e repositorios Supabase

Documentacao:

- Migrations finais baseadas no schema real.
- Politica de RLS e uso de service role.

Teste primeiro:

- Inserir, consultar, atualizar e excluir `evolution_servers`, `proxies`, `phone_numbers`, `instances` e `execution_logs`.

Falha esperada inicial:

- Tabelas nao existem ou repositorios nao existem.

Implementacao:

- Migrations.
- Repositorios Go.
- Transacoes para jobs e logs.

Validacao real:

- Aplicar migration em Supabase.
- Rodar testes integration.
- Confirmar limpeza dos registros de teste.

### Fase 4: gerenciamento de instancias com proxy

Documentacao:

- Fluxo de criacao de instancia.
- Regras de escolha de Evolution API.
- Regras de escolha de proxy.

Teste primeiro:

- Criar telefone de teste.
- Criar proxy de teste.
- Chamar rota `POST /api/v1/instances`.
- Verificar payload enviado para Evolution contem `proxyHost`, `proxyPort`, `proxyProtocol`, `proxyUsername`, `proxyPassword`.
- Verificar persistencia em `instances`.
- Excluir instancia/dados de teste.

Falha esperada inicial:

- Rota e servico nao existem.

Implementacao:

- Handler de instancias.
- Service de criacao.
- Adapter Evolution create instance.
- Persistencia do retorno.

Validacao real:

- Criar instancia real em Evolution de teste.
- Consultar `fetchInstances`.
- Remover registros do backend criados pelo teste.
- Remocao da instancia na Evolution depende de endpoint confirmado da versao instalada.

### Fase 5: conexao e monitoramento de instancias

Documentacao:

- Estados internos.
- Politica de restart.
- Cron `instance_connection_check`.

Teste primeiro:

- `POST /api/v1/instances/{id}/connect` chama Evolution real.
- `POST /api/v1/instances/{id}/sync-state` atualiza estado no banco.
- Cron processa uma instancia de teste.

Falha esperada inicial:

- Rotas e scheduler nao existem.

Implementacao:

- Connect.
- Sync state.
- Scheduler base.

Validacao real:

- Instancia real retorna pairing/QR.
- Estado persistido no Supabase.
- Dados de teste removidos quando aplicavel.

### Fase 6: templates e scripts de conversa

Documentacao:

- Modelo de templates.
- Modelo de conversation scripts e steps.
- Categorias e pesos.

Teste primeiro:

- Criar templates reais no banco.
- Criar script com steps.
- Selecionar script por score e peso.
- Limpar dados.

Falha esperada inicial:

- Repositorios e rotas nao existem.

Implementacao:

- CRUD de templates.
- CRUD de scripts.
- Validador de steps.

Validacao real:

- Insercao, consulta e exclusao no Supabase real.

### Fase 7: executor de aquecimento

Documentacao:

- Maquina de estados de `warming_jobs`.
- Regras de delays, retries e logs.

Teste primeiro:

- Criar job real com dois numeros de teste.
- Rodar executor.
- Enviar presenca e texto via Evolution real.
- Salvar `execution_logs`.
- Atualizar `warming_jobs`.
- Limpar dados.

Falha esperada inicial:

- Executor nao existe.

Implementacao:

- Planner.
- Executor.
- Lock de jobs.
- Registro de logs.

Validacao real:

- Mensagem real enviada entre instancias de teste.
- Log salvo com chave da mensagem Evolution.
- Dados de teste removidos.

### Fase 8: reactions, replies, stickers e midias

Documentacao:

- Capacidades confirmadas na Evolution instalada.
- Payloads por acao.
- Dependencias de mensagem anterior.

Teste primeiro:

- Enviar texto real.
- Usar message key retornada para reply/reaction.
- Enviar sticker/midia com asset de teste.
- Limpar dados.

Falha esperada inicial:

- Acoes nao implementadas.

Implementacao:

- Action dispatcher.
- Tipos por action.
- Persistencia de message keys.

Validacao real:

- Acao visivel no WhatsApp de teste.
- Logs completos no Supabase.

### Fase 9: score de aquecimento

Documentacao:

- Formula de score.
- Variaveis `.env`.
- Penalidades e bonus.

Teste primeiro:

- Criar logs reais/fakes controlados no Supabase.
- Rodar recalculo.
- Verificar score esperado.
- Limpar dados.

Falha esperada inicial:

- Recalculador nao existe.

Implementacao:

- Score service.
- Cron diario.
- Atualizacao de `phone_numbers`.

Validacao real:

- Score muda apos execucao real.
- Status muda para `warm` ao atingir threshold.

### Fase 10: redundancia multi-Evolution

Documentacao:

- Weighted routing.
- Health status.
- Politicas de fallback.

Teste primeiro:

- Configurar ao menos duas Evolutions de teste.
- Marcar uma como `down`.
- Verificar que novas instancias/jobs usam servidor saudavel.
- Limpar dados.

Falha esperada inicial:

- Router nao existe.

Implementacao:

- Evolution router.
- Health check.
- Capacidade/concurrency.

Validacao real:

- Jobs nao sao enviados para Evolution marcada como indisponivel.

## Protocolo de limpeza

Todo teste real deve:

1. Gerar `testRunId`.
2. Escrever `testRunId` em `metadata` ou prefixo textual.
3. Registrar IDs criados durante o teste.
4. Executar cleanup em `defer`.
5. Verificar que nao sobraram registros com o mesmo `testRunId`.

Ordem de remocao:

1. `execution_logs`
2. `warming_jobs`
3. `conversation_steps`
4. `conversation_scripts`
5. `message_templates`
6. `instances`
7. `phone_numbers`
8. `proxies` criados pelo teste
9. `evolution_servers` criados pelo teste
10. `evolution_events`

## Criterios de pronto por feature

Uma feature esta pronta apenas quando:

- documentacao da feature existe;
- teste foi criado antes da implementacao;
- falha inicial foi observada;
- teste passa localmente;
- teste real passou quando a feature depende de Supabase/Evolution;
- dados reais de teste foram limpos;
- falhas e limitacoes foram documentadas.
