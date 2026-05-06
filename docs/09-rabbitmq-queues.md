# Filas RabbitMQ

O RabbitMQ sera usado para desacoplar planejamento, execucao e processamento de eventos. O banco continua sendo a fonte da verdade; a fila transporta trabalho pendente e permite retry, concorrencia controlada e isolamento de falhas.

## Objetivos

- Publicar jobs de aquecimento vencidos em uma fila de execucao.
- Consumir jobs com concorrencia e prefetch configuraveis.
- Publicar webhooks da Evolution em uma fila de eventos.
- Isolar falhas em dead-letter queue.
- Garantir que cada execucao relevante continue persistida no Supabase.

## Topologia

Exchange principal:

```text
aquecedor.events
```

Filas:

```text
aquecedor.warming.jobs
aquecedor.evolution.events
aquecedor.dead_letter
```

Routing keys:

```text
warming.job.due
evolution.event.received
dead_letter
```

## Declaracao esperada

- Exchange `direct`, durable.
- Filas durable.
- Mensagens persistent.
- Dead-letter configurada para falhas apos retries controlados.
- `prefetch` definido por `RABBITMQ_PREFETCH`.

## Payloads

### Job de aquecimento

```json
{
  "type": "warming.job.due",
  "version": 1,
  "jobId": "uuid",
  "testRunId": "optional-test-run-id",
  "publishedAt": "2026-04-29T12:00:00Z"
}
```

### Evento Evolution

```json
{
  "type": "evolution.event.received",
  "version": 1,
  "eventId": "uuid",
  "instanceName": "chip_5511999999999",
  "eventType": "MESSAGES_UPSERT",
  "testRunId": "optional-test-run-id",
  "publishedAt": "2026-04-29T12:00:00Z"
}
```

## Regras de consumo

- O consumidor deve validar `type` e `version`.
- O consumidor deve buscar o estado atual no Supabase antes de executar.
- Ack somente apos persistir resultado final ou estado intermediario seguro.
- Nack com requeue apenas para erro transiente.
- Erro permanente deve ir para dead-letter com motivo registrado.
- Jobs devem ser idempotentes: consumir duas vezes nao pode duplicar envio se ja houver log de sucesso para o step.

## Testes

### Unitarios

- Montagem de DSN RabbitMQ.
- Validacao de configuracao.
- Serializacao dos payloads.
- Declaracao esperada de exchange, filas e bindings usando interface fake.
- Publisher JSON com mensagens persistentes.

### Integracao real

Pre-condicao: `ENABLE_REAL_TESTS=true` e `RABBITMQ_URL` configurado.

Fluxo:

1. Declarar topologia.
2. Publicar mensagem com `testRunId`.
3. Consumir a mensagem.
4. Confirmar conteudo.
5. Ack.
6. Excluir filas temporarias quando forem criadas para teste.

Para nao impactar filas de producao, testes reais devem usar nomes com sufixo:

```text
aquecedor.test.<testRunId>.warming.jobs
```

Comando:

```bash
ENABLE_REAL_TESTS=true \
TEST_RUN_ID=manual-20260429-001 \
RABBITMQ_URL='amqp://user:password@host:5672/' \
go test -tags=integration ./internal/queue -run TestRabbitMQPublishConsumeReal -v
```

Status atual: o teste real esta implementado, mas a conexao com o host de teste informado falhou por timeout TCP na porta `5672`.

## Validacao local em 2026-05-04

Foi executado um RabbitMQ local descartavel via Docker:

```bash
docker run -d --name aquecedor-rabbit-local -p 5672:5672 rabbitmq:3.13-management
```

Teste executado:

```bash
ENABLE_REAL_TESTS=true \
RABBITMQ_URL='amqp://guest:guest@127.0.0.1:5672/' \
TEST_RUN_ID='rabbit-local-manual-002' \
go test -tags=integration ./internal/queue -run TestRabbitMQPublishConsumeReal -v
```

Resultado: `PASS`.

Durante essa validacao, foi identificado e corrigido um bug real na declaracao da fila:

- a fila estava sendo criada com `x-dead-letter-routing-key`, mas sem `x-dead-letter-exchange`;
- RabbitMQ local respondeu `PRECONDITION_FAILED`;
- a topologia foi ajustada para incluir ambos os argumentos nas filas normais.

## Primeira implementacao

A primeira entrega deve conter:

- pacote de config RabbitMQ;
- pacote `internal/queue`;
- interface `Broker`;
- implementacao RabbitMQ;
- publisher para `warming.job.due`;
- declaracao de topologia;
- testes unitarios da config e payload.
