# Limites de concorrencia do worker

Data: 2026-05-05

## Objetivo

Evitar execucoes simultaneas demais para o mesmo par de numeros e para o mesmo Evolution server.

## TDD aplicado

1. Testes de repository para:
   - `UpdateStatus`
   - `CountRunningByPair`
   - `CountRunningByEvolutionServer`
2. Testes do runner para:
   - marcar `running -> success`
   - marcar `running -> failed`
   - bloquear execucao quando a gate falha
3. Testes da gate de concorrencia:
   - abaixo do limite
   - limite por par
   - limite por Evolution server
   - erro de contador
4. Falha inicial:
   - `cmd/worker` ainda usava a assinatura antiga de `NewWarmingJobRunner`
5. Implementacao:
   - `runner.MaxConcurrencyGate`
   - `WarmingJobRepository.UpdateStatus`
   - `WarmingJobRepository.CountRunningByPair`
   - `WarmingJobRepository.CountRunningByEvolutionServer`
   - wiring dos limites por env no `cmd/worker`
6. Validacao:
   - `go test ./...`
   - `go test -tags=integration ./internal/repository -run TestRepositoriesRealDatabase -v`

## Regras implementadas

- O runner tenta resolver as instancias do job.
- Antes de executar os steps, consulta a gate de concorrencia.
- Se a gate reprovar:
  - o job vai para `failed`
  - o erro fica em `warming_jobs.error`
  - nenhuma action e executada
- Se a gate aprovar:
  - o job vai para `running`
  - executa os steps
  - termina em `success` ou `failed`

## Variaveis de ambiente

```env
WARMING_MAX_RUNNING_JOBS_PER_PAIR=1
WARMING_MAX_RUNNING_JOBS_PER_EVOLUTION_SERVER=5
```

## Arquivos principais

- `internal/runner/concurrency.go`
- `internal/runner/warming_jobs.go`
- `internal/repository/warming_jobs.go`
- `cmd/worker/main.go`
- `internal/config/planner.go`

## Validacao real

Validado no projeto Supabase `rxdophybnwoocsdyxyjm`:

- criou phones, server, instances e warming job de teste
- atualizou status para `running`
- confirmou:
  - `CountRunningByPair = 1`
  - `CountRunningByEvolutionServer = 1`
- atualizou status para `failed`
- confirmou leitura de `status` e `error`
- cleanup executado pelo proprio teste
