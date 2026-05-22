# SPEC: Backend Instances HTTP Surface

## Contexto

A documentacao do backend preve mais operacoes de instancias do que o servidor HTTP expoe hoje. Isso bloqueia a superficie operacional de instancias no frontend novo.

## Solucao Proposta

Completar a API HTTP de instancias com:

- `GET /api/v1/instances`
- `GET /api/v1/instances/{id}`
- `POST /api/v1/instances/{id}/connect`
- `POST /api/v1/instances/{id}/sync-state`
- `POST /api/v1/instances/{id}/restart`

Tambem fechar os contratos de repository/service necessarios para listagem, detalhe e atualizacao de estado.

## Arquivos que Serao Criados/Modificados

- `backend/docs/specs/SPEC_backend_instances_http_surface.md`
- `backend/internal/httpserver/instances.go`
- `backend/internal/httpserver/instances_test.go`
- `backend/internal/repository/instances.go`
- `backend/internal/instance/service.go`
- `backend/internal/instance/service_test.go`

## Criterios de Aceite

- [ ] Listagem retorna instancias reais com os campos operacionais principais
- [ ] Detalhe por id retorna a instancia correta
- [ ] `connect` aciona Evolution para a instancia selecionada
- [ ] `sync-state` atualiza o status persistido
- [ ] `restart` reutiliza a logica de reinicio
- [ ] Criacao atual de instancia continua funcionando

## Casos de Borda

- id inexistente
- instancia sem Evolution server associado
- Evolution retorna erro em `connect` ou `sync-state`

## Impacto e Riscos

- expande superficie publica do backend
- precisa manter compatibilidade com a criacao de instancias ja validada

## Estrutura de Testes Planejada

- `TestListInstancesRoute`
- `TestGetInstanceRoute`
- `TestConnectInstanceRoute`
- `TestSyncInstanceStateRoute`
- `TestRestartInstanceRoute`

## Status: APROVADO PARA IMPLEMENTACAO
