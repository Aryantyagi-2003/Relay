# Relay

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

Shared publicly on 2026-08-17. No install or feedback numbers have come
in yet — this line will be replaced with the real count (even if it
stays at or near zero) as soon as there's real data to report, not
before.

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
