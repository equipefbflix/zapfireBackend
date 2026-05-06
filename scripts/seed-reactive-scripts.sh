#!/usr/bin/env zsh
set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:8081}"
AUTH_HEADER=()
if [[ -n "${API_BEARER_TOKEN:-}" ]]; then
  AUTH_HEADER=(-H "Authorization: Bearer ${API_BEARER_TOKEN}")
fi

post_script() {
  local payload="$1"
  curl -fsS -X POST "${API_BASE_URL}/api/v1/conversation-scripts" \
    -H 'Content-Type: application/json' \
    "${AUTH_HEADER[@]}" \
    --data "$payload"
  echo
}

post_script '{
  "name": "reactive_short_v1",
  "category": "reactive",
  "enabled": true,
  "weight": 30,
  "minWarmingScore": 0,
  "maxWarmingScore": 25,
  "steps": [
    {
      "stepOrder": 1,
      "senderRole": "a",
      "actionType": "send_reaction",
      "payload": {
        "remoteJid": "{{triggerRemoteJid}}",
        "fromMe": false,
        "messageId": "{{triggerMessageId}}",
        "reaction": "👀"
      },
      "minDelaySeconds": 1,
      "maxDelaySeconds": 2
    },
    {
      "stepOrder": 2,
      "senderRole": "a",
      "actionType": "send_typing",
      "payload": {
        "number": "{{phoneB}}",
        "delay": 1200
      },
      "minDelaySeconds": 1,
      "maxDelaySeconds": 3
    },
    {
      "stepOrder": 3,
      "senderRole": "a",
      "actionType": "send_reply",
      "payload": {
        "number": "{{phoneB}}",
        "text": "Vi sua mensagem agora. Seguindo por aqui.",
        "delay": 1400,
        "remoteJid": "{{triggerRemoteJid}}",
        "fromMe": false,
        "messageId": "{{triggerMessageId}}"
      },
      "minDelaySeconds": 3,
      "maxDelaySeconds": 8
    },
    {
      "stepOrder": 4,
      "senderRole": "b",
      "actionType": "send_typing",
      "payload": {
        "number": "{{phoneA}}",
        "delay": 900
      },
      "minDelaySeconds": 4,
      "maxDelaySeconds": 10
    },
    {
      "stepOrder": 5,
      "senderRole": "b",
      "actionType": "send_text",
      "payload": {
        "number": "{{phoneA}}",
        "text": "Fechado, seguimos aquecendo essa linha.",
        "delay": 1200
      },
      "minDelaySeconds": 6,
      "maxDelaySeconds": 14
    }
  ]
}'

post_script '{
  "name": "reactive_medium_v1",
  "category": "reactive",
  "enabled": true,
  "weight": 20,
  "minWarmingScore": 10,
  "maxWarmingScore": 60,
  "steps": [
    {
      "stepOrder": 1,
      "senderRole": "a",
      "actionType": "send_typing",
      "payload": {
        "number": "{{phoneB}}",
        "delay": 1000
      },
      "minDelaySeconds": 1,
      "maxDelaySeconds": 4
    },
    {
      "stepOrder": 2,
      "senderRole": "a",
      "actionType": "send_reply",
      "payload": {
        "number": "{{phoneB}}",
        "text": "Recebi aqui. Vamos seguir aquecendo com calma.",
        "delay": 1300,
        "remoteJid": "{{triggerRemoteJid}}",
        "fromMe": false,
        "messageId": "{{triggerMessageId}}"
      },
      "minDelaySeconds": 3,
      "maxDelaySeconds": 10
    },
    {
      "stepOrder": 3,
      "senderRole": "b",
      "actionType": "send_typing",
      "payload": {
        "number": "{{phoneA}}",
        "delay": 1100
      },
      "minDelaySeconds": 5,
      "maxDelaySeconds": 12
    },
    {
      "stepOrder": 4,
      "senderRole": "b",
      "actionType": "send_text",
      "payload": {
        "number": "{{phoneA}}",
        "text": "Perfeito. Mantendo o fluxo dessa conversa.",
        "delay": 1400
      },
      "minDelaySeconds": 7,
      "maxDelaySeconds": 15
    },
    {
      "stepOrder": 5,
      "senderRole": "a",
      "actionType": "send_typing",
      "payload": {
        "number": "{{phoneB}}",
        "delay": 1200
      },
      "minDelaySeconds": 8,
      "maxDelaySeconds": 18
    },
    {
      "stepOrder": 6,
      "senderRole": "a",
      "actionType": "send_text",
      "payload": {
        "number": "{{phoneB}}",
        "text": "Boa. Esse ritmo fica natural para o aquecimento.",
        "delay": 1200
      },
      "minDelaySeconds": 10,
      "maxDelaySeconds": 22
    }
  ]
}'

post_script '{
  "name": "reactive_long_v1",
  "category": "reactive",
  "enabled": true,
  "weight": 10,
  "minWarmingScore": 30,
  "maxWarmingScore": 100,
  "steps": [
    {
      "stepOrder": 1,
      "senderRole": "a",
      "actionType": "send_reaction",
      "payload": {
        "remoteJid": "{{triggerRemoteJid}}",
        "fromMe": false,
        "messageId": "{{triggerMessageId}}",
        "reaction": "👍"
      },
      "minDelaySeconds": 1,
      "maxDelaySeconds": 3
    },
    {
      "stepOrder": 2,
      "senderRole": "a",
      "actionType": "send_typing",
      "payload": {
        "number": "{{phoneB}}",
        "delay": 900
      },
      "minDelaySeconds": 2,
      "maxDelaySeconds": 5
    },
    {
      "stepOrder": 3,
      "senderRole": "a",
      "actionType": "send_reply",
      "payload": {
        "number": "{{phoneB}}",
        "text": "Confirmado. Linha ativa e com bom ritmo.",
        "delay": 1300,
        "remoteJid": "{{triggerRemoteJid}}",
        "fromMe": false,
        "messageId": "{{triggerMessageId}}"
      },
      "minDelaySeconds": 4,
      "maxDelaySeconds": 10
    },
    {
      "stepOrder": 4,
      "senderRole": "b",
      "actionType": "send_typing",
      "payload": {
        "number": "{{phoneA}}",
        "delay": 1000
      },
      "minDelaySeconds": 6,
      "maxDelaySeconds": 12
    },
    {
      "stepOrder": 5,
      "senderRole": "b",
      "actionType": "send_text",
      "payload": {
        "number": "{{phoneA}}",
        "text": "Ótimo. Seguimos com troca espaçada e consistente.",
        "delay": 1200
      },
      "minDelaySeconds": 8,
      "maxDelaySeconds": 16
    },
    {
      "stepOrder": 6,
      "senderRole": "a",
      "actionType": "send_typing",
      "payload": {
        "number": "{{phoneB}}",
        "delay": 1000
      },
      "minDelaySeconds": 10,
      "maxDelaySeconds": 18
    },
    {
      "stepOrder": 7,
      "senderRole": "a",
      "actionType": "send_text",
      "payload": {
        "number": "{{phoneB}}",
        "text": "Fechando esse ciclo por aqui. Depois retomamos.",
        "delay": 1200
      },
      "minDelaySeconds": 12,
      "maxDelaySeconds": 24
    }
  ]
}'
