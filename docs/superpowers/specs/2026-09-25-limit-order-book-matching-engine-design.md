# Design — Limit Order Book Matching Engine (v1)

Status: Approved
Date: 2026-09-25
Author: Jayraj Jadeja

## Summary

A from-scratch **limit order book (LOB) matching engine** in Go. v1 is a
headless, deterministic library for a single symbol with price-time priority
matching, fed by a synthetic order-flow generator and exercised by a small
replay CLI. It is the first project in an A → B → C series:

- **A (this spec):** matching engine — produces a stream of trades/ticks.
- **B (future):** tick store — a time-series storage engine that persists and
  serves the trade stream.
- **C (future):** KV / LSM storage engine — the substrate B can sit on.

Goals: deep systems learning, a tool to build on, and content material.

## Goals (v1)

- Correct single-symbol limit order book with **price-time priority**.
- Operations: `Add` (limit order), `Cancel`, and a matching loop emitting trades.
- Deterministic and pure: same input → same trades, no wall-clock, no I/O in
  the core.
- Synthetic, **seedable** order generator (reproducible streams; doubles as a
  load generator).
- Replay CLI that runs generator → engine and prints trades, final book depth,
  and a throughput stat.
- Thorough unit tests + a golden test pinning the exact trade sequence.

## Non-goals (v1)

- Market orders (arrives in v2).
- Interactive CLI REPL / network server (v2 = `b + ii`).
- Multiple symbols.
- Persistence, database, message broker, Docker, cloud deployment (see
  Future work).
- A high-performance price index — v1 uses a deliberately simple sorted-slice
  index (documented O(n) worst case); a heap/tree is a benchmarked follow-up.

## Language & conventions

- **Go.** Module `github.com/jayrajadeja/lob`.
- Idiomatic Go: **structs with methods** (no "classes"; Go has none). Encapsulate
  state with unexported fields.
- Separation of concerns (the sound instinct behind "MVC"), mapped to Go:
  - **Model / data:** `order`, `trade` packages (pure structs, no logic).
  - **Engine / logic:** `book` package (the matching core).
  - **I/O / wiring:** `cmd/replay` (the only package that does I/O).
- No forced MVC scaffolding in the headless v1. A controller-style input loop
  appears naturally in v2 (interactive REPL).

## Package layout

```
lob/                        module: github.com/jayrajadeja/lob
├── cmd/replay/main.go      wire generator → engine → printer; flags -seed -orders -symbol
├── order/order.go          Order, Side, enums (pure data, no deps)
├── book/book.go            OrderBook: Add, Cancel, Match; emits []Trade
├── book/pricelevel.go      one price level = FIFO queue of resting orders
├── trade/trade.go          Trade (maker, taker, price, qty, ts)
├── gen/gen.go              synthetic seedable order-flow generator
└── (tests alongside each package)
```

Responsibilities:

- `order` — plain structs/enums, no dependencies.
- `book` — owns the two-sided book, runs price-time priority matching, returns
  trades. Pure and deterministic; timestamps are passed in, never read from a
  clock inside matching.
- `gen` — produces a reproducible order stream from a seed; knows nothing about
  book internals.
- `trade` — the output type the book emits and (later) B ingests.
- `cmd/replay` — the only place that does I/O.

**Key principle:** `book` is a pure library — `engine := book.New(symbol)`, feed
orders, get trades. That single clean interface is what makes the future CLI,
network server, and tick store (B) trivial to add without touching the core.

## Data model

```go
type Side uint8            // Buy, Sell

type Order struct {
    ID    uint64
    Side  Side
    Price int64            // integer ticks, NOT float
    Qty   uint64           // remaining quantity
    TS    int64            // sequence number / logical time, passed in
}

type Trade struct {
    MakerID uint64
    TakerID uint64
    Price   int64
    Qty     uint64
    TS      int64
}
```

**Integer prices (correctness guard):** floats break exact price comparison and
summation (`0.1 + 0.2`). Real exchanges use integer ticks. Price `10025` =
$100.25 at a tick of 0.01.

## Order book structure

- Two sides: **bids** (buy) and **asks** (sell).
- Each side: a map `price → *PriceLevel` plus an ordered index of prices to find
  the best bid/ask.
- `PriceLevel` = a **FIFO queue** of resting orders at that price → enforces
  **time** priority.
- Best bid = highest price; best ask = lowest price → **price** priority.
- **v1 index:** a **sorted slice** of price levels (simple, correct, readable).
  Documented O(n) worst-case insert; heap/tree is a benchmarked follow-up (and a
  content post in its own right).

## Matching algorithm (price-time priority)

On an incoming order:

1. Look at the opposite side's best price.
2. While the incoming order still has qty **and** the best opposite price
   crosses (buy price ≥ best ask, or sell price ≤ best bid):
   - Match against the **front** resting order in that level (time priority).
   - Emit a `Trade` at the **resting** order's price (standard maker-price rule).
   - Decrement both quantities; pop the resting order if fully filled; remove the
     price level if it becomes empty.
3. When no more crossing or qty is exhausted: if qty remains, **rest** the
   incoming order on its own side.

`Cancel(id)` removes a resting order by ID (and prunes an emptied level).

**Determinism:** no wall-clock; no reliance on Go map iteration order inside
matching; `TS` is a passed-in sequence number. Same seed → same trades every run.

## Synthetic generator

`gen.Generator` (seeded `math/rand`) emits a reproducible order stream.

- Params: seed, order count, mid-price, tick size, max qty, cancel probability.
- Behavior: prices random-walk around a mid; each step emits a limit buy/sell
  near the mid, occasionally a `Cancel` of a live order ID.
- Same seed → identical stream. Pure data out (`[]Order` or channel); knows
  nothing about the book. Doubles as a load generator for throughput numbers.

## Replay binary

`cmd/replay` with flags `-seed -orders -symbol`:

- Builds a generator, feeds orders into the book.
- Prints the trade sequence, final book depth (top N levels each side), and a
  one-line throughput stat (orders/sec).
- The only I/O in the project.

## Error handling

- `Cancel` of an unknown/already-filled ID: no-op returning a boolean/typed
  "not found" — never a panic.
- Reject malformed orders (zero qty, non-positive price) at the `book` boundary
  with a typed error; the core never panics on bad input.
- `cmd/replay` surfaces errors to stderr and exits non-zero; the library returns
  errors, never calls `os.Exit`.

## Testing

- **Unit tests** per package:
  - Matching correctness: price-time priority, partial fills, full-level sweep,
    cross-then-rest, cancel-then-no-match.
  - Integer-price edge cases (touching vs crossing).
  - `Cancel` of unknown/filled IDs.
  - Generator determinism: same seed → identical output.
- **Golden test:** a fixed seed → assert the exact emitted trade sequence, so a
  refactor cannot silently change matching behavior.

## Future work / deployment (NOT v1)

Captured so nothing is lost; the v1 boundaries are chosen to make these additive,
not rewrites.

- **v2:** market orders + interactive CLI REPL (`b + ii`).
- **Real data sources:** replay free public crypto feeds (Binance/Coinbase L2 +
  trades); parse & replay Nasdaq **ITCH** sample files (real order-by-order flow).
- **Services:** Dockerfile + docker-compose; publish trades to **RabbitMQ**;
  persist to a **DB** and/or the tick store (**B**).
- **Cloud:** deploy on **AWS** (local-first now, cloud later).
- **Performance:** swap the sorted-slice index for a heap/tree; benchmark and
  write it up.

Because `book` emits a clean `Trade` stream and `gen` is a swappable source,
each of these can be added without changing the matching core.

## Success criteria (v1)

- `go test ./...` green, including the golden test.
- `go run ./cmd/replay -seed 1 -orders 10000` runs deterministically and prints
  trades, final depth, and throughput.
- The `book` package has no imports from `cmd`, `gen`, or any I/O package.
