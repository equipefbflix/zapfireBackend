# SPEC: Supabase Auth ES256 Support

## Contexto

Os testes reais de integração do frontend contra a API autenticada falharam com `401 invalid bearer token`.

Diagnóstico:

- o Supabase do projeto `rxdophybnwoocsdyxyjm` publica JWKS com chave `EC`
- os access tokens reais vêm com `alg = ES256`
- o verifier do backend hoje aceita apenas `RS256` e faz parse apenas de JWK `RSA`

Resultado: tokens reais válidos do Supabase são rejeitados pelo backend.

## Solução Proposta

Expandir o verifier do backend para:

- aceitar `ES256` além de `RS256`
- carregar chaves `EC P-256` do JWKS
- continuar suportando `RSA`

## Arquivos que Serão Criados/Modificados

- `backend/internal/auth/verifier.go` — suporte a JWKS EC e verificação ES256
- `backend/internal/auth/verifier_test.go` — teste novo cobrindo token ES256

## Critérios de Aceite

- [ ] token RS256 continua válido
- [ ] token ES256 válido do formato Supabase é aceito
- [ ] claims essenciais continuam sendo validadas

## Casos de Borda

- JWK EC com curva não suportada deve falhar
- token com `kid` ausente continua falhando
- JWKS sem chaves válidas continua falhando

## Impacto e Riscos

- impacto direto no middleware global de autenticação
- mudança pequena, mas de alta criticidade porque afeta toda a API protegida

## Estrutura de Testes Planejada

- `TestSupabaseVerifierVerify`
- `TestSupabaseVerifierVerifyES256`

## Status: APROVADA PARA IMPLEMENTAÇÃO
