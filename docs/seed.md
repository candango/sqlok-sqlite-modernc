# Project seed

## Mission

Build the pure-Go SQLite adapter for `github.com/candango/sqlok` using
`modernc.org/sqlite`. This repository is intentionally separate from the
`sqlok` core and from the CGO-based `sqlok-sqlite-mattn` adapter.

The deployment default is `CGO_ENABLED=0`. The adapter owns the pinned driver,
SQLite connection/bootstrap details, SQLite placeholder behavior, and the
real-database E2E suite.

## Current decisions

- Module: `github.com/candango/sqlok-sqlite-modernc`.
- Driver: `modernc.org/sqlite` v1.59.0.
- Initial Go baseline: Go 1.25.
- CI matrix: Go 1.25, 1.26, and 1.27.
- Build model: pure Go; validate every matrix entry with `CGO_ENABLED=0`.
- Core dependency: `github.com/candango/sqlok` at the published pseudo-version
  for commit `18e5570`, which provides numeric generated-key propagation.
- Core boundary: use the public `github.com/candango/sqlok` API; do not copy
  compiler, mapper, session, or execution internals into this repository.
- SQLite adapters remain separate so an application selects exactly one driver
  registration.
- Public entry point: `sqlite.Open(dataSourceName string) (*sql.DB, error)`.
- The caller owns `*sql.DB` lifetime and transactions. The adapter never begins,
  commits, or rolls back a transaction.
- SQLite uses the core question-mark placeholder dialect; no adapter-specific
  dialect hook is required for this first slice.

## First implementation slice

1. Add the smallest public adapter entry point for opening/configuring SQLite.
2. Validate `Open`, `Ping`, schema setup, SQL operations, and caller-owned
   transaction behavior against a real SQLite database.
3. Add real-database E2E tests for:
   - deterministic schema setup and cleanup;
   - Mapper scanning and value extraction;
   - `LoadContext` and Identity Map pointer reuse;
   - `Flush` inside an application-owned transaction;
   - commit and rollback behavior;
   - one numeric generated primary key;
   - simple and composite primary keys where SQLite supports the case;
   - question-mark placeholders;
   - missing rows;
   - actionable mapping and database errors.
4. Validate Go 1.25, 1.26, and 1.27 with `CGO_ENABLED=0`.
5. Document supported versions, DSNs, ownership, fixtures, transaction
   behavior, and exact commands used for tests.

## Generated-key contract

The core's published contract uses `sql.Result.LastInsertId` for a pending
insert with exactly one numeric primary-key field. It assigns the generated
value to the entity and registers the entity in the Identity Map. Unsupported
shapes or unavailable driver results return `sqlok.ErrGeneratedKeyUnsupported`.

The adapter does not access core internals and does not hide transaction
ownership.

## Explicit non-goals

- Do not add SQLite-specific imports to `github.com/candango/sqlok`.
- Do not implement the CGO/mattn adapter here.
- Do not add a global application cache or hide transaction ownership inside
  the adapter.
- Do not optimize based on a single microbenchmark; profile or measure the
  complete path first.

## Suggested shape

```text
.
├── AGENTS.md
├── README.md
├── go.mod
├── docs/
│   └── seed.md
├── adapter.go
├── adapter_test.go
└── integration/
    └── sqlite_test.go
```

## Completion evidence

Before closing the initial implementation tickets, show:

- the core dependency resolves from the published module proxy and is not a
  local workspace replacement;
- CI green on Go 1.25, 1.26, and 1.27 with `CGO_ENABLED=0`;
- `CGO_ENABLED=0 go test ./...` and `CGO_ENABLED=0 go vet ./...` output;
- the driver, core, Go, and SQLite fixture versions;
- the E2E schema and transaction cases covered;
- documentation updated with API, DSN, ownership, and behavior decisions;
- benchmark method and results if a performance claim is made.
