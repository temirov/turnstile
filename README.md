# Browser-Safe API Gateway (Go) — Turnstile + JWT + DPoP + Tiny JS SDK

A compact Go **gateway** that sits in front of any HTTP API so a **front-end-only** app can call it **without exposing provider keys**.

* `POST /tvm/issue` — optional **Turnstile** check → mint a **short-lived HS256 access token** bound to the browser’s **DPoP** key (`cnf.jkt`).
* `POST /api` — verify **Origin allowlist**, **rate-limit**, **JWT**, **DPoP**, **replay protection** → **reverse-proxy** to your upstream API.
* **Built-in browser SDK** served at `/sdk/tvm.mjs` so integration is a **one-liner**.

> “`/api`” is used as the example **public** path. You can expose any path you want; just keep your reverse proxy and SDK options in sync.

---

## Why use this

* **No secrets in the browser.** Clients only hold a 5-minute, **DPoP-bound** token.
* **General-purpose.** Works for *any* JSON HTTP API (search, generation, tiles, internal tools).
* **No accounts or mTLS.** Capability tokens + optional Turnstile.
* **Zero crypto in your app.** The gateway serves a **tiny JS SDK** that handles keygen, token minting, and DPoP.

---

## What’s in the box

* **Gateway service (Go)** with two endpoints: `/tvm/issue` and `/api`.
* **Embedded SDK** served from `/sdk/tvm.mjs` (ES module).

  * Generates a browser **P-256** keypair (WebCrypto)
  * Mints/refreshes the short-lived token
  * Builds correct **DPoP ES256** JWS (JOSE r||s format)
  * Exposes `postJson()` and `fetchResponse()` convenience methods

---

## Request flow

```
Browser ──(public JWK, optional Turnstile)──►  POST /tvm/issue
Gateway  ── verifies Origin + Turnstile ──► mint HS256 JWT (5 min, cnf.jkt)

Browser ──(Bearer JWT + DPoP)──────────────►  POST /api
Gateway  ── Origin + rate + JWT + DPoP ───► reverse-proxy to upstream
```

---

## Quick start (Docker Compose)

> **Important:** `UPSTREAM_BASE_URL` is the **base origin only** (no path). The gateway preserves `/api`.

```yaml
version: "3.9"
services:
  gateway:
    build: .
    ports: ["8080:8080"]
    environment:
      LISTEN_ADDR: ":8080"
      ORIGIN_ALLOWLIST: "https://your-frontend.example,https://www.your-frontend.example"
      REQUIRE_TURNSTILE: "true"                     # set "false" for local/dev
      TURNSTILE_SECRET_KEY: "replace-with-turnstile-secret"
      TOKEN_LIFETIME_SECONDS: "300"
      TVM_JWT_HS256_KEY: "replace-with-strong-32B-secret"
      UPSTREAM_BASE_URL: "http://upstream-service:8080"  # base origin ONLY
      RATE_LIMIT_PER_MINUTE: "60"
      UPSTREAM_TIMEOUT_SECONDS: "40"

  upstream-service:
    image: your-upstream:latest
    expose: ["8080"]
```

**Run as a binary (no container):**

```bash
go mod tidy && go build -o bin/turnstile .
LISTEN_ADDR=":8080" \
ORIGIN_ALLOWLIST="https://your-frontend.example" \
REQUIRE_TURNSTILE="false" \
TOKEN_LIFETIME_SECONDS="300" \
TVM_JWT_HS256_KEY="replace-with-strong-32B-secret" \
UPSTREAM_BASE_URL="http://localhost:8081" \
RATE_LIMIT_PER_MINUTE="60" \
UPSTREAM_TIMEOUT_SECONDS="40" \
./bin/turnstile
```

## Prebuilt Docker image

Published images are available on GitHub Container Registry and are rebuilt on every push to `master` and on tags:

```bash
docker pull ghcr.io/temirov/turnstile:latest

docker run --rm \
  -e LISTEN_ADDR=":8080" \
  -e ORIGIN_ALLOWLIST="https://app.example" \
  -e REQUIRE_TURNSTILE="true" \
  -e TURNSTILE_SECRET_KEY="1x0000000000000000000000000000000AA" \
  -e TVM_JWT_HS256_KEY="replace-with-strong-32B-secret" \
  -e UPSTREAM_BASE_URL="http://upstream:8080" \
  ghcr.io/temirov/turnstile:latest
```

Use the same environment variables described in the configuration reference to match your deployment.

---

## Reverse proxy (TLS, public)

Keep CORS in the **gateway**. Ensure the proxy passes the client scheme (`X-Forwarded-Proto`) so DPoP `htu` matches.

### Nginx

```nginx
server {
  listen 443 ssl http2;
  server_name api.example.com;

  # ssl_certificate ...;  ssl_certificate_key ...;

  proxy_set_header X-Forwarded-Proto $scheme;

  location /tvm/issue { proxy_pass http://gateway:8080/tvm/issue; }
  location /api       { proxy_pass http://gateway:8080; }   # no rewrite needed

  location = /health  { return 200; }
}
```

### Caddy 2

```caddy
api.example.com {
	encode zstd gzip

	handle_path /tvm/issue* {
		reverse_proxy http://gateway:8080 {
			header_up X-Forwarded-Proto {scheme}
			header_up Host {host}
		}
	}

	handle_path /api* {
		reverse_proxy http://gateway:8080 {
			header_up X-Forwarded-Proto {scheme}
			header_up Host {host}
		}
	}

	respond /health 200
}
```

*If the proxy and gateway share a Docker network, point to `http://gateway:8080`.*

---

## Front-end usage (with the built-in SDK)

You don’t write crypto. Import the module the gateway serves and call **one function**.

```html
<script type="module">
  import { createGatewayClient } from "/sdk/tvm.mjs";

  const gatewayClient = createGatewayClient({
    baseUrl: "https://api.example.com",
    // Optional if Turnstile is enabled:
    // turnstileTokenProvider: () => window.turnstile.getResponse(),
    // Optional if you expose a different public path:
    // apiPath: "/api",
    // tokenPath: "/tvm/issue",
  });

  // Call your protected upstream via the gateway:
  const result = await gatewayClient.postJson({ any: "payload your upstream expects" });
  console.log(result);
</script>
```

* `postJson(payload, { path })` — sends JSON to `baseUrl + (path||apiPath)`; returns parsed JSON.
* `fetchResponse(payload, { path })` — same, but returns the raw `Response`.

You can route multiple backends by varying `path` (e.g., `"/api/search"`, `"/api/generate"`), all protected by the same checks.

---

## Configuration reference

| Env var                    | Req        | Example                                       | Default | Purpose                                     |
| -------------------------- | ---------- | --------------------------------------------- | ------- | ------------------------------------------- |
| `LISTEN_ADDR`              | no         | `:8080`                                       | `:8080` | Bind address.                               |
| `ORIGIN_ALLOWLIST`         | **yes**    | `https://app.example,https://www.app.example` | —       | Exact Origins allowed (admission + CORS).   |
| `REQUIRE_TURNSTILE`        | no         | `true`                                        | `false` | If `true`, `/tvm/issue` verifies Turnstile. |
| `TURNSTILE_SECRET_KEY`     | when above | `1x000…`                                      | —       | Secret for Turnstile verification.          |
| `TOKEN_LIFETIME_SECONDS`   | no         | `300`                                         | `300`   | Access token TTL; keep short.               |
| `TVM_JWT_HS256_KEY`        | **yes**    | random 32+ bytes                              | —       | HS256 signing key for tokens.               |
| `UPSTREAM_BASE_URL`        | **yes**    | `http://upstream:8080`                        | —       | **Base origin only** (no path).             |
| `UPSTREAM_SERVICE_SECRET`  | no         | `super-secret-value`                          | —       | Injected as `key` query parameter for upstreams that expect a shared secret. |
| `RATE_LIMIT_PER_MINUTE`    | no         | `60`                                          | `60`    | Per Origin+IP limit per 60s window.         |
| `UPSTREAM_TIMEOUT_SECONDS` | no         | `40`                                          | `40`    | Per-request upstream timeout.               |

---

## Security model (concise)

* **Capability token**: HS256 JWT, audience-scoped to the gateway, TTL ≈ 5 minutes.
* **Proof-of-possession**: Token carries `cnf.jkt` (JWK thumbprint). Each request must present a **DPoP** JWS signed by that key; gateway verifies method (`htm`) and URL (`htu`).
* **Bot friction**: Optional **Turnstile** at token issuance (not per request).
* **Replay defense**: In-memory `jti` cache until expiry.
* **Origin enforcement**: Exact allowlist; CORS headers added by gateway.
* **Rate limiting**: Per Origin + IP within a 60-second window.

**Scaling**: For multiple replicas, keep traffic sticky per client or back the `jti` cache with a shared store.

---

## Customizing paths / multiple APIs

* Public path is **your choice** (we use `/api` as the example).
* Keep the reverse proxy routes and SDK `apiPath` consistent.
* To protect multiple upstream routes, just call `postJson(payload, { path: "/api/whatever" })`. The gateway applies the same checks before proxying.

---

## Troubleshooting

* **CORS blocked** → The app’s `Origin` must *exactly* match an entry in `ORIGIN_ALLOWLIST`.
* **`htu_mismatch`** → Ensure the reverse proxy sets `X-Forwarded-Proto` correctly; the browser URL must match what the gateway computes.
* **`cnf_mismatch`** → Token was minted for a different DPoP key; re-mint after generating the keypair (SDK handles this).
* **502/504** → Verify `UPSTREAM_BASE_URL` and upstream health; adjust `UPSTREAM_TIMEOUT_SECONDS`.

---

## License

Add your preferred license text here.
