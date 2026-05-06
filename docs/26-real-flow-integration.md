# Teste integrado de fluxo minimo

Este bloco adiciona cobertura integrada para o fluxo minimo de persistencia:

1. criar dois `phone_numbers`;
2. criar um `warming_job`;
3. criar um `execution_log` associado ao job;
4. remover dados de teste.

## Execucao

O teste fica protegido por build tag e variavel de ambiente:

```bash
ENABLE_REAL_TESTS=true \
DATABASE_URL='postgres://...' \
go test -tags=integration ./internal/repository -run TestRepositoriesRealDatabase -v
```

## Resultado esperado

- O teste nao roda por padrao em `go test ./...`.
- Quando habilitado, usa `TEST_RUN_ID` para cleanup.
- `execution_logs` associados ao job sao apagados antes do job.
