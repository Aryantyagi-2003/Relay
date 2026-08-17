# Changelog

All notable changes to Relay are recorded here.

## [Unreleased]

### Added
- Core diff engine: `.env.example` / `.env` parsing, shape inference and
  validation (url/port/number/bool/string), and markdown-link-leak
  detection — fully unit-tested, no platform dependency yet.
- Vercel API adapter and `relay` CLI wiring local + remote checks.

### Fixed
- Sensitive Vercel env vars were reported as both `missing_remote` and
  `remote_redacted` simultaneously — a contradiction, since a redacted
  key is present by definition. Root cause and fix in the `9cda09e`
  commit.

### Verified
- Ran `relay` against a real, live Vercel project (Orbit) with a real
  API token: 0 errors, 10 warnings, all correctly attributed to
  `remote_redacted` sensitive variables with no false `missing_remote`
  reports — confirms the fix above against live data, not just the
  regression test.
