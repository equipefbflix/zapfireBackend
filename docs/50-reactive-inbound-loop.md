# Loop reativo por webhook

Esta etapa liga o fluxo automatico:

`webhook inbound -> planner -> warming_job -> scheduler -> fila -> worker -> runner`

## Objetivo

Quando uma instancia gerenciada receber mensagem de outra instancia tambem gerenciada:

- detectar o inbound real no webhook da Evolution
- identificar qual numero local recebeu a mensagem
- identificar qual numero remoto gerenciado enviou a mensagem
- escolher um `conversation_script` elegivel via planner
- criar um `warming_job` automatico com `scheduled_at` calculado

## Regras desta etapa

- reagir apenas a `MESSAGES_UPSERT`
- ignorar eventos `fromMe = true`
- ignorar `status@broadcast`
- ignorar remetentes que nao existam em `phone_numbers`
- criar job apenas se nao houver job recente do mesmo par dentro de um cooldown curto de inbound
- usar `category = reactive` no planner para separar scripts automaticos dos scripts de validacao/manual
- calcular `pairScore` pelo menor `warming_score` do par
- gravar metadata do gatilho:
  - `autoReactive = true`
  - `triggerEventId`
  - `triggerMessageId`
  - `triggerInstanceName`
  - `triggerRemoteJid`
  - `triggerMessageType`

## Placeholders de payload

Para que o mesmo script funcione para qualquer direcao do par, os payloads dos steps reativos podem usar placeholders:

- `{{phoneA}}`
- `{{phoneB}}`
- `{{triggerMessageId}}`
- `{{triggerRemoteJid}}`

Exemplo:

```json
{
  "number": "{{phoneB}}",
  "text": "Recebi aqui, vou te responder em seguida"
}
```

## Scheduler

Um novo processo `cmd/scheduler` publica continuamente jobs `pending` e vencidos para a fila RabbitMQ.

Sem ele, o webhook cria o job, mas o job nao sai do banco para execucao automatica.

## TDD

1. documentar regras
2. criar testes unitarios do reactor
3. criar teste de integracao do `evolutionsync` chamando o reactor
4. confirmar falha
5. implementar service e wiring
6. criar `cmd/scheduler`
7. validar com banco real e duas instancias conectadas
