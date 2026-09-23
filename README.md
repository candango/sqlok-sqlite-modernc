# sqlok-sqlite-modernc

Pure-Go SQLite adapter for [`github.com/candango/sqlok`](https://github.com/candango/sqlok), using [`modernc.org/sqlite`](https://modernc.org/sqlite).

## Status

Project scaffold. Implementation and real-database E2E coverage are tracked in
this repository's GitHub issue and seeded in [`docs/seed.md`](docs/seed.md).

## Goals

- Keep SQLite support outside the driver-agnostic `sqlok` core.
- Support `CGO_ENABLED=0` builds.
- Exercise Mapper, `LoadContext`, `Flush`, transactions, generated keys, and
  SQLite placeholder behavior against a real SQLite database.
- Benchmark representative adapter paths without confusing driver cost with
  application query volume.

## Development

The initial module targets Go 1.25 and pins `modernc.org/sqlite` in `go.mod`.
The adapter must remain usable through the public `sqlok` API; callers should
not need to know about compiler plans or execution buffers.

CI validates Go 1.25, 1.26, and 1.27 with `CGO_ENABLED=0`.

```bash
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go vet ./...
```

See [`AGENTS.md`](AGENTS.md) for repository-specific guidance.
