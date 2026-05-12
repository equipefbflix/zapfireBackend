# SPEC: FBFlix Proxy Integration

## Context
The backend needs to obtain real proxies from ProxyFbflix/FBFlix and use them when creating WhatsApp instances in Evolution API. The API is hosted as a Supabase Edge Function and uses a permanent B2B bearer token.

## Proposed Solution
Implement a client for:

- `POST https://[platform].supabase.co/functions/v1/proxyfbflix-api`
- `Authorization: Bearer fbx_b2b_...`
- `action=get-proxy-list` to list already allocated proxies.
- `action=purchase-proxy-with-balance` to purchase proxies when product/order data is provided.

The sync service maps the real FBFlix/Webshare response (`results[].proxy_address`, `valid`, `country_code`, `city_name`) and purchase response (`proxies[].ip`) into the internal `proxies` table. When FBFlix returns a literal proxy password, it is stored with the `literal:` prefix in `password_secret_name` so the instance service can resolve it for Evolution without writing credentials into files.

## Files to be Created/Modified
- `backend/internal/proxy/fbflix/client.go` — API client for the Edge Function contract.
- `backend/internal/proxy/fbflix/sync.go` — Service to sync proxies to the database.
- `backend/internal/instance/evolution_factory.go` — Resolve `literal:` proxy passwords.
- `backend/internal/instance/service.go` — Support literal proxy secrets in tests and service.
- `backend/internal/httpserver/proxies.go` — Redact literal proxy passwords in HTTP responses.
- `backend/cmd/api/main.go` — Default FBFlix endpoint to the Edge Function URL.

## Acceptance Criteria
- [x] Successfully fetch proxy list from FBFlix API using `POST` and `action=get-proxy-list`.
- [x] Map FBFlix proxy format to internal `Proxy` model.
- [x] Upsert proxies into the database (avoid duplicates based on host/port).
- [x] Handle API errors, HTML fallback responses and timeouts gracefully.
- [x] Support proxy credentials returned by FBFlix when creating Evolution instances.
- [x] Support purchase response parsing via `action=purchase-proxy-with-balance`.

## Edge Cases
- What happens when the FBFlix API returns an empty list? -> Log warning, keep existing proxies.
- What happens when the token is invalid? -> Log error, notify administrator.
- What happens when a proxy already exists? -> Update its status/metadata.
- What happens when the configured URL returns the SPA HTML? -> Return an explicit `expected json response, got html` error.

## Impact and Risks
- External dependency: Changes in FBFlix API might break the sync.
- Security: FBFlix returns literal proxy passwords. They must not be logged or exposed in HTTP responses.
- Database: Current schema only has `password_secret_name`; literal proxy credentials use a `literal:` prefix until a vault-backed secret store is added.

## Planned Test Structure
- `TestClient/ListProxiesSuccess` — Validates `POST`, action payload and real response parsing.
- `TestClient/ListProxiesUnauthorized` — Validates error handling for invalid token.
- `TestClient/PurchaseProxyWithBalanceSuccess` — Validates purchase response parsing.
- `TestSyncService` — Validates that proxies are saved with `literal:` password support.
- `TestServiceCreateWithLiteralProxyPassword` — Validates Evolution instance creation receives the real proxy password.
- `TestClientListProxiesRealAPI` — Validates real FBFlix API list call with `FBFLIX_B2B_TOKEN`.
- `TestServiceCreateRealEvolutionInstanceWithFBFlixProxyE2E` — Validates real FBFlix proxy + Supabase + Evolution instance creation when Evolution credentials are valid.

## Validation

Passing:

- `go test ./...`
- `ENABLE_REAL_TESTS=true go test -tags=integration ./internal/proxy/fbflix -run TestClientListProxiesRealAPI -count=1 -v`
- `ENABLE_REAL_TESTS=true go test -tags=integration ./internal/evolution -run TestEvolutionFetchInstancesReal -count=1 -v`
- `ENABLE_REAL_TESTS=true go test -tags=e2e ./internal/instance -run TestServiceCreateRealEvolutionInstanceWithFBFlixProxyE2E -count=1 -v`

The E2E uses the real FBFlix API, real Supabase database and real Evolution API. It creates one instance with an FBFlix proxy, verifies it through `FetchInstances`, and runs cleanup for the created Evolution instance and database rows.

## Status: IMPLEMENTED
