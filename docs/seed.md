# Project seed

## Mission

Build the pure-Go SQLite adapter for `github.com/candango/sqlok` using
`modernc.org/sqlite`. This repository is intentionally separate from the
`sqlok` core and from the CGO-based `sqlok-sqlite-mattn` adapter.

The deployment default is `CGO_ENABLED=0`. The adapter owns the pinned driver,
SQLite connection/bootstrap details, SQLite dialect configuration, and the
real-database E2E suite.

## Current decisions

- Module: `github.com/candango/sqlok-sqlite-modernc`.
- Driver: `modernc.org/sqlite`.
- Initial pin: `v1.59.0`.
- Initial Go baseline: Go 1.25.
- CI matrix: Go 1.25, 1.26, and 1.27.
- Build model: pure Go; validate every matrix entry with `CGO_ENABLED=0`.
- Core boundary: use the public `github.com/candango/sqlok` API; do not copy
  compiler, mapper, session, or execution internals into this repository.
- SQLite adapters remain separate so an application selects exactly one driver
  registration.

## First implementation slice

1. Add the smallest public adapter entry point for opening/configuring SQLite.
2. Define the SQLite dialect behavior required by the core API, including
   placeholder rendering where the core contract requires it.
3. Add real-database E2E tests for:
   - schema setup and cleanup;
   - Mapper scanning and value extraction;
   - `LoadContext` and Identity Map reuse;
   - `Flush` inside an application-owned transaction;
   - commit and rollback behavior;
   - generated keys;
   - simple and composite primary keys where SQLite supports the case;
   - missing rows and actionable mapping/database errors.
4. Add a reproducible benchmark for representative Load and Flush paths. Keep
   adapter cost separate from application-level query fan-out.
5. Document supported Go, SQLite, and driver versions plus the exact commands
   used for tests and benchmarks.

## Explicit non-goals

- Do not add SQLite-specific imports to `github.com/candango/sqlok`.
- Do not implement the CGO/mattn adapter here.
- Do not add a global application cache or hide transaction ownership inside
  the adapter.
- Do not optimize based on a single microbenchmark; profile or measure the
  complete path first.

## Suggested shape after the first slice

```text
.
├── AGENTS.md
├── README.md
├── go.mod
├── docs/
│   └── seed.md
├── adapter.go              # small public connection/driver boundary
├── adapter_test.go         # unit-level adapter behavior
└── integration/
    └── sqlite_test.go      # real SQLite E2E contract
```

The exact package and file names are still open. Keep the design concrete until
there are multiple real consumers that justify an abstraction.

## Completion evidence

Before closing the initial issue, show:

- CI green on Go 1.25, 1.26, and 1.27 with `CGO_ENABLED=0`;
- `CGO_ENABLED=0 go test ./...` and `CGO_ENABLED=0 go vet ./...` output;
- the driver and Go versions used;
- the E2E schema/fixture and transaction cases covered;
- benchmark method and results, if a performance claim is made;
- documentation updated with any support or behavior decision.
