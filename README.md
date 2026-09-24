# sqlok-sqlite-modernc

Pure-Go SQLite adapter for [`github.com/candango/sqlok`](https://github.com/candango/sqlok), using [`modernc.org/sqlite`](https://modernc.org/sqlite).

## Status

The adapter entry point and real SQLite E2E contract are implemented locally.
The adapter consumes the published `sqlok` generated-key contract from core
commit `18e5570` through this module version:

```text
github.com/candango/sqlok v0.0.0-20260924024447-18e5570f637c
```

## Public API

`sqlite.Open` opens a `database/sql` handle through modernc's registered
`sqlite` driver:

```go
package main

import (
	"context"
	"log"

	sqlok "github.com/candango/sqlok"
	sqlite "github.com/candango/sqlok-sqlite-modernc"
)

func main() {
	db, err := sqlite.Open("file:app.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		log.Fatal(err)
	}

	session := sqlok.NewSession(db)
	_ = session
}
```

The caller owns the returned `*sql.DB` lifetime and every transaction. The
adapter never begins, commits, or rolls back a transaction. `Flush` receives a
caller-owned `*sql.Tx`, and the caller decides whether to commit or roll back.

SQLite uses question-mark placeholders. Use a file DSN for persistent data or a
unique `file:<name>?mode=memory&cache=shared` DSN for an isolated in-memory
fixture. The E2E suite uses one temporary SQLite file per test, creates its
schema explicitly, and removes it through `testing.T.TempDir` cleanup.

## Supported versions

| Component | Supported value |
|---|---|
| Go | 1.25, 1.26, 1.27 |
| CGO | `CGO_ENABLED=0` |
| Driver | `modernc.org/sqlite` v1.59.0 |
| Core | `github.com/candango/sqlok` pseudo-version at commit `18e5570` |
| SQLite fixture | Temporary file database with `users`, `pairs`, and `invalid_users` tables |

Generated-key support follows the core contract: one numeric generated primary
key is read through `sql.Result.LastInsertId`, assigned to the entity, and
registered in the Session Identity Map. Unsupported generated-key shapes return
`sqlok.ErrGeneratedKeyUnsupported`.

## Validation

Run the configured checks with pure-Go mode enabled:

```bash
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go vet ./...
```

The GitHub Actions matrix repeats these checks on Go 1.25, 1.26, and 1.27.

The E2E suite covers Mapper scanning and value extraction, `LoadContext`,
Identity Map pointer reuse, caller-owned `Flush`, commit, rollback, generated
keys, simple and composite primary keys, question-mark placeholders, missing
rows, and actionable database/mapping errors.

No performance improvement is claimed without a reproducible benchmark.

See [`AGENTS.md`](AGENTS.md) and [`docs/seed.md`](docs/seed.md) for repository
boundaries and implementation evidence requirements.
