# ISSUES (Append-only Log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here.

### Features

- [ ] [TS-01] Protect llm-proxy endpoint using turnstile.
      - analyze turnstile and ensure comprehensive code documentation @README.md. Add documentation, if needed
      - Use tools/mprlab-gateway to work on orchestration of the final solution using docker. It's a separate repo and will require creating its own branches/pushing to its own repo/opening its own PRs
      - Analyze tools/llm-proxy to understand the urls to protect
      - deliverable is three parts:
            1. code changes that build a docker turnstile image on Github
            2. Changes to tools/mprlab-gateway to orchestrate the protection of llm-proxy endpoints. It shall rely/pull the newly built turnstile docker image and supply configuration through .env
            3. A write up (EXAMPLE.md) on how to integrate the changes into a front-end app (JS) so that a front-end application can make requests to llm-proxy without exposing a secret to authenticate against llm-proxy.

### Improvements

### BugFixes

- [ ] [TS-02] `/sdk/tvm.mjs` returns 404 (embedded path mismatch).
      - `AttachGatewaySdk` strips `/sdk/` and serves from `http.FS(embeddedSdkFiles)` but the embedded file lives at `sdk/tvm.mjs`, so lookups for `tvm.mjs` fail.
      - Update the handler or embedded path so `/sdk/tvm.mjs` resolves correctly.
      - Status: Resolved locally; regression test and handler fix in place.

### Maintenance
