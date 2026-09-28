# lob `replay --emit` — binary trade stream (design)

**Status:** approved-by-delegation (user granted autopilot; review pending).
**Goal:** let `lob`'s replay produce a byte stream of trades that
`tickstore ingest` consumes, so `lob-replay --emit | tickstore ingest --symbol X`
works end-to-end — without either project importing the other.

## Why

`tickstore` (project B) reads a bare stream of fixed 25-byte trade records. `lob`
(project A) already computes every trade; it just never emits them in that wire
format, and its `trade.Trade` doesn't record which side crossed (the *aggressor*),
which `tickstore`'s record needs.

## The wire contract (shared, not shared code)

One trade encodes to **exactly 25 bytes, little-endian** — identical to
`tickstore`'s on-disk/record layout:

```
TS int64 | Price int64 | Qty uint64 | Side uint8      = 25 bytes
```

- `TS`   = the trade's logical timestamp (the taker order's `TS`).
- `Price`= maker (resting) price, integer ticks.
- `Qty`  = fill quantity.
- `Side` = **aggressor** side: `Buy=0`, `Sell=1` — the side of the incoming
  (taker) order that crossed the spread.

The stream is **bare records**: no header, no delimiters (a `tickstore` log adds
its own 8-byte header on ingest). `lob` and `tickstore` share only this 25-byte
contract — no import in either direction. `lob` re-declares the layout locally; a
golden hex fixture guards against drift.

`lob`'s `order.Side` is already `Buy=0, Sell=1`, so `Side` encodes as
`byte(aggressorSide)` directly. Trades are emitted in production (match) order,
which is non-decreasing in `TS` — satisfying `tickstore`'s sorted-log assumption.

## Changes

### 1. `trade.Trade` gains the aggressor side
Add `AggressorSide order.Side` to `trade.Trade` (package `trade` imports `order`;
no cycle — `order` imports nothing). `book.Add` sets `AggressorSide: o.Side` on
every emitted trade (the incoming order is the aggressor). The existing
`cmd/replay` golden fixture is regenerated to carry the new field.

### 2. `cmd/replay/emit.go` — the encoder (I/O layer only)
- `encodeTrade(t trade.Trade, buf []byte)` writes the 25 bytes, little-endian.
- A small streaming helper writes each trade to a `bufio.Writer`, flushed at end.
- Constant `recordSize = 25`. No import of `tickstore`.

### 3. `cmd/replay` `--emit` flag
- `--emit` (bool, default false). Behavior unchanged when off.
- When **on**: the raw binary trade stream goes to **stdout**; the human-readable
  summary (symbol/actions/trades, depth, throughput) is routed to **stderr**.
  Trades are streamed as they are produced; the binary writer is flushed before
  exit (including the `os.Exit(1)` add-error path).

## Error handling
- Library (`book`, `trade`) unchanged in contract: return errors, never panic /
  `os.Exit`. Only `cmd/replay` does I/O and may exit.
- A generator `Add` error mid-run: already-written whole records remain valid
  (matches `tickstore` ingest tolerating a clean record-boundary EOF); replay
  still exits non-zero as today.

## Testing
- **book:** update matching tests to assert `AggressorSide` on emitted trades.
- **golden:** regenerate the `cmd/replay` trade golden with `AggressorSide`.
- **emit:** `emit_test.go` — (a) round-trip: a test-only decoder reverses
  `encodeTrade` and recovers the trades, plus exact 25-byte length/offset checks;
  (b) golden hex: pin the exact bytes for a small fixed trade set covering both a
  Buy-aggressor and a Sell-aggressor trade.
- Gates: `go build ./... && go vet ./... && go test ./...` green (keyed struct
  literals so `go vet` stays clean).

## Out of scope (deferred)
Wall-clock timestamps, multi-symbol interleaving in one stream, a length-prefixed
or self-describing wire format, network transport. v1 stays a bare byte pipe.
