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
