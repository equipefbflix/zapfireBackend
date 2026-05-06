# Scheduler de warming jobs

Esta feature implementa o agendador que busca jobs pendentes vencidos no Supabase e publica mensagens `warming.job.due` no RabbitMQ.

## Fluxo

1. Buscar `public.warming_jobs` com:
   - `status = 'pending'`;
   - `scheduled_at <= now`;
   - limite configuravel.
2. Para cada job, publicar mensagem:

```json
{
  "type": "warming.job.due",
  "version": 1,
  "jobId": "uuid",
  "testRunId": "optional-test-run-id",
  "publishedAt": "2026-05-04T15:00:00Z"
}
```

3. A execucao real fica para o worker consumidor.

## Resultado esperado

- `WarmingJobRepository.ListDuePending`.
- Service `scheduler.WarmingJobScheduler`.
- Testes unitarios garantindo selecao de jobs vencidos e publicacao no publisher.
- Sem validacao MCP propria neste bloco, pois a validacao de banco de `warming_jobs` ja foi coberta; aqui o foco e orquestracao local.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `WarmingJobRepository.ListDuePending` e `scheduler.WarmingJobScheduler.PublishDue`.
2. `go test ./internal/repository ./internal/scheduler` falhou por ausencia dos contratos.
3. Query de jobs pendentes vencidos e scheduler de publicacao implementados.
4. `go test ./...` passou.
5. A validacao real de insert/delete em `public.warming_jobs` ja foi coberta no bloco das rotas; este bloco nao alterou schema nem dados.
