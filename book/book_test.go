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
