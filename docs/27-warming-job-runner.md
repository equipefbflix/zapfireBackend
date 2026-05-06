# Runner de warming jobs

Esta feature implementa o executor interno de um job de aquecimento.

## Fluxo

1. Buscar `warming_jobs` por `jobID`.
2. Exigir `script_id`.
3. Buscar `conversation_steps` do script em ordem.
4. Buscar instancia aberta do `phone_a_id` e do `phone_b_id`.
5. Para cada step:
   - escolher instancia pelo `sender_role` (`a` ou `b`);
   - executar action com `executor.StepExecutor`;
   - registrar `execution_logs`.

## Escopo

- Execucao sequencial.
- Log de sucesso por step.
- Log de falha quando a action retorna erro.
- Sem retry ou idempotencia neste bloco.

## Resultado esperado

- `InstanceRepository.GetOpenByPhoneNumberID`.
- `runner.WarmingJobRunner`.
- Testes unitarios com fakes.
- `go test ./...` passando.

## Validacao realizada

Fluxo executado:

1. Testes criados antes da implementacao para `InstanceRepository.GetOpenByPhoneNumberID` e `runner.WarmingJobRunner`.
2. `go test ./internal/repository ./internal/runner` falhou por ausencia dos contratos.
3. Metodo de repository e runner implementados.
4. `go test ./...` passou.
5. Sem validacao real neste bloco porque a execucao externa ainda usa Evolution fake nos testes; persistencia das tabelas envolvidas ja foi validada nos blocos anteriores.
