# ISSUES (Append-only Log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here.

### Features

- [ ] [TS-01] Protect llm-proxy endpoint using turnstile.
      - Use tools/mprlab-gateway to work on orchestration. It's a separate repo and will require creating its own branches/pushing to its own repo/opening its own PRs
      - use tools/llm-proxy to understand the urls to protect
      - deliverable is three parts:
            1. code changes that build a docker turnstile image on Github
            2. Changes to tools/mprlab-gateway to orchestrate the protection of llm-proxy endpoints
            3. A write up on how to integrate the changes into the front-end (JS) so that a front-end application can make requests to llm-proxy without exposing a secret

### Improvements

### BugFixes

### Maintenance
