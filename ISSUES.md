# ISSUES (Append-only Log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here.

### Features

- [x] [TS-01] Protect llm-proxy endpoint using turnstile.
      - analyze turnstile and ensure comprehensive code documentation @README.md. Add documentation, if needed
      - Use tools/mprlab-gateway to work on orchestration of the final solution using docker. It's a separate repo and will require creating its own branches/pushing to its own repo/opening its own PRs
      - Analyze tools/llm-proxy to understand the urls to protect
      - deliverable is three parts:
            1. code changes that build a docker turnstile image on Github
            2. Changes to tools/mprlab-gateway to orchestrate the protection of llm-proxy endpoints. It shall rely/pull the newly built turnstile docker image and supply configuration through .env
            3. A write up (EXAMPLE.md) on how to integrate the changes into a front-end app (JS) so that a front-end application can make requests to llm-proxy without exposing a secret to authenticate against llm-proxy.
      - Status: Added Docker image workflow, wired mprlab gateway through the Turnstile container, and documented browser integration in `EXAMPLE.md`.

### Improvements

### BugFixes

- [x] [TS-02] `/sdk/tvm.mjs` returns 404 (embedded path mismatch).
      - `AttachGatewaySdk` strips `/sdk/` and serves from `http.FS(embeddedSdkFiles)` but the embedded file lives at `sdk/tvm.mjs`, so lookups for `tvm.mjs` fail.
      - Update the handler or embedded path so `/sdk/tvm.mjs` resolves correctly.
      - Status: Resolved locally; regression test and handler fix in place.
- [X] [TS-03] Token replay cache marks tokens before DPoP verification.
      - `handleProtectedProxy` calls `replayCache.mark` immediately after validating JWT claims (before DPoP checks).
      - An attacker can send an invalid DPoP proof with a stolen token to poison the cache and block legitimate requests.
      - Move the replay marking to occur only after all request proofs succeed.
      - Status: Fixed by deferring replay marking until after DPoP validation; regression test added.
- [x] [TS-05] Access tokens become single-use due to replay cache storing by token ID.
      - `/sdk/tvm.mjs` caches the short-lived access token and reuses it until expiry, but the gateway's `replayStore.mark` rejects the second request because it treats the token ID as a replay.
      - Legitimate clients get `401 replay` on every request after the first one within the token lifetime.
      - Track DPoP `jti` values instead so tokens remain reusable; expire proofs on a short rolling window.
      - Status: Resolved by validating DPoP `iat`, caching proof `jti` within a bounded window, and extending coverage for multi-request flows.
- [x] [TS-06] Gateway skips injecting service secret when client supplies `key` query parameter.
      - `newReverseProxy` only sets the `key` query parameter when it is absent.
      - A browser request that includes `key` (empty or wrong) reaches `llm-proxy` without the expected secret, causing 403s despite valid Turnstile/JWT proofs.
      - Override incoming `key` values so the upstream always receives the configured secret.
      - Status: Overwrite `key` during proxying and add regression coverage for existing query parameters.

### Maintenance

- [x] [TS-04] Add GitHub actions that run go tests before merging to master. As an example and an inspiration take a look at this yaml
```yaml
name: Go CI

on:
  push:
    branches:
      - master
    paths:
      - '**/*.go'
      - 'go.mod'
      - 'go.sum'
  pull_request:
    branches:
      - master
    paths:
      - '**/*.go'
      - 'go.mod'
      - 'go.sum'

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true

jobs:
  test:
    runs-on: ubuntu-latest
    timeout-minutes: 15

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          check-latest: true
          cache: true

      - name: Verify modules
        run: |
          go mod tidy
          git diff --exit-code || (echo "::error::go.mod/go.sum drift after tidy" && exit 1)

      - name: Build
        run: go build ./...

      - name: Vet
        run: go vet ./...

      - name: Test (verbose, race)
        run: go test ./... -v -race -count=1

```
      - Status: Added `.github/workflows/go-ci.yml` mirroring the template with enforced `timeout` wrappers and formatting checks.
- [x] [TS-07] Expose a lightweight `/health` endpoint for orchestration.
      - Deployment environments require an unauthenticated readiness probe while `/api` remains protected.
      - Implement `/health` handler returning JSON 200 and ensure documentation covers the endpoint.
      - Status: Added `.github/workflows/go-ci.yml` mirroring the template with enforced `timeout` wrappers and formatting checks.
- [ ] [TS-07] Unify the terminology: this app is called `turnstile`, its binary shall be called `turnstile`, its docker image shall be called `turnstile`. It will for now live at `turnstile.mprlab.com` and the JS will be served from that url. Work through all places and remove any ambiguities or imaginary examples in favor of turnstile.mprlab.com
