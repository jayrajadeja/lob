# lob `replay --emit` — implementation plan

Spec: `docs/superpowers/specs/2026-09-28-replay-emit-design.md`.
Execution: direct TDD by the controller (small, fully-specified change), then a
final whole-branch code review. Every commit ends with the
`Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>` trailer.

## Global constraints
- Integer ticks (`int64` price/TS, `uint64` qty); no floats.
- Little-endian; record = 25 bytes `{TS int64, Price int64, Qty uint64, Side uint8}`.
- `Side` = aggressor (taker) side; `Buy=0, Sell=1`.
- Layering unchanged: `order`/`trade` pure data; `book` matches; only `cmd/replay`
  does I/O. No import of `tickstore`.
- Library returns errors, never panics / `os.Exit` (only `cmd/replay` may exit).

## Task 1 — aggressor side on `trade.Trade`
- Add `AggressorSide order.Side` to `trade.Trade` (package `trade` imports
  `order`).
- `book.Add`: set `AggressorSide: o.Side` on the emitted `trade.Trade` literal.
- Update `book/matching_test.go` assertions to check `AggressorSide`.
- Regenerate the `cmd/replay/golden_test.go` fixture so every entry carries the
  correct `AggressorSide` (keyed literals; `go vet` clean).
- Gate: `go build ./... && go vet ./... && go test ./...` green.
- Commit: `feat: record aggressor side on each trade`.

## Task 2 — `cmd/replay/emit.go` encoder (TDD)
- Test first (`cmd/replay/emit_test.go`):
  - round-trip: a test-only decoder reverses `encodeTrade`, recovers the trades;
    assert exact 25-byte length and field offsets.
  - golden hex: pin exact bytes for a small fixed trade set with one Buy-aggressor
    and one Sell-aggressor trade.
- Implement `emit.go`: `const recordSize = 25`; `encodeTrade(t trade.Trade, buf
  []byte)` (little-endian, `buf[24]=byte(t.AggressorSide)`); a streaming helper
  `streamTrades(w io.Writer, trades []trade.Trade) error` using a `bufio.Writer`.
- Gate green. Commit: `feat: add binary trade-stream encoder for replay`.

## Task 3 — `--emit` flag wiring in `cmd/replay/main.go`
- Add `--emit` bool (default false). When off: unchanged.
- When on: summary output (symbol/actions/trades, depth, throughput) goes to
  `os.Stderr`; each produced trade is streamed as 25 bytes to `os.Stdout` via a
  `bufio.Writer`, flushed before exit (including the add-error `os.Exit(1)` path).
- Refactor summary prints to write to a selectable `io.Writer`.
- Manual/e2e smoke (documented): `go run ./cmd/replay --emit --orders 200 |
  tickstore ingest ...` round-trips (run against the built tickstore binary).
- Gate green. Commit: `feat: add replay --emit binary trade stream`.

## Final review
Whole-branch code review (capable model): layering intact, no `tickstore` import,
wire bytes match the 25-byte contract, `--emit` routing correct, flush on all exit
paths, golden regenerated correctly, tests assert real behavior. Then
finishing-a-development-branch → PR on `github.com/jayrajadeja/lob`.
