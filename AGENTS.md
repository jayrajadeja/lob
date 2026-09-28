# AGENTS.md — lob

Guidance for AI coding agents working in this repository. Human contributors: see
[`CONTRIBUTING.md`](CONTRIBUTING.md).

**What this is:** a deterministic, single-symbol limit order book matching engine
(price-time priority) with a seedable synthetic order generator, emitting a 25-byte
trade stream. Pure Go, standard library only. **Project A** in the
`lob → tickstore → candle` series.

## Commands you must run

```bash
go build ./...            # compile
go build -o lob-replay ./cmd/replay
go vet ./...              # static checks
go test ./...             # full test suite (incl. golden-file engine tests)
go test ./... -bench .    # benchmarks (when touching a hot path)
```

**Never claim a change is done until `go build ./... && go vet ./... && go test ./...`
passes.** Show the output; don't assert.

## Non-negotiable conventions

- **Integers only.** Prices are `int64` tick counts, quantities `uint64`. No
  floating-point in price/size math — ever. Float rounding in a comparison is how
  real venues ship mismatched fills.
- **Determinism.** `TS` is a logical sequence counter, not a wall clock; the
  generator is seeded. Same seed ⇒ byte-identical run — this is what the golden-file
  tests rely on. Do not introduce wall-clock time, unseeded randomness, or
  nondeterministic map ordering into the core.
- **Price-time priority.** Best price first; FIFO within a price level. Don't change
  matching semantics without updating the golden files and the spec.
- **I/O is isolated.** `order`/`trade`/`book`/`gen` are pure: they return errors and
  never `os.Exit`, print, or panic. Only `cmd/replay` touches argv/stdin/stdout.
- **The 25-byte record is a contract.** `--emit` writes a bare little-endian
  `TS int64 | Price int64 | Qty uint64 | Side uint8` (25 bytes) stream consumed by
  `tickstore ingest` and `candle`. Do not change its layout without updating the
  whole series.
- **Minimal, surgical diffs.** No speculative features, no unrelated refactors. Keep
  every existing safety guard.
- **Test-driven.** Write the failing test first. Pin every behavior change with a test.

## Workflow

- Design specs and implementation plans live under `docs/superpowers/`
  (`specs/` and `plans/`). Read the relevant spec before a non-trivial change.
- **Commits:** include the trailer
  `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`.
  Git identity is configured at the machine level — do not hard-code author info.
- **Pull requests:** open a PR and let a human merge it. **Never self-merge.**

## Layout

| package | job |
|---------|-----|
| `order/` | `Order` and `Side` data types |
| `trade/` | `Trade` output type + 25-byte encode/decode |
| `book/`  | matching engine core (pure, deterministic) |
| `gen/`   | synthetic, seedable order-flow generator |
| `cmd/replay/` | replay CLI — the only I/O layer (incl. `--emit`) |
