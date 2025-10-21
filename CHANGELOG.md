# Changelog

All notable changes to Gravity Notes live here. Entries follow the [Keep a Changelog](https://keepachangelog.com/) format
and are grouped by the date the work landed on `master`.

## [Unreleased]

### Added

- Go CI workflow (`.github/workflows/go-ci.yml`) ensuring tidy, fmt, vet, and race-tested builds on pushes and PRs.
- Docker image publishing pipeline (`.github/workflows/docker-image.yml`) and `EXAMPLE.md` showing browser integration with the gateway.

### Fixed

- Ensure `/sdk/tvm.mjs` serves the embedded SDK module and add coverage for the handler.
- Prevent replay cache poisoning by marking tokens only after successful DPoP verification and add guard tests.
- Permit multiple requests per access token by enforcing DPoP `jti` replay detection with issued-at validation.
- Support GET requests, inject upstream service secrets, and extend the SDK for llm-proxy coverage.
- Ensure upstream service secret injection overrides client-provided `key` parameters.
- Expose `/health` readiness endpoint and document its usage.

### Documentation

- Align binaries, container image, and docs with the Turnstile naming and domain (`turnstile.mprlab.com`).
- Remove references to external vendors and point widget/verification flows to Turnstile-hosted endpoints.


### Removed
