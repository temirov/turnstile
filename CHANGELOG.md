# Changelog

All notable changes to Gravity Notes live here. Entries follow the [Keep a Changelog](https://keepachangelog.com/) format
and are grouped by the date the work landed on `master`.

## [Unreleased]

### Added

- Go CI workflow (`.github/workflows/go-ci.yml`) ensuring tidy, fmt, vet, and race-tested builds on pushes and PRs.

### Fixed

- Ensure `/sdk/tvm.mjs` serves the embedded SDK module and add coverage for the handler.
- Prevent replay cache poisoning by marking tokens only after successful DPoP verification and add guard tests.
- Permit multiple requests per access token by enforcing DPoP `jti` replay detection with issued-at validation.

### Documentation


### Removed
