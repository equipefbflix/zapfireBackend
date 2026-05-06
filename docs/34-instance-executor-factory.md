# Executor por instancia

Esta feature corrige o runner para ambientes com multiplas Evolution APIs.

## Problema

O runner atual usa um `StepExecutor` unico. Isso ignora `instance.evolution_server_id`, o que fica incorreto quando diferentes instancias pertencem a servidores Evolution diferentes.

## Solucao

1. Buscar `evolution_servers` pelo `instance.EvolutionServerID`.
2. Resolver `api_key` pelo `api_key_secret_name`.
3. Criar `evolution.Client` por instancia/servidor.
4. Criar `executor.StepExecutor` a partir desse client.

## Resultado esperado

- `EvolutionServerRepository.GetByID`.
- `runner.InstanceExecutorFactory`.
- Runner escolhe executor por instancia.
- Testes unitarios cobrindo resolucao por servidor.
