# Agent instructions

## Project mission

`sqlok-sqlite-modernc` is the pure-Go SQLite adapter for
`github.com/candango/sqlok`. It owns the SQLite driver dependency and the
real-database end-to-end tests; the `sqlok` core remains driver-agnostic.

## Non-negotiable boundaries

- Keep the adapter pure Go and validate it with `CGO_ENABLED=0`.
- Pin `modernc.org/sqlite` in this module; do not import or register the driver
  from the `sqlok` core repository.
- Keep this adapter separate from `sqlok-sqlite-mattn`; applications select one
  SQLite adapter and must not register both drivers accidentally.
- Keep compiler plans, caches, and bind buffers behind the public `sqlok` API.
- Do not claim performance improvements without a reproducible benchmark.

## First milestone

Implement the adapter structure and a real SQLite E2E suite covering the
contract described in `docs/seed.md`. The first implementation should be small,
explicit, and easy to extend; do not introduce speculative interfaces or a
framework layer.

## Validation baseline

The CI matrix must remain green on Go 1.25, 1.26, and 1.27. Every
matrix entry runs with `CGO_ENABLED=0`. Run the repository's configured checks
plus, at minimum:

```bash
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go vet ./...
```

Format Go code with `gofmt`; use `goimports` when it is available. Record the
SQLite version, Go version, CGO setting, schema fixture, and benchmark method
in documentation when reporting results.

## Workflow and repository operations

- Read `docs/seed.md` before implementation and update it when the contract
  changes.
- Use `tw-flow` or `taskp`, never raw `task`, for workflow state.
- Use `jacazul-broker` for GitHub operations; never handle raw tokens.
- Stage only task-relevant files. Commits and pushes require explicit operator
  authorization.
- Keep persistent task notes and documentation in English; communicate with
  the operator in the session's language.
