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
