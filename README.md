# Relay

[![CI](https://github.com/Aryantyagi-2003/Relay/actions/workflows/ci.yml/badge.svg)](https://github.com/Aryantyagi-2003/Relay/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/Aryantyagi-2003/Relay)](https://github.com/Aryantyagi-2003/Relay/releases)
[![downloads](https://img.shields.io/github/downloads/Aryantyagi-2003/Relay/total)](https://github.com/Aryantyagi-2003/Relay/releases)
[![go report card](https://goreportcard.com/badge/github.com/Aryantyagi-2003/Relay)](https://goreportcard.com/report/github.com/Aryantyagi-2003/Relay)
[![stars](https://img.shields.io/github/stars/Aryantyagi-2003/Relay)](https://github.com/Aryantyagi-2003/Relay/stargazers)
[![license](https://img.shields.io/github/license/Aryantyagi-2003/Relay)](LICENSE)

Relay is a CLI that diffs your local `.env` against what's actually
configured on your deploy platform, and flags values that don't look
right *before* you deploy.

## Origin story

During a previous project's first Vercel deploy, a database URL got
copied out of rendered markdown docs — brackets and all — and pasted
straight into an env var. It sat there invisibly wrong until the build
failed with a cryptic error that took real time to trace back to a
malformed string. Relay exists to catch exactly that class of mistake
before it ever reaches a deploy.

## Installation

Requires Go 1.21+.

```
go install github.com/Aryantyagi-2003/Relay/cmd/relay@latest
```

This installs the `relay` binary to `$(go env GOPATH)/bin` — make sure
that directory is on your `PATH`. Prebuilt binaries for macOS, Linux, and
Windows are also attached to each [release](https://github.com/Aryantyagi-2003/Relay/releases),
if you'd rather not build from source.

## Usage

```
relay --example .env.example --env .env
```

Checks your local `.env` against `.env.example`: missing variables,
values that don't match their expected shape (a URL that isn't a URL, a
port that isn't a number), and — the bug this tool exists for — values
that look like a pasted markdown link instead of a raw value.

Real output, run against a fixture reproducing the exact bug that
motivated this tool:

```
$ relay --example .env.example --env .env
[ERROR] DATABASE_URL: local value looks like a pasted markdown link (e.g. "[text](url)"), not a raw value (markdown_leak)
[WARNING] IS_PRODUCTION: declared in .env.example but not set locally (missing_local)

1 error(s), 1 warning(s)
$ echo $?
1
```

Add `--project <vercel-project>` (with `VERCEL_TOKEN` set, or `--token`)
to also check what's actually configured on Vercel:

```
export VERCEL_TOKEN=...
relay --example .env.example --env .env --project my-app
```

Exits `0` with no error-level issues, `1` if any were found (so it's
safe to drop into CI), `2` on a usage/setup problem (bad flags, missing
files, a failed API call).

Run `relay --help` for the full flag list.

## Status

Published as [v0.1.0](https://github.com/Aryantyagi-2003/Relay/releases/tag/v0.1.0):
real, checksum-verified cross-platform binaries, installable via
`go install` or GitHub Releases. Core logic, the Vercel adapter, and the
CLI are all unit-tested and verified end-to-end against a real, live
Vercel project. See [CHANGELOG.md](CHANGELOG.md) for what's landed so
far.

## Real usage

The badges at the top of this README (release, downloads, stars, CI,
code quality) are live GitHub/shields.io data, not static screenshots —
they reflect the real current numbers whenever you're reading this,
not a snapshot from when this was written.

**Honest snapshot as of 2026-08-17** (the day of the v0.1.0 launch
post): 0 stars, 0 forks, 2 total release-asset downloads — and those 2
downloads are from this project's own clean-install verification, not
from an external user yet. Shared publicly the same day; no install or
feedback numbers from that post have come in yet. This section gets
updated with whatever the real numbers turn out to be, including if
that's still close to zero — see [known limitations](#known-limitations)
for the same standard applied to the tool itself.

## Development

```
go build ./...
go test ./...
```

## Known limitations

- Vercel only for now; other platforms (Railway, etc.) aren't supported.
- Shape inference from variable names is a heuristic (e.g. `*_URL` →
  URL) and can be wrong for unconventionally named variables — use a
  `# hint:<shape>` comment in `.env.example` to override it.
- Vercel-marked "sensitive" variables can't have their values checked
  remotely (Vercel withholds them from the API by design); Relay reports
  them as present-but-unverifiable rather than silently skipping them.
- Real-world adoption is early (see [Real usage](#real-usage) above) —
  this is a newly launched tool, not an established one.
