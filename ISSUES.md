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

- [ ] [TS-18] Support configurable upstream authentication and routing.
      - Allow operators to define per-upstream credentials (headers, query params, bearer tokens) instead of the hard-coded `key` query param.
      - Permit routing multiple public paths to distinct upstream endpoints within ETS configuration.
      - Document the contract so front-end integrations understand which routes map to which upstreams.

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
- [x] [TS-08] Unify the terminology: this app is called `turnstile`, its binary shall be called `turnstile`, its docker image shall be called `turnstile`. It will for now live at `turnstile.mprlab.com` and the JS will be served from that url. Work through all places and remove any ambiguities or imaginary examples in favor of turnstile.mprlab.com
      - Status: Renamed the Docker binary/entrypoint to `turnstile`, updated runtime messaging, and rewrote docs to reference `turnstile.mprlab.com` and the aligned SDK endpoint.
- [x] [TS-09] Remove third-party vendor references across code and documentation now that Turnstile runs independently of external services.
      - Status: Replaced verification endpoints and examples with Turnstile-hosted URLs and scrubbed vendor-specific wording from docs.
- [x] [TS-10] Rebrand the service as Ephemeral Token Service (ETS), rename binaries, and update all URLs to `ets.mprlab.com`, including integrations under `tools/mprlab-gateway/`.
      - Status: Updated code, docs, SDK, and gateway integrations to emit ETS naming, env vars, and `ets.mprlab.com` endpoints.
- [x] [TS-12] `/api/*` requests bypass the gateway.
      - `http.ServeMux` patterns without a trailing slash only match the exact path. The server registers `/api` but not `/api/`, so `/api/search` or `/api/generate` fall through to a 404 despite the README promising multi-route support.
      - Register subtree handlers for `/api/` alongside the existing `/api` route and add regression coverage proving nested paths proxy correctly.
      - Status: Added `/api/` handler, kept `/api`, and introduced `TestNewHTTPServer_RoutesApiSubpaths` to assert nested routes return the DPoP 401 instead of 404.
- [x] [TS-11] Propagate the same renames from TS-10 in the nested tools/mprlab-gateway repo (service name, .env.ets, ETS image tag) and push that project’s branch when ready—the files were updated locally but remain outside this commit.
      - Status: Renamed the compose service to `llm-ets`, switched env samples to `.env.ets`, refreshed docs, and pushed nested repo branch `maintenance/TS-11-ets-alignment`.
- [x] [TS-13] Provide an operator-facing CLI helper to mint strong secrets for ETS deployments.
      - Status: Added Cobra-backed `generate-secrets` subcommand that emits both assignments, wrapped entropy errors, and documented usage in the main README.
- [x] [TS-14] Align gateway orchestration samples with the ETS CLI secret workflow.
      - Status: Pointed `.env.ets.sample` and the gateway README at `ets generate-secrets`, and clarified the front-end example to show how `ets.mprlab.com` proxies to `llm-proxy`.
- [x] [TS-15] Remove the legacy `REQUIRE_ETS` flag so ETS verification is always enforced.
      - Status: Config loader temporarily required `ETS_SECRET_KEY`, handlers verified ETS tokens, and docs captured the interim behavior (superseded by TS-16).
- [x] [TS-16] Make ETS self-contained by dropping external challenge verification.
      - Status: Removed `ETS_SECRET_KEY`, inlined issuance checks, updated SDK/docs, and refreshed gateway configs.
- [x] [TS-17] `/tvm/issue` retains GET method and lacks regression coverage after removing ETS tokens.
      - Status: Restricted `/tvm/issue` to POST and added positive/negative issuance tests in `handlers_issue_test.go` to cover the self-contained flow.
