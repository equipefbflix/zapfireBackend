# Planner real de conversas

Esta feature implementa o primeiro planner operacional para escolher scripts de conversa de forma consistente.

## Escopo desta etapa

- selecionar `conversation_script` por score
- respeitar categoria/contexto quando informado
- evitar repeticao recente do mesmo script para o mesmo par
- aplicar cooldown por par
- calcular `scheduledAt` dentro de uma janela configuravel

## Regras iniciais

- filtrar scripts `enabled = true`
- filtrar scripts cujo score esteja entre `min_warming_score` e `max_warming_score`
- se `category` vier no contexto, priorizar scripts daquela categoria
- excluir scripts usados recentemente para o mesmo par dentro do cooldown
- se todos tiverem sido usados recentemente, fazer fallback para os elegiveis
- escolher de forma deterministica:
  - maior `weight`
  - em empate, o menos recentemente usado
  - em empate, ordem alfabetica do nome

## Agendamento

- delay calculado dentro de `WARMING_MIN_DELAY_SECONDS` e `WARMING_MAX_DELAY_SECONDS`
- horario respeita janela `WARMING_WINDOW_START_HOUR` e `WARMING_WINDOW_END_HOUR`
- se o horario cair fora da janela, mover para a proxima abertura

## TDD

1. documentar regras
2. criar testes unitarios do planner
3. confirmar falha
4. implementar planner e repositorios auxiliares
5. criar teste integration real com banco
6. ajustar ate passar

## Status em 2026-05-05

Primeiro subbloco concluido:

- `PlannerConfig` carregado de env
- selecao por score e categoria
- cooldown por par
- anti-repeticao do mesmo script no par
- delay deterministico dentro da janela
- validacao integration real em banco passou
