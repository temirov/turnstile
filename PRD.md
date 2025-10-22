# ETS Product Requirements (User Perspective)

## Summary

Ephemeral Token Service (ETS) lets browser-only front ends talk to protected
HTTP APIs without embedding long-lived secrets. ETS issues short-lived,
DPoP-bound access tokens and proxies calls to the upstream on behalf of the
user. This document captures the product expectations from an end-user and
operator perspective.

## Primary User Stories

1. **Front-end developer** embeds ETS SDK to call `llm-proxy` securely.
2. **Authenticated browser user** obtains a token transparently and receives
   upstream responses with minimal latency.
3. **Operator** monitors ETS readiness and configures origins/secrets without
   redeploying front-end code.

## Functional Requirements

### Token Issuance (`POST /tvm/issue`)

- Accepts a JSON payload containing the caller’s P-256 public JWK.
- Returns `200 OK` with `{ accessToken, expiresIn }` when the origin is
  allow-listed and rate limits are respected.
- Rejects unsupported methods with `405 method_not_allowed` and malformed JWKs
  with `400` error codes.
- Tokens must expire within five minutes and encode the JWK thumbprint so they
  cannot be re-used by other keys.

### Protected Proxy (`/api` subtree)

- Requires `Authorization: Bearer <token>` and `DPoP` headers on every request.
- Validates DPoP `htu`, `htm`, signature, and ensures the thumbprint matches the
  issued token (`cnf.jkt`).
- Rejects replayed DPoP proofs with `401` while leaving the upstream untouched.
- Injects the configured `UPSTREAM_SERVICE_SECRET` as `key` query parameter
  before forwarding to the upstream server.

### SDK Expectations

- Hosted at `https://ets.mprlab.com/sdk/tvm.mjs`.
- Generates ephemeral ECDSA P-256 key pairs per session and refreshes tokens as
  needed.
- Exposes `postJson` and `fetchResponse` helpers that retry token minting when
  expiry is near (≤20 seconds remaining).
- Surfaces gateway errors (HTTP status + JSON payload) to the caller for UI
  display.

### Health and Observability

- `GET /health` responds with `200` and `{ "status": "ok" }` within 50ms on a
  healthy instance.
- Logs must omit secrets and include request identifiers for correlation.

## Non-Functional Requirements

- **Security**: No long-lived credentials in the browser; JWTs are unusable
  without the matching DPoP key. CORS restricted to the configured origin
  allowlist.
- **Performance**: Token issuance completes within 250ms P95; proxied requests
  add ≤50ms overhead to upstream latency under nominal load.
- **Reliability**: Rate limiter defaults prevent a single origin/IP from making
  more than 60 requests per minute. Replay cache survives process lifetime and
  evicts entries when tokens expire.
- **Scalability**: ETS instances can be horizontally scaled behind the Caddy
  gateway; sticky sessions or shared replay stores are required to prevent DPoP
  reuse across nodes.

## Error Handling

- User-facing errors must be structured JSON `{ "error": "code" }` so front
  ends can display helpful messages.
- Distinct error codes:
  - `method_not_allowed` for non-POST issuance calls.
  - `invalid_json`, `unsupported_jwk`, `bad_jwk_thumbprint` for malformed input.
  - `rate_limited` when window caps are hit.
  - `missing_bearer`, `invalid_token`, `missing_dpop`, `bad_dpop_*` for auth
    failures.
- Operators receive actionable log entries at WARN level when repeated failures
  occur (e.g., persistent `bad_dpop_sig`).

## Configuration Expectations

- All runtime tweaks flow through environment variables (`ORIGIN_ALLOWLIST`,
  `TVM_JWT_HS256_KEY`, `UPSTREAM_BASE_URL`, etc.) documented in `README.md`.
- Configuration changes take effect on process restart; no manual code edits are
  required for new origins or upstream secrets.

## Success Metrics

- <1% of issuance requests fail outside of intentional rate limiting or invalid
  inputs.
- 99% of proxied requests complete successfully when the upstream is healthy.
- Integration time for a new front-end is ≤30 minutes using the hosted SDK and
  README instructions.
