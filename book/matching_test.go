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
	if tr.AggressorSide != order.Buy {
		t.Errorf("aggressor side = %v, want Buy", tr.AggressorSide)
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
	b.Add(limit(1, order.Sell, 105, 5))             // ask at 105
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
