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

## Status

Core logic (parsing, shape validation, diffing) is built and tested.
Vercel integration and CLI packaging are in progress — not yet
published or installable. This section will be replaced with real
installation instructions once that's true.

## Development

```
go build ./...
go test ./...
```

## Known limitations

- Core-only so far; no platform adapter or CLI entrypoint yet.
