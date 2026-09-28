# lob — a limit order book matching engine (Go)

A from-scratch, deterministic, single-symbol limit order book with **price-time
priority** matching, driven by a synthetic order-flow generator. Pure Go, standard
library only, no floats in the hot path.

This is **project A** in a four-part series that builds a tiny market-data stack
end to end:

```
A  lob        generate   — match orders, emit a trade stream   (this repo)
B  tickstore  store      — append-only, indexed tick store
C  candle     aggregate  — OHLCV candles over the tick stream
```

The pieces share nothing but a **25-byte trade record** on a Unix pipe:

```bash
lob-replay --emit | tickstore ingest --symbol SYNTH
lob-replay --emit | candle --width 20
```

## Build

```bash
go build ./...
go build -o lob-replay ./cmd/replay   # the CLI binary used in the pipe
```

Requires Go 1.26+ (see `go.mod`). No third-party dependencies.

## Run

```bash
go run ./cmd/replay -seed 1 -orders 10000
```

Flags:

| flag | default | meaning |
|------|---------|---------|
| `-seed` | `1` | RNG seed — same seed ⇒ byte-identical run |
| `-orders` | `10000` | number of synthetic orders to generate |
| `-symbol` | `SYNTH` | symbol label |
| `-mid` | — | starting mid price (integer ticks) |
| `-tick` | — | tick size |
| `-maxqty` | — | max order quantity |
| `-cancelprob` | — | probability an order is a cancel |
| `--emit` | off | write the raw 25-byte trade stream to stdout (feeds B/C) |

## How it works

- **Prices and quantities are integers.** Prices are `int64` tick counts, quantities
  `uint64`. There are **no floats anywhere** — float rounding in a price comparison
  is how real venues have shipped mismatched fills.
- **Deterministic.** `TS` is a logical sequence counter, not a wall clock, so a
  given seed replays bit-for-bit. This is what makes golden-file tests possible.
- **Price-time priority.** Best price first; within a price level, earliest order
  first (FIFO).
- **I/O lives in exactly one package.** The `book`/`order`/`trade`/`gen` cores are
  pure and return errors; only `cmd/replay` touches argv/stdin/stdout.

## Layout

| package | job |
|---------|-----|
| `order/` | `Order` and `Side` data types |
| `trade/` | `Trade` output type + 25-byte encode/decode |
| `book/`  | matching engine core (pure, deterministic) |
| `gen/`   | synthetic, seedable order-flow generator |
| `cmd/replay/` | replay CLI — the only I/O layer (incl. `--emit`) |

## Development

```bash
go build ./... && go vet ./... && go test ./...
```

Design specs and implementation plans live under `docs/superpowers/`.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). Agent contributors: read
[`AGENTS.md`](AGENTS.md).

## License

MIT — see [`LICENSE`](LICENSE).
