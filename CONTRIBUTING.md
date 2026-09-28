# Contributing to lob

Thanks for helping improve **lob** — a deterministic limit order book matching
engine that emits a trade stream.

## Prerequisites

- Go 1.26 or newer (see `go.mod`).
- Standard library only. **Do not add third-party dependencies** without discussion;
  keeping the module dependency-free is a design goal of this series.

## Local checks

Run the full gate from the repository root before opening a pull request:

```bash
go build ./... && go vet ./... && go test ./...
```

If you touched a hot path, run the benchmarks too:

```bash
go test ./... -run '^$' -bench . -benchmem
```

## Coding expectations

- **Test-first.** Add or update a failing test before the change that makes it pass.
  Every behavior change ships with a test that pins it.
- **Minimal, surgical changes.** One clean line over fifty. No speculative features
  (YAGNI), no drive-by refactors of unrelated code.
- **Keep the core pure.** The library packages return errors and never call
  `os.Exit`, print, or panic on bad input. All I/O (argv/stdin/stdout/files) lives in
  the `cmd/` leaf package only.
- **Integers, not floats.** Prices are `int64` tick counts and quantities `uint64`.
  Never introduce floating-point into price or size arithmetic.
- **Determinism.** Same seed ⇒ byte-identical output. Don't add wall-clock time,
  map-iteration ordering, or unseeded randomness to the core.
- **Match existing style.** Follow the patterns already in the file you're editing.
- Update documentation when behavior, flags, or the wire format change.
- Never commit secrets.

## Pull-request checklist

- `go build ./... && go vet ./... && go test ./...` is green.
- New behavior is covered by tests (golden-file tests for engine output).
- Docs (README, specs) reflect the change.
- No unrelated files are modified.

## Design docs

Specs and implementation plans live under `docs/superpowers/`. For a non-trivial
change, add or update the relevant spec before implementing.
