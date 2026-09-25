// Package book is the matching engine core: a two-sided limit order book with
// price-time priority. It is pure and deterministic — no I/O, no wall clock —
// and imports only the order and trade data packages.
package book

import (
	"sort"

	"github.com/jayrajadeja/lob/order"
	"github.com/jayrajadeja/lob/trade"
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
