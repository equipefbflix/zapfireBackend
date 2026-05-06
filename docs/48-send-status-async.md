# `send_status` assíncrono

Data: 2026-05-05

## Contexto

Na validacao real com a Evolution em `https://evo.askgeni.us`, o endpoint:

- `POST /message/sendStatus/{instance}`

deixou de falhar por payload, mas permaneceu sem resposta HTTP final por longo tempo.

Em paralelo, o servidor real continuou emitindo eventos de webhook para `status@broadcast`, indicando processamento assincrono.

## Decisao

O backend passa a tratar `send_status` como operacao de aceite assincrono:

- se a Evolution responder normalmente, o backend usa a resposta;
- se a Evolution consumir a requisicao e ficar sem responder ate o timeout de espera do status, o backend considera o step aceito;
- o `execution_log` grava `response_payload.acceptedAsync = true`;
- nao ha dependencia de `messageKey` para considerar o step executado.

## Implementacao

Arquivos:

- `internal/evolution/client.go`
- `internal/evolution/client_test.go`
- `internal/executor/step_executor.go`
- `internal/executor/step_executor_test.go`
- `internal/runner/warming_jobs.go`
- `internal/runner/warming_jobs_test.go`

## Regra tecnica

`send_status` usa uma janela curta de espera no client:

- no maximo `10s`
- ou menos, se o timeout global do client for menor

Erros aceitos como `acceptedAsync`:

- `context deadline exceeded`
- timeouts de rede
- fechamento de conexao sem resposta apos envio
- `EOF`

Erros HTTP reais, como `400` e `500`, continuam falhando o step.

## Impacto

- o runner nao fica bloqueado por muito tempo em `send_status`
- o backend passa a refletir o comportamento real da Evolution
- o fechamento funcional de `send_status` passa a depender do webhook, nao da resposta HTTP sincrona
