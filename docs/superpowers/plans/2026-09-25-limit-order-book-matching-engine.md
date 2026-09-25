# Limit Order Book Matching Engine (v1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a from-scratch, deterministic single-symbol limit order book matching engine in Go, fed by a synthetic order generator and exercised by a replay CLI.

**Architecture:** Small focused packages. `order`/`trade` are pure data. `book` is the deterministic matching core (no I/O, no clock). `gen` produces a reproducible action stream. `cmd/replay` is the only I/O layer, wiring generator → engine → printer.

**Tech Stack:** Go 1.26 (stdlib only — `math/rand`, `sort`, `flag`, `testing`). Module `github.com/jayrajadeja/lob`.

## Global Constraints

Every task's requirements implicitly include these:

- Module path: `github.com/jayrajadeja/lob`; `go 1.26` in `go.mod`.
- **Prices are integer ticks (`int64`), never floats.** Quantities are `uint64`.
- **Deterministic core:** timestamps (`TS`) are passed in, never read from a wall clock inside matching; matching must not depend on Go map iteration order. Same seed → same trades every run.
- **`book` must not import `cmd`, `gen`, or any I/O package.** It may import only `order` and `trade`.
- Idiomatic Go: structs with methods, unexported fields for encapsulation.
- The library returns errors; it never calls `os.Exit` and never panics on bad input. Only `cmd/replay` does I/O and process exit.
- A trade executes at the **resting (maker) order's price**.
- **Price-time priority:** best price first; FIFO within a price level.
- Stdlib only — no third-party dependencies in v1.

---

### Task 1: Data types (`order`, `trade`) + module init

**Files:**
- Create: `go.mod`
- Create: `order/order.go`
- Create: `trade/trade.go`
- Test: `order/order_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `order.Side` (`uint8`) with constants `order.Buy`, `order.Sell` and `func (Side) String() string`.
  - `order.Order struct { ID uint64; Side Side; Price int64; Qty uint64; TS int64 }` with `func (Order) Validate() error`.
  - Sentinel errors `order.ErrInvalidQty`, `order.ErrInvalidPrice`.
  - `trade.Trade struct { MakerID uint64; TakerID uint64; Price int64; Qty uint64; TS int64 }`.

- [ ] **Step 1: Initialize the module**

Run:
```bash
cd ~/Personal/lob && go mod init github.com/jayrajadeja/lob
```
Expected: creates `go.mod` containing `module github.com/jayrajadeja/lob` and a `go 1.26` line.

- [ ] **Step 2: Write the failing test**

Create `order/order_test.go`:
```go
package order

import (
	"errors"
	"testing"
)

func TestSideString(t *testing.T) {
	if Buy.String() != "BUY" {
		t.Errorf("Buy.String() = %q, want BUY", Buy.String())
	}
	if Sell.String() != "SELL" {
		t.Errorf("Sell.String() = %q, want SELL", Sell.String())
	}
}

func TestOrderValidate(t *testing.T) {
	tests := []struct {
		name string
		o    Order
		want error
	}{
		{"ok", Order{ID: 1, Side: Buy, Price: 100, Qty: 5, TS: 1}, nil},
		{"zero qty", Order{ID: 2, Side: Buy, Price: 100, Qty: 0, TS: 1}, ErrInvalidQty},
		{"zero price", Order{ID: 3, Side: Sell, Price: 0, Qty: 5, TS: 1}, ErrInvalidPrice},
		{"negative price", Order{ID: 4, Side: Sell, Price: -1, Qty: 5, TS: 1}, ErrInvalidPrice},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.Validate(); !errors.Is(got, tt.want) {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd ~/Personal/lob && go test ./order/`
Expected: FAIL — build error, `Buy`/`Sell`/`Order`/`Validate` undefined.

- [ ] **Step 4: Write minimal implementation**

Create `order/order.go`:
```go
// Package order defines the pure data types for the matching engine's input:
// an Order and which Side of the book it belongs to. No engine logic lives here.
package order

import "errors"

// Side is the side of the book an order sits on.
type Side uint8

const (
	Buy Side = iota
	Sell
)

func (s Side) String() string {
	switch s {
	case Buy:
		return "BUY"
	case Sell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}

// Sentinel validation errors returned by Order.Validate.
var (
	ErrInvalidQty   = errors.New("order: quantity must be > 0")
	ErrInvalidPrice = errors.New("order: price must be > 0")
)

// Order is a single limit order. Price is an integer tick count (never a float);
// TS is a caller-supplied logical timestamp / sequence number, so the engine
// stays deterministic and free of wall-clock reads.
type Order struct {
	ID    uint64
	Side  Side
	Price int64
	Qty   uint64
	TS    int64
}

// Validate reports whether the order is well-formed.
func (o Order) Validate() error {
	if o.Qty == 0 {
		return ErrInvalidQty
	}
	if o.Price <= 0 {
		return ErrInvalidPrice
	}
	return nil
}
```

Create `trade/trade.go`:
```go
// Package trade defines the Trade type the matching engine emits when two
// orders match. It is the output stream later projects (the tick store) consume.
package trade

// Trade is a single execution: TakerID crossed the book and matched against the
// resting MakerID. Price is always the maker (resting) order's price.
type Trade struct {
	MakerID uint64
	TakerID uint64
	Price   int64
	Qty     uint64
	TS      int64
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd ~/Personal/lob && go test ./order/ ./trade/`
Expected: PASS (`ok` for `order`; `trade` reports no test files, which is fine).

- [ ] **Step 6: Commit**

```bash
cd ~/Personal/lob
git add go.mod order/ trade/
git commit -m "feat: add order and trade data types

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 2: Price level (FIFO queue)

**Files:**
- Create: `book/pricelevel.go`
- Test: `book/pricelevel_test.go`

**Interfaces:**
- Consumes: `order.Order` from Task 1.
- Produces (unexported, used by Task 3 & 4 within package `book`):
  - `type priceLevel struct { price int64; orders []order.Order }`
  - `func newPriceLevel(price int64) *priceLevel`
  - `func (l *priceLevel) push(o order.Order)` — append to back (FIFO tail).
  - `func (l *priceLevel) front() (order.Order, bool)` — read head; false if empty.
  - `func (l *priceLevel) reduceFront(qty uint64)` — subtract `qty` from head; pop head if it reaches 0. Caller guarantees `qty <= front.Qty`.
  - `func (l *priceLevel) removeByID(id uint64) bool` — remove first order with `id`, preserving FIFO order; false if absent.
  - `func (l *priceLevel) totalQty() uint64`
  - `func (l *priceLevel) isEmpty() bool`

- [ ] **Step 1: Write the failing test**

Create `book/pricelevel_test.go`:
```go
package book

import (
	"testing"

	"github.com/jayrajadeja/lob/order"
)

func ord(id uint64, qty uint64) order.Order {
	return order.Order{ID: id, Side: order.Buy, Price: 100, Qty: qty, TS: int64(id)}
}

func TestPriceLevelPushFrontFIFO(t *testing.T) {
	l := newPriceLevel(100)
	if !l.isEmpty() {
		t.Fatal("new level should be empty")
	}
	l.push(ord(1, 5))
	l.push(ord(2, 3))
	f, ok := l.front()
	if !ok || f.ID != 1 {
		t.Fatalf("front = %+v ok=%v, want ID 1", f, ok)
	}
	if l.totalQty() != 8 {
		t.Errorf("totalQty = %d, want 8", l.totalQty())
	}
}

func TestPriceLevelReduceFront(t *testing.T) {
	l := newPriceLevel(100)
	l.push(ord(1, 5))
	l.push(ord(2, 3))

	l.reduceFront(2) // 1 now has 3 left
	f, _ := l.front()
	if f.ID != 1 || f.Qty != 3 {
		t.Fatalf("front = %+v, want ID 1 Qty 3", f)
	}

	l.reduceFront(3) // 1 fully filled -> popped, front becomes 2
	f, ok := l.front()
	if !ok || f.ID != 2 {
		t.Fatalf("front = %+v ok=%v, want ID 2", f, ok)
	}
	if l.totalQty() != 3 {
		t.Errorf("totalQty = %d, want 3", l.totalQty())
	}
}

func TestPriceLevelRemoveByID(t *testing.T) {
	l := newPriceLevel(100)
	l.push(ord(1, 5))
	l.push(ord(2, 3))
	l.push(ord(3, 7))

	if !l.removeByID(2) {
		t.Fatal("removeByID(2) = false, want true")
	}
	if l.removeByID(99) {
		t.Fatal("removeByID(99) = true, want false")
	}
	// Remaining FIFO order should be 1 then 3.
	f, _ := l.front()
	if f.ID != 1 {
		t.Errorf("front = %d, want 1", f.ID)
	}
	if l.totalQty() != 12 {
		t.Errorf("totalQty = %d, want 12", l.totalQty())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd ~/Personal/lob && go test ./book/`
Expected: FAIL — `newPriceLevel`, `priceLevel`, and its methods undefined.

- [ ] **Step 3: Write minimal implementation**

Create `book/pricelevel.go`:
```go
package book

import "github.com/jayrajadeja/lob/order"

// priceLevel is the set of resting orders at one price, held as a FIFO queue:
// the head (index 0) is the oldest order and matches first, enforcing time
// priority within a price.
type priceLevel struct {
	price  int64
	orders []order.Order
}

func newPriceLevel(price int64) *priceLevel {
	return &priceLevel{price: price}
}

// push adds an order to the back of the queue.
func (l *priceLevel) push(o order.Order) {
	l.orders = append(l.orders, o)
}

// front returns the head order without removing it; ok is false if empty.
func (l *priceLevel) front() (order.Order, bool) {
	if len(l.orders) == 0 {
		return order.Order{}, false
	}
	return l.orders[0], true
}

// reduceFront subtracts qty from the head order, popping it if it reaches zero.
// The caller guarantees qty <= head quantity.
func (l *priceLevel) reduceFront(qty uint64) {
	if len(l.orders) == 0 {
		return
	}
	l.orders[0].Qty -= qty
	if l.orders[0].Qty == 0 {
		l.orders = l.orders[1:]
	}
}

// removeByID removes the first order with the given id, preserving FIFO order.
// It reports whether an order was removed.
func (l *priceLevel) removeByID(id uint64) bool {
	for i := range l.orders {
		if l.orders[i].ID == id {
			l.orders = append(l.orders[:i], l.orders[i+1:]...)
			return true
		}
	}
	return false
}

func (l *priceLevel) totalQty() uint64 {
	var sum uint64
	for i := range l.orders {
		sum += l.orders[i].Qty
	}
	return sum
}

func (l *priceLevel) isEmpty() bool {
	return len(l.orders) == 0
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd ~/Personal/lob && go test ./book/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd ~/Personal/lob
git add book/pricelevel.go book/pricelevel_test.go
git commit -m "feat: add FIFO price level to order book

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 3: Order book structure — resting, best prices, cancel, depth

**Files:**
- Create: `book/book.go`
- Test: `book/book_test.go`

**Interfaces:**
- Consumes: `priceLevel` (Task 2), `order.Order` (Task 1).
- Produces:
  - `type Level struct { Price int64; Qty uint64 }`
  - `func New(symbol string) *OrderBook`
  - `func (b *OrderBook) Cancel(id uint64) bool` — remove a resting order by ID; false if unknown.
  - `func (b *OrderBook) BestBid() (int64, bool)` — highest bid price; false if no bids.
  - `func (b *OrderBook) BestAsk() (int64, bool)` — lowest ask price; false if no asks.
  - `func (b *OrderBook) Depth(n int) (bids []Level, asks []Level)` — up to `n` levels per side, best first.
  - Unexported for Task 4: `func (b *OrderBook) rest(o order.Order)` — place a resting order and index it; `type side` with `bestLevel()`, `levelAt(price, create)`, `dropEmpty(price)`; `sideFor(order.Side)`; `locs map[uint64]orderLoc` bookkeeping.

**Design notes for the implementer:**
- Each `side` keeps `levels []*priceLevel` sorted **ascending by price**. Best bid = last element (highest); best ask = first element (lowest).
- `OrderBook.locs map[uint64]orderLoc` maps order ID → `{side *side, price int64}` so `Cancel` is a direct lookup. This map is used only for cancel/removal, never iterated during matching, so it does not affect determinism.

- [ ] **Step 1: Write the failing test**

Create `book/book_test.go`:
```go
package book

import (
	"testing"

	"github.com/jayrajadeja/lob/order"
)

func limit(id uint64, side order.Side, price int64, qty uint64) order.Order {
	return order.Order{ID: id, Side: side, Price: price, Qty: qty, TS: int64(id)}
}

func TestRestAndBestPrices(t *testing.T) {
	b := New("TEST")
	b.rest(limit(1, order.Buy, 100, 5))
	b.rest(limit(2, order.Buy, 101, 3))
	b.rest(limit(3, order.Sell, 105, 4))
	b.rest(limit(4, order.Sell, 104, 2))

	if bid, ok := b.BestBid(); !ok || bid != 101 {
		t.Errorf("BestBid = %d ok=%v, want 101", bid, ok)
	}
	if ask, ok := b.BestAsk(); !ok || ask != 104 {
		t.Errorf("BestAsk = %d ok=%v, want 104", ask, ok)
	}
}

func TestEmptyBookBestPrices(t *testing.T) {
	b := New("TEST")
	if _, ok := b.BestBid(); ok {
		t.Error("BestBid on empty book should be false")
	}
	if _, ok := b.BestAsk(); ok {
		t.Error("BestAsk on empty book should be false")
	}
}

func TestCancel(t *testing.T) {
	b := New("TEST")
	b.rest(limit(1, order.Buy, 100, 5))
	b.rest(limit(2, order.Buy, 100, 3))

	if !b.Cancel(1) {
		t.Fatal("Cancel(1) = false, want true")
	}
	if b.Cancel(99) {
		t.Fatal("Cancel(99) = true, want false")
	}
	bids, _ := b.Depth(10)
	if len(bids) != 1 || bids[0].Qty != 3 {
		t.Fatalf("bids = %+v, want one level qty 3", bids)
	}

	// Cancelling the last order at a price removes the level entirely.
	if !b.Cancel(2) {
		t.Fatal("Cancel(2) = false, want true")
	}
	if _, ok := b.BestBid(); ok {
		t.Error("book should have no bids after cancelling all")
	}
}

func TestDepthOrdering(t *testing.T) {
	b := New("TEST")
	b.rest(limit(1, order.Buy, 100, 5))
	b.rest(limit(2, order.Buy, 102, 1))
	b.rest(limit(3, order.Buy, 101, 2))
	b.rest(limit(4, order.Sell, 105, 4))
	b.rest(limit(5, order.Sell, 103, 6))

	bids, asks := b.Depth(2)
	// Bids: best (highest) first -> 102, 101.
	if len(bids) != 2 || bids[0].Price != 102 || bids[1].Price != 101 {
		t.Errorf("bids = %+v, want prices 102,101", bids)
	}
	// Asks: best (lowest) first -> 103, 105.
	if len(asks) != 2 || asks[0].Price != 103 || asks[1].Price != 105 {
		t.Errorf("asks = %+v, want prices 103,105", asks)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd ~/Personal/lob && go test ./book/ -run 'Rest|Best|Cancel|Depth'`
Expected: FAIL — `New`, `rest`, `BestBid`, `BestAsk`, `Cancel`, `Depth` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `book/book.go`:
```go
// Package book is the matching engine core: a two-sided limit order book with
// price-time priority. It is pure and deterministic — no I/O, no wall clock —
// and imports only the order and trade data packages.
package book

import (
	"sort"

	"github.com/jayrajadeja/lob/order"
)

// Level is a snapshot of resting quantity at one price, used by Depth.
type Level struct {
	Price int64
	Qty   uint64
}

// side holds one side of the book. levels is sorted ascending by price, so the
// highest price is last and the lowest is first; bestLevel picks the correct end.
type side struct {
	isBid  bool
	levels []*priceLevel
}

// bestLevel returns the best-priced non-empty level, or nil if the side is empty.
func (s *side) bestLevel() *priceLevel {
	if len(s.levels) == 0 {
		return nil
	}
	if s.isBid {
		return s.levels[len(s.levels)-1] // highest bid
	}
	return s.levels[0] // lowest ask
}

// levelAt returns the level at price, creating and inserting it (keeping levels
// sorted ascending) when create is true. Returns nil if absent and !create.
func (s *side) levelAt(price int64, create bool) *priceLevel {
	i := sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price >= price })
	if i < len(s.levels) && s.levels[i].price == price {
		return s.levels[i]
	}
	if !create {
		return nil
	}
	lvl := newPriceLevel(price)
	s.levels = append(s.levels, nil)
	copy(s.levels[i+1:], s.levels[i:])
	s.levels[i] = lvl
	return lvl
}

// dropEmpty removes the level at price if it has no orders left.
func (s *side) dropEmpty(price int64) {
	i := sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price >= price })
	if i < len(s.levels) && s.levels[i].price == price && s.levels[i].isEmpty() {
		s.levels = append(s.levels[:i], s.levels[i+1:]...)
	}
}

// orderLoc records where a resting order lives, so Cancel is a direct lookup.
type orderLoc struct {
	side  *side
	price int64
}

// OrderBook is a single-symbol limit order book.
type OrderBook struct {
	symbol string
	bids   *side
	asks   *side
	locs   map[uint64]orderLoc
}

// New creates an empty order book for the given symbol.
func New(symbol string) *OrderBook {
	return &OrderBook{
		symbol: symbol,
		bids:   &side{isBid: true},
		asks:   &side{isBid: false},
		locs:   make(map[uint64]orderLoc),
	}
}

// sideFor returns the resting side for an order's own side.
func (b *OrderBook) sideFor(s order.Side) *side {
	if s == order.Buy {
		return b.bids
	}
	return b.asks
}

// rest places o as a resting order on its own side and indexes it for cancel.
func (b *OrderBook) rest(o order.Order) {
	s := b.sideFor(o.Side)
	s.levelAt(o.Price, true).push(o)
	b.locs[o.ID] = orderLoc{side: s, price: o.Price}
}

// Cancel removes a resting order by ID, reporting whether it existed.
func (b *OrderBook) Cancel(id uint64) bool {
	loc, ok := b.locs[id]
	if !ok {
		return false
	}
	lvl := loc.side.levelAt(loc.price, false)
	if lvl == nil || !lvl.removeByID(id) {
		delete(b.locs, id)
		return false
	}
	loc.side.dropEmpty(loc.price)
	delete(b.locs, id)
	return true
}

// BestBid returns the highest bid price, or ok=false if there are no bids.
func (b *OrderBook) BestBid() (int64, bool) {
	if lvl := b.bids.bestLevel(); lvl != nil {
		return lvl.price, true
	}
	return 0, false
}

// BestAsk returns the lowest ask price, or ok=false if there are no asks.
func (b *OrderBook) BestAsk() (int64, bool) {
	if lvl := b.asks.bestLevel(); lvl != nil {
		return lvl.price, true
	}
	return 0, false
}

// Depth returns up to n levels per side, best price first.
func (b *OrderBook) Depth(n int) (bids []Level, asks []Level) {
	// Bids: iterate from highest price (end of ascending slice) downward.
	for i := len(b.bids.levels) - 1; i >= 0 && len(bids) < n; i-- {
		lvl := b.bids.levels[i]
		bids = append(bids, Level{Price: lvl.price, Qty: lvl.totalQty()})
	}
	// Asks: iterate from lowest price (start of ascending slice) upward.
	for i := 0; i < len(b.asks.levels) && len(asks) < n; i++ {
		lvl := b.asks.levels[i]
		asks = append(asks, Level{Price: lvl.price, Qty: lvl.totalQty()})
	}
	return bids, asks
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd ~/Personal/lob && go test ./book/`
Expected: PASS (Task 2 tests still pass too).

- [ ] **Step 5: Commit**

```bash
cd ~/Personal/lob
git add book/book.go book/book_test.go
git commit -m "feat: add order book structure, resting, cancel, and depth

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 4: Matching (`Add`) — price-time priority

**Files:**
- Modify: `book/book.go` (add the `trade` import, `crosses`, and `Add`)
- Test: `book/matching_test.go`

**Interfaces:**
- Consumes: `rest`, `sideFor`, `side.bestLevel`, `priceLevel.front/reduceFront`, `side.dropEmpty`, `orderLoc` bookkeeping (Task 3); `order.Order`, `trade.Trade`.
- Produces:
  - `func (b *OrderBook) Add(o order.Order) ([]trade.Trade, error)` — validate `o`; match against the opposite side by price-time priority; emit trades at the resting (maker) price; rest any remaining quantity. Returns `order.Validate` errors unchanged.

**Matching rules (implement exactly):**
- Opposite side = asks for a Buy, bids for a Sell.
- A Buy crosses when `o.Price >= bestAsk`; a Sell crosses when `o.Price <= bestBid`.
- Loop while `o.Qty > 0` and the opposite best level crosses:
  - Read the front resting order `m`. `fill = min(o.Qty, m.Qty)`.
  - Emit `trade.Trade{MakerID: m.ID, TakerID: o.ID, Price: m.Price, Qty: fill, TS: o.TS}`.
  - `reduceFront(fill)`; if `m` is now fully filled, delete it from `locs`.
  - `o.Qty -= fill`. After reducing, `dropEmpty` the maker price so an emptied level disappears.
- When the loop ends, if `o.Qty > 0`, `rest(o)`.

- [ ] **Step 1: Write the failing test**

Create `book/matching_test.go`:
```go
package book

import (
	"errors"
	"testing"

	"github.com/jayrajadeja/lob/order"
)

func TestAddRejectsInvalid(t *testing.T) {
	b := New("TEST")
	if _, err := b.Add(limit(1, order.Buy, 100, 0)); !errors.Is(err, order.ErrInvalidQty) {
		t.Errorf("err = %v, want ErrInvalidQty", err)
	}
	if _, err := b.Add(limit(2, order.Sell, 0, 5)); !errors.Is(err, order.ErrInvalidPrice) {
		t.Errorf("err = %v, want ErrInvalidPrice", err)
	}
}

func TestAddNoCrossRests(t *testing.T) {
	b := New("TEST")
	trades, err := b.Add(limit(1, order.Buy, 100, 5))
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 0 {
		t.Fatalf("trades = %+v, want none", trades)
	}
	if bid, ok := b.BestBid(); !ok || bid != 100 {
		t.Errorf("BestBid = %d ok=%v, want 100", bid, ok)
	}
}

func TestAddFullMatch(t *testing.T) {
	b := New("TEST")
	b.Add(limit(1, order.Sell, 100, 5)) // resting ask
	trades, _ := b.Add(limit(2, order.Buy, 100, 5))
	if len(trades) != 1 {
		t.Fatalf("got %d trades, want 1", len(trades))
	}
	tr := trades[0]
	if tr.MakerID != 1 || tr.TakerID != 2 || tr.Price != 100 || tr.Qty != 5 {
		t.Errorf("trade = %+v, want maker 1 taker 2 price 100 qty 5", tr)
	}
	if _, ok := b.BestAsk(); ok {
		t.Error("ask should be fully consumed")
	}
}

func TestAddPartialFillRestsRemainder(t *testing.T) {
	b := New("TEST")
	b.Add(limit(1, order.Sell, 100, 3)) // resting ask qty 3
	trades, _ := b.Add(limit(2, order.Buy, 100, 5))
	if len(trades) != 1 || trades[0].Qty != 3 {
		t.Fatalf("trades = %+v, want one qty 3", trades)
	}
	// 2 units remain and rest as a bid at 100.
	if bid, ok := b.BestBid(); !ok || bid != 100 {
		t.Errorf("BestBid = %d ok=%v, want 100", bid, ok)
	}
	bids, _ := b.Depth(1)
	if bids[0].Qty != 2 {
		t.Errorf("resting bid qty = %d, want 2", bids[0].Qty)
	}
}

func TestAddSweepsMultipleLevelsTimePriority(t *testing.T) {
	b := New("TEST")
	// Two resting asks at 100 (time priority: 1 before 2) and one at 101.
	b.Add(limit(1, order.Sell, 100, 2))
	b.Add(limit(2, order.Sell, 100, 2))
	b.Add(limit(3, order.Sell, 101, 5))

	// Aggressive buy for 6 at price 101 sweeps 100 (both), then 101.
	trades, _ := b.Add(limit(4, order.Buy, 101, 6))
	if len(trades) != 3 {
		t.Fatalf("got %d trades, want 3", len(trades))
	}
	wantMakers := []uint64{1, 2, 3}
	wantPrices := []int64{100, 100, 101}
	wantQty := []uint64{2, 2, 2}
	for i, tr := range trades {
		if tr.MakerID != wantMakers[i] || tr.Price != wantPrices[i] || tr.Qty != wantQty[i] {
			t.Errorf("trade[%d] = %+v, want maker %d price %d qty %d",
				i, tr, wantMakers[i], wantPrices[i], wantQty[i])
		}
	}
	// 101 level had 5, 2 consumed -> 3 remain as best ask.
	if ask, ok := b.BestAsk(); !ok || ask != 101 {
		t.Errorf("BestAsk = %d ok=%v, want 101", ask, ok)
	}
}

func TestAddNoCrossWhenPriceTooLow(t *testing.T) {
	b := New("TEST")
	b.Add(limit(1, order.Sell, 105, 5))            // ask at 105
	trades, _ := b.Add(limit(2, order.Buy, 104, 5)) // bid below ask -> no cross
	if len(trades) != 0 {
		t.Fatalf("trades = %+v, want none", trades)
	}
	if bid, _ := b.BestBid(); bid != 104 {
		t.Errorf("bid should rest at 104, got %d", bid)
	}
}

func TestCancelledMakerIsRemovedFromLocs(t *testing.T) {
	b := New("TEST")
	b.Add(limit(1, order.Sell, 100, 5))
	b.Add(limit(2, order.Buy, 100, 5)) // fully fills maker 1
	// Maker 1 is gone; cancelling it must report false.
	if b.Cancel(1) {
		t.Error("Cancel(1) = true, want false (already filled)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd ~/Personal/lob && go test ./book/ -run TestAdd`
Expected: FAIL — `Add` undefined.

- [ ] **Step 3: Write minimal implementation**

Add `"github.com/jayrajadeja/lob/trade"` to the import block in `book/book.go` so it reads:
```go
import (
	"sort"

	"github.com/jayrajadeja/lob/order"
	"github.com/jayrajadeja/lob/trade"
)
```

Append to `book/book.go`:
```go
// crosses reports whether an incoming order at price can trade against a resting
// order at makerPrice on the opposite side.
func crosses(incoming order.Side, price, makerPrice int64) bool {
	if incoming == order.Buy {
		return price >= makerPrice
	}
	return price <= makerPrice
}

// Add matches an incoming limit order against the opposite side by price-time
// priority, emits the resulting trades (at each maker's price), and rests any
// remaining quantity. It returns a validation error for malformed orders.
func (b *OrderBook) Add(o order.Order) ([]trade.Trade, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}

	var opposite *side
	if o.Side == order.Buy {
		opposite = b.asks
	} else {
		opposite = b.bids
	}

	var trades []trade.Trade
	for o.Qty > 0 {
		best := opposite.bestLevel()
		if best == nil || !crosses(o.Side, o.Price, best.price) {
			break
		}
		maker, ok := best.front()
		if !ok {
			opposite.dropEmpty(best.price)
			continue
		}
		fill := maker.Qty
		if o.Qty < fill {
			fill = o.Qty
		}
		trades = append(trades, trade.Trade{
			MakerID: maker.ID,
			TakerID: o.ID,
			Price:   maker.Price,
			Qty:     fill,
			TS:      o.TS,
		})
		best.reduceFront(fill)
		if fill == maker.Qty {
			delete(b.locs, maker.ID) // maker fully filled
		}
		o.Qty -= fill
		opposite.dropEmpty(best.price)
	}

	if o.Qty > 0 {
		b.rest(o)
	}
	return trades, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd ~/Personal/lob && go test ./book/`
Expected: PASS (all Task 2–4 tests).

- [ ] **Step 5: Commit**

```bash
cd ~/Personal/lob
git add book/book.go book/matching_test.go
git commit -m "feat: add price-time priority matching to order book

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 5: Synthetic order generator

**Files:**
- Create: `gen/gen.go`
- Test: `gen/gen_test.go`

**Interfaces:**
- Consumes: `order.Order`, `order.Side` (Task 1).
- Produces:
  - `type Kind uint8` with `gen.Place`, `gen.CancelOrder`.
  - `type Action struct { Kind Kind; Order order.Order; CancelID uint64 }`
  - `type Params struct { Seed int64; Count int; Mid int64; Tick int64; MaxQty uint64; CancelProb float64 }`
  - `func New(p Params) *Generator`
  - `func (g *Generator) Next() (Action, bool)` — returns the next action; `ok=false` once `Count` actions have been produced. Deterministic for a given seed.

**Design notes:**
- Use `math/rand.New(rand.NewSource(p.Seed))` — never the global rand — so the stream is reproducible.
- Each call: with probability `CancelProb`, if there is at least one live (placed, not yet cancelled) order ID, emit a `CancelOrder` action for a randomly chosen live ID and forget it. Otherwise emit a `Place`: pick a side (50/50), a price = `Mid + (rand step in [-5,5]) * Tick`, and qty in `[1, MaxQty]`. Assign a monotonically increasing order ID and TS (both start at 1), track the new ID as live.
- Keep `Price > 0`: clamp the generated price to a minimum of `Tick`.

- [ ] **Step 1: Write the failing test**

Create `gen/gen_test.go`:
```go
package gen

import (
	"testing"
)

func collect(p Params) []Action {
	g := New(p)
	var out []Action
	for {
		a, ok := g.Next()
		if !ok {
			break
		}
		out = append(out, a)
	}
	return out
}

func TestGeneratorDeterministic(t *testing.T) {
	p := Params{Seed: 42, Count: 500, Mid: 10000, Tick: 1, MaxQty: 10, CancelProb: 0.1}
	a := collect(p)
	b := collect(p)
	if len(a) != 500 || len(b) != 500 {
		t.Fatalf("counts = %d, %d, want 500 each", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("action %d differs: %+v vs %+v", i, a[i], b[i])
		}
	}
}

func TestGeneratorPlacesValidOrders(t *testing.T) {
	p := Params{Seed: 7, Count: 200, Mid: 10000, Tick: 1, MaxQty: 10, CancelProb: 0.2}
	for _, a := range collect(p) {
		if a.Kind == Place {
			if a.Order.Qty == 0 || a.Order.Qty > 10 {
				t.Fatalf("qty out of range: %d", a.Order.Qty)
			}
			if a.Order.Price <= 0 {
				t.Fatalf("non-positive price: %d", a.Order.Price)
			}
		}
	}
}

func TestGeneratorCancelsReferencePlacedIDs(t *testing.T) {
	p := Params{Seed: 3, Count: 400, Mid: 10000, Tick: 1, MaxQty: 5, CancelProb: 0.3}
	placed := map[uint64]bool{}
	for _, a := range collect(p) {
		switch a.Kind {
		case Place:
			placed[a.Order.ID] = true
		case CancelOrder:
			if !placed[a.CancelID] {
				t.Fatalf("cancel references unplaced ID %d", a.CancelID)
			}
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd ~/Personal/lob && go test ./gen/`
Expected: FAIL — `New`, `Params`, `Action`, `Place`, `CancelOrder` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `gen/gen.go`:
```go
// Package gen produces a deterministic, seedable stream of order-book actions
// (limit placements and cancels) for driving and load-testing the engine. It
// depends only on the order data types and never touches the book internals.
package gen

import (
	"math/rand"

	"github.com/jayrajadeja/lob/order"
)

// Kind distinguishes a placement from a cancel.
type Kind uint8

const (
	Place Kind = iota
	CancelOrder
)

// Action is one generated instruction: a Place carries Order; a CancelOrder
// carries CancelID.
type Action struct {
	Kind     Kind
	Order    order.Order
	CancelID uint64
}

// Params configures a generator run.
type Params struct {
	Seed       int64
	Count      int
	Mid        int64
	Tick       int64
	MaxQty     uint64
	CancelProb float64
}

// Generator emits a reproducible action stream for a given seed.
type Generator struct {
	p       Params
	rng     *rand.Rand
	emitted int
	nextID  uint64
	nextTS  int64
	live    []uint64
}

// New creates a generator. A MaxQty of 0 is treated as 1, and a Tick <= 0 as 1.
func New(p Params) *Generator {
	if p.MaxQty == 0 {
		p.MaxQty = 1
	}
	if p.Tick <= 0 {
		p.Tick = 1
	}
	return &Generator{
		p:      p,
		rng:    rand.New(rand.NewSource(p.Seed)),
		nextID: 1,
		nextTS: 1,
	}
}

// Next returns the next action, or ok=false once Count actions are produced.
func (g *Generator) Next() (Action, bool) {
	if g.emitted >= g.p.Count {
		return Action{}, false
	}
	g.emitted++

	if len(g.live) > 0 && g.rng.Float64() < g.p.CancelProb {
		idx := g.rng.Intn(len(g.live))
		id := g.live[idx]
		g.live[idx] = g.live[len(g.live)-1]
		g.live = g.live[:len(g.live)-1]
		return Action{Kind: CancelOrder, CancelID: id}, true
	}

	side := order.Buy
	if g.rng.Intn(2) == 1 {
		side = order.Sell
	}
	step := int64(g.rng.Intn(11) - 5) // [-5, 5]
	price := g.p.Mid + step*g.p.Tick
	if price < g.p.Tick {
		price = g.p.Tick
	}
	qty := uint64(g.rng.Intn(int(g.p.MaxQty))) + 1

	o := order.Order{
		ID:    g.nextID,
		Side:  side,
		Price: price,
		Qty:   qty,
		TS:    g.nextTS,
	}
	g.nextID++
	g.nextTS++
	g.live = append(g.live, o.ID)
	return Action{Kind: Place, Order: o}, true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd ~/Personal/lob && go test ./gen/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd ~/Personal/lob
git add gen/
git commit -m "feat: add deterministic synthetic order generator

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 6: Replay CLI + golden determinism test

**Files:**
- Create: `cmd/replay/main.go`
- Create: `cmd/replay/golden_test.go`
- Create: `README.md`

**Interfaces:**
- Consumes: `gen.New/Params/Next/Action/Place/CancelOrder`, `book.New/Add/Cancel/Depth`, `trade.Trade`.
- Produces: a runnable binary and a golden test pinning the exact trade stream for a fixed seed.

**Design notes:**
- `main` parses flags, runs the generator through the book, accumulates trades, and prints: total actions, total trades, top-5 depth per side, and actions/sec throughput. Errors from `book.Add` go to stderr and are skipped (the generator should not produce invalid orders, but the loop must not panic).
- The golden test runs a fixed seed through `gen`+`book` and asserts an exact trade slice. The implementer captures the golden values from a real run (Step 3), then pastes them in. Do NOT invent numbers.

- [ ] **Step 1: Write the replay binary**

Create `cmd/replay/main.go`:
```go
// Command replay drives the matching engine with the synthetic generator and
// prints the resulting trades, final book depth, and throughput. This is the
// only package in the project that performs I/O.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jayrajadeja/lob/book"
	"github.com/jayrajadeja/lob/gen"
)

func main() {
	seed := flag.Int64("seed", 1, "generator seed")
	orders := flag.Int("orders", 10000, "number of actions to generate")
	symbol := flag.String("symbol", "SYNTH", "symbol name")
	mid := flag.Int64("mid", 10000, "mid price in ticks")
	tick := flag.Int64("tick", 1, "tick size")
	maxQty := flag.Uint64("maxqty", 10, "max order quantity")
	cancelProb := flag.Float64("cancelprob", 0.1, "cancel probability")
	flag.Parse()

	b := book.New(*symbol)
	g := gen.New(gen.Params{
		Seed:       *seed,
		Count:      *orders,
		Mid:        *mid,
		Tick:       *tick,
		MaxQty:     *maxQty,
		CancelProb: *cancelProb,
	})

	var totalTrades int
	start := time.Now()
	for {
		a, ok := g.Next()
		if !ok {
			break
		}
		switch a.Kind {
		case gen.Place:
			trades, err := b.Add(a.Order)
			if err != nil {
				fmt.Fprintf(os.Stderr, "add error: %v\n", err)
				continue
			}
			totalTrades += len(trades)
		case gen.CancelOrder:
			b.Cancel(a.CancelID)
		}
	}
	elapsed := time.Since(start)

	bids, asks := b.Depth(5)
	fmt.Printf("symbol=%s actions=%d trades=%d\n", *symbol, *orders, totalTrades)
	fmt.Println("top bids (price x qty):")
	for _, l := range bids {
		fmt.Printf("  %d x %d\n", l.Price, l.Qty)
	}
	fmt.Println("top asks (price x qty):")
	for _, l := range asks {
		fmt.Printf("  %d x %d\n", l.Price, l.Qty)
	}
	if elapsed > 0 {
		fmt.Printf("throughput=%.0f actions/sec\n", float64(*orders)/elapsed.Seconds())
	}
}
```

- [ ] **Step 2: Verify the binary builds and runs**

Run: `cd ~/Personal/lob && go run ./cmd/replay -seed 1 -orders 2000`
Expected: prints `symbol=SYNTH actions=2000 trades=...`, depth lines, and a throughput line. No panic, exit 0.

- [ ] **Step 3: Capture golden trade values**

Create a temporary helper, run it once, and copy the output:
```bash
cd ~/Personal/lob
mkdir -p /tmp/goldgen
cat > /tmp/goldgen/main.go <<'EOF'
package main

import (
	"fmt"

	"github.com/jayrajadeja/lob/book"
	"github.com/jayrajadeja/lob/gen"
)

func main() {
	b := book.New("G")
	g := gen.New(gen.Params{Seed: 99, Count: 200, Mid: 100, Tick: 1, MaxQty: 4, CancelProb: 0.0})
	for {
		a, ok := g.Next()
		if !ok {
			break
		}
		trades, _ := b.Add(a.Order)
		for _, tr := range trades {
			fmt.Printf("{%d, %d, %d, %d, %d},\n", tr.MakerID, tr.TakerID, tr.Price, tr.Qty, tr.TS)
		}
	}
}
EOF
go run /tmp/goldgen/main.go
```
Expected: prints one `{maker, taker, price, qty, ts},` line per trade. Copy these lines verbatim into the `want` slice in Step 4. (`CancelProb: 0.0` keeps the golden stream free of cancels for a stable sequence.) If the output is empty, the seed produced no crosses — bump `Count` to 500 here and in Step 4 until at least a few trades appear, then capture.

- [ ] **Step 4: Write the golden test**

Create `cmd/replay/golden_test.go` — paste the captured tuples into `want` (replace the example row with the real captured output; keep the `Count` matching whatever Step 3 used):
```go
package main

import (
	"testing"

	"github.com/jayrajadeja/lob/book"
	"github.com/jayrajadeja/lob/gen"
	"github.com/jayrajadeja/lob/trade"
)

func runGolden() []trade.Trade {
	b := book.New("G")
	g := gen.New(gen.Params{Seed: 99, Count: 200, Mid: 100, Tick: 1, MaxQty: 4, CancelProb: 0.0})
	var trades []trade.Trade
	for {
		a, ok := g.Next()
		if !ok {
			break
		}
		got, _ := b.Add(a.Order)
		trades = append(trades, got...)
	}
	return trades
}

func TestGoldenTradeStream(t *testing.T) {
	want := []trade.Trade{
		// PASTE captured {maker, taker, price, qty, ts} rows here, e.g.:
		// {3, 5, 100, 2, 5},
	}
	got := runGolden()
	if len(got) != len(want) {
		t.Fatalf("trade count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("trade[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
```

- [ ] **Step 5: Run the golden test**

Run: `cd ~/Personal/lob && rm -rf /tmp/goldgen && go test ./...`
Expected: PASS across all packages, including `cmd/replay`.

- [ ] **Step 6: Write the README**

Create `README.md`:
```markdown
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
```

- [ ] **Step 7: Commit**

```bash
cd ~/Personal/lob
go vet ./...
git add cmd/ README.md
git commit -m "feat: add replay CLI and golden determinism test

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Notes for the executor

- Run `go test ./...` after each task; keep it green before moving on.
- `book` must never import `gen` or `cmd` — if `go list -deps ./book` shows either, the boundary was violated.
- The golden test values MUST come from a real run (Step 3), never invented.
