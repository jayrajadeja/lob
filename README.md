# lob — a limit order book matching engine (Go)

A from-scratch, deterministic single-symbol limit order book with price-time
priority matching, driven by a synthetic order generator. First project in an
A → B → C series (matching engine → tick store → storage engine).

## Run

    go run ./cmd/replay -seed 1 -orders 10000

Flags: `-seed -orders -symbol -mid -tick -maxqty -cancelprob`.

## Test

    go test ./...

## Design

See `docs/superpowers/specs/2026-09-25-limit-order-book-matching-engine-design.md`.

## Layout

- `order/`  — Order and Side data types
- `trade/`  — Trade output type
- `book/`   — matching engine core (pure, deterministic)
- `gen/`    — synthetic seedable order-flow generator
- `cmd/replay/` — replay CLI (the only I/O layer)
