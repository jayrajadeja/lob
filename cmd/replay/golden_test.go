package main

import (
	"testing"

	"github.com/jayrajadeja/lob/book"
	"github.com/jayrajadeja/lob/gen"
	"github.com/jayrajadeja/lob/order"
	"github.com/jayrajadeja/lob/trade"
)

func runGolden(t *testing.T) []trade.Trade {
	t.Helper()
	b := book.New("G")
	g := gen.New(gen.Params{Seed: 99, Count: 200, Mid: 100, Tick: 1, MaxQty: 4, CancelProb: 0.0})
	var trades []trade.Trade
	for {
		a, ok := g.Next()
		if !ok {
			break
		}
		got, err := b.Add(a.Order)
		if err != nil {
			t.Fatalf("add order %d: %v", a.Order.ID, err)
		}
		trades = append(trades, got...)
	}
	return trades
}

func TestGoldenTradeStream(t *testing.T) {
	want := []trade.Trade{
		{MakerID: 1, TakerID: 2, Price: 95, Qty: 2, TS: 2, AggressorSide: order.Buy},
		{MakerID: 1, TakerID: 4, Price: 95, Qty: 1, TS: 4, AggressorSide: order.Buy},
		{MakerID: 3, TakerID: 4, Price: 102, Qty: 3, TS: 4, AggressorSide: order.Buy},
		{MakerID: 6, TakerID: 8, Price: 104, Qty: 2, TS: 8, AggressorSide: order.Buy},
		{MakerID: 7, TakerID: 10, Price: 102, Qty: 1, TS: 10, AggressorSide: order.Sell},
		{MakerID: 10, TakerID: 16, Price: 100, Qty: 2, TS: 16, AggressorSide: order.Buy},
		{MakerID: 10, TakerID: 19, Price: 100, Qty: 1, TS: 19, AggressorSide: order.Buy},
		{MakerID: 12, TakerID: 19, Price: 102, Qty: 2, TS: 19, AggressorSide: order.Buy},
		{MakerID: 19, TakerID: 21, Price: 102, Qty: 1, TS: 21, AggressorSide: order.Sell},
		{MakerID: 5, TakerID: 22, Price: 99, Qty: 1, TS: 22, AggressorSide: order.Sell},
		{MakerID: 26, TakerID: 27, Price: 100, Qty: 1, TS: 27, AggressorSide: order.Buy},
		{MakerID: 5, TakerID: 28, Price: 99, Qty: 2, TS: 28, AggressorSide: order.Sell},
		{MakerID: 13, TakerID: 28, Price: 99, Qty: 1, TS: 28, AggressorSide: order.Sell},
		{MakerID: 30, TakerID: 31, Price: 99, Qty: 4, TS: 31, AggressorSide: order.Sell},
		{MakerID: 26, TakerID: 33, Price: 100, Qty: 2, TS: 33, AggressorSide: order.Buy},
		{MakerID: 36, TakerID: 37, Price: 101, Qty: 1, TS: 37, AggressorSide: order.Sell},
		{MakerID: 24, TakerID: 40, Price: 102, Qty: 2, TS: 40, AggressorSide: order.Buy},
		{MakerID: 35, TakerID: 40, Price: 102, Qty: 2, TS: 40, AggressorSide: order.Buy},
		{MakerID: 35, TakerID: 41, Price: 102, Qty: 1, TS: 41, AggressorSide: order.Buy},
		{MakerID: 39, TakerID: 41, Price: 103, Qty: 1, TS: 41, AggressorSide: order.Buy},
		{MakerID: 36, TakerID: 42, Price: 101, Qty: 2, TS: 42, AggressorSide: order.Sell},
		{MakerID: 36, TakerID: 43, Price: 101, Qty: 1, TS: 43, AggressorSide: order.Sell},
		{MakerID: 34, TakerID: 45, Price: 100, Qty: 1, TS: 45, AggressorSide: order.Sell},
		{MakerID: 11, TakerID: 45, Price: 98, Qty: 1, TS: 45, AggressorSide: order.Sell},
		{MakerID: 11, TakerID: 46, Price: 98, Qty: 1, TS: 46, AggressorSide: order.Sell},
		{MakerID: 23, TakerID: 46, Price: 98, Qty: 1, TS: 46, AggressorSide: order.Sell},
		{MakerID: 47, TakerID: 49, Price: 99, Qty: 1, TS: 49, AggressorSide: order.Buy},
		{MakerID: 47, TakerID: 51, Price: 99, Qty: 1, TS: 51, AggressorSide: order.Buy},
		{MakerID: 23, TakerID: 52, Price: 98, Qty: 2, TS: 52, AggressorSide: order.Sell},
		{MakerID: 9, TakerID: 52, Price: 97, Qty: 1, TS: 52, AggressorSide: order.Sell},
		{MakerID: 47, TakerID: 53, Price: 99, Qty: 1, TS: 53, AggressorSide: order.Buy},
		{MakerID: 47, TakerID: 56, Price: 99, Qty: 1, TS: 56, AggressorSide: order.Buy},
		{MakerID: 48, TakerID: 56, Price: 102, Qty: 2, TS: 56, AggressorSide: order.Buy},
		{MakerID: 48, TakerID: 57, Price: 102, Qty: 1, TS: 57, AggressorSide: order.Buy},
		{MakerID: 48, TakerID: 59, Price: 102, Qty: 1, TS: 59, AggressorSide: order.Buy},
		{MakerID: 39, TakerID: 59, Price: 103, Qty: 1, TS: 59, AggressorSide: order.Buy},
		{MakerID: 62, TakerID: 67, Price: 98, Qty: 2, TS: 67, AggressorSide: order.Buy},
		{MakerID: 67, TakerID: 69, Price: 98, Qty: 1, TS: 69, AggressorSide: order.Sell},
		{MakerID: 9, TakerID: 69, Price: 97, Qty: 2, TS: 69, AggressorSide: order.Sell},
		{MakerID: 63, TakerID: 70, Price: 99, Qty: 1, TS: 70, AggressorSide: order.Buy},
		{MakerID: 72, TakerID: 73, Price: 99, Qty: 3, TS: 73, AggressorSide: order.Buy},
		{MakerID: 68, TakerID: 73, Price: 100, Qty: 1, TS: 73, AggressorSide: order.Buy},
		{MakerID: 60, TakerID: 75, Price: 101, Qty: 1, TS: 75, AggressorSide: order.Buy},
		{MakerID: 60, TakerID: 76, Price: 101, Qty: 2, TS: 76, AggressorSide: order.Buy},
		{MakerID: 9, TakerID: 78, Price: 97, Qty: 1, TS: 78, AggressorSide: order.Sell},
		{MakerID: 17, TakerID: 78, Price: 96, Qty: 2, TS: 78, AggressorSide: order.Sell},
		{MakerID: 71, TakerID: 81, Price: 101, Qty: 1, TS: 81, AggressorSide: order.Buy},
		{MakerID: 82, TakerID: 83, Price: 100, Qty: 1, TS: 83, AggressorSide: order.Sell},
		{MakerID: 71, TakerID: 84, Price: 101, Qty: 2, TS: 84, AggressorSide: order.Buy},
		{MakerID: 71, TakerID: 87, Price: 101, Qty: 1, TS: 87, AggressorSide: order.Buy},
		{MakerID: 85, TakerID: 87, Price: 101, Qty: 2, TS: 87, AggressorSide: order.Buy},
		{MakerID: 85, TakerID: 89, Price: 101, Qty: 1, TS: 89, AggressorSide: order.Buy},
		{MakerID: 65, TakerID: 90, Price: 102, Qty: 1, TS: 90, AggressorSide: order.Buy},
		{MakerID: 89, TakerID: 92, Price: 101, Qty: 2, TS: 92, AggressorSide: order.Sell},
		{MakerID: 82, TakerID: 92, Price: 100, Qty: 1, TS: 92, AggressorSide: order.Sell},
		{MakerID: 82, TakerID: 95, Price: 100, Qty: 1, TS: 95, AggressorSide: order.Sell},
		{MakerID: 94, TakerID: 95, Price: 100, Qty: 1, TS: 95, AggressorSide: order.Sell},
		{MakerID: 94, TakerID: 96, Price: 100, Qty: 1, TS: 96, AggressorSide: order.Sell},
		{MakerID: 79, TakerID: 96, Price: 99, Qty: 3, TS: 96, AggressorSide: order.Sell},
		{MakerID: 65, TakerID: 97, Price: 102, Qty: 2, TS: 97, AggressorSide: order.Buy},
		{MakerID: 103, TakerID: 107, Price: 101, Qty: 1, TS: 107, AggressorSide: order.Buy},
		{MakerID: 66, TakerID: 107, Price: 102, Qty: 1, TS: 107, AggressorSide: order.Buy},
		{MakerID: 86, TakerID: 107, Price: 102, Qty: 2, TS: 107, AggressorSide: order.Buy},
		{MakerID: 99, TakerID: 108, Price: 100, Qty: 4, TS: 108, AggressorSide: order.Sell},
		{MakerID: 101, TakerID: 109, Price: 100, Qty: 1, TS: 109, AggressorSide: order.Sell},
		{MakerID: 88, TakerID: 109, Price: 99, Qty: 1, TS: 109, AggressorSide: order.Sell},
		{MakerID: 110, TakerID: 113, Price: 101, Qty: 4, TS: 113, AggressorSide: order.Sell},
		{MakerID: 111, TakerID: 115, Price: 101, Qty: 1, TS: 115, AggressorSide: order.Sell},
		{MakerID: 88, TakerID: 115, Price: 99, Qty: 2, TS: 115, AggressorSide: order.Sell},
		{MakerID: 116, TakerID: 117, Price: 101, Qty: 2, TS: 117, AggressorSide: order.Sell},
		{MakerID: 86, TakerID: 120, Price: 102, Qty: 1, TS: 120, AggressorSide: order.Buy},
		{MakerID: 104, TakerID: 121, Price: 102, Qty: 4, TS: 121, AggressorSide: order.Buy},
		{MakerID: 118, TakerID: 122, Price: 100, Qty: 2, TS: 122, AggressorSide: order.Sell},
		{MakerID: 118, TakerID: 124, Price: 100, Qty: 1, TS: 124, AggressorSide: order.Sell},
		{MakerID: 123, TakerID: 124, Price: 99, Qty: 1, TS: 124, AggressorSide: order.Sell},
		{MakerID: 114, TakerID: 129, Price: 102, Qty: 1, TS: 129, AggressorSide: order.Buy},
		{MakerID: 123, TakerID: 131, Price: 99, Qty: 3, TS: 131, AggressorSide: order.Sell},
		{MakerID: 125, TakerID: 131, Price: 99, Qty: 1, TS: 131, AggressorSide: order.Sell},
		{MakerID: 125, TakerID: 133, Price: 99, Qty: 3, TS: 133, AggressorSide: order.Sell},
		{MakerID: 128, TakerID: 133, Price: 99, Qty: 1, TS: 133, AggressorSide: order.Sell},
		{MakerID: 128, TakerID: 134, Price: 99, Qty: 1, TS: 134, AggressorSide: order.Sell},
		{MakerID: 91, TakerID: 137, Price: 98, Qty: 4, TS: 137, AggressorSide: order.Sell},
		{MakerID: 144, TakerID: 145, Price: 99, Qty: 1, TS: 145, AggressorSide: order.Buy},
		{MakerID: 144, TakerID: 148, Price: 99, Qty: 3, TS: 148, AggressorSide: order.Buy},
		{MakerID: 106, TakerID: 149, Price: 98, Qty: 3, TS: 149, AggressorSide: order.Sell},
		{MakerID: 136, TakerID: 151, Price: 100, Qty: 1, TS: 151, AggressorSide: order.Buy},
		{MakerID: 143, TakerID: 151, Price: 100, Qty: 1, TS: 151, AggressorSide: order.Buy},
		{MakerID: 138, TakerID: 151, Price: 101, Qty: 1, TS: 151, AggressorSide: order.Buy},
		{MakerID: 139, TakerID: 152, Price: 101, Qty: 1, TS: 152, AggressorSide: order.Buy},
		{MakerID: 150, TakerID: 152, Price: 101, Qty: 2, TS: 152, AggressorSide: order.Buy},
		{MakerID: 146, TakerID: 160, Price: 98, Qty: 1, TS: 160, AggressorSide: order.Sell},
		{MakerID: 100, TakerID: 160, Price: 97, Qty: 1, TS: 160, AggressorSide: order.Sell},
		{MakerID: 130, TakerID: 160, Price: 97, Qty: 1, TS: 160, AggressorSide: order.Sell},
		{MakerID: 161, TakerID: 163, Price: 98, Qty: 1, TS: 163, AggressorSide: order.Buy},
		{MakerID: 161, TakerID: 164, Price: 98, Qty: 3, TS: 164, AggressorSide: order.Buy},
		{MakerID: 155, TakerID: 165, Price: 99, Qty: 2, TS: 165, AggressorSide: order.Buy},
		{MakerID: 155, TakerID: 166, Price: 99, Qty: 2, TS: 166, AggressorSide: order.Buy},
		{MakerID: 153, TakerID: 166, Price: 100, Qty: 1, TS: 166, AggressorSide: order.Buy},
		{MakerID: 153, TakerID: 168, Price: 100, Qty: 2, TS: 168, AggressorSide: order.Buy},
		{MakerID: 157, TakerID: 168, Price: 101, Qty: 1, TS: 168, AggressorSide: order.Buy},
		{MakerID: 157, TakerID: 169, Price: 101, Qty: 1, TS: 169, AggressorSide: order.Buy},
		{MakerID: 162, TakerID: 169, Price: 101, Qty: 2, TS: 169, AggressorSide: order.Buy},
		{MakerID: 162, TakerID: 174, Price: 101, Qty: 2, TS: 174, AggressorSide: order.Buy},
		{MakerID: 171, TakerID: 174, Price: 101, Qty: 1, TS: 174, AggressorSide: order.Buy},
		{MakerID: 175, TakerID: 177, Price: 100, Qty: 2, TS: 177, AggressorSide: order.Sell},
		{MakerID: 173, TakerID: 177, Price: 99, Qty: 2, TS: 177, AggressorSide: order.Sell},
		{MakerID: 173, TakerID: 178, Price: 99, Qty: 1, TS: 178, AggressorSide: order.Sell},
		{MakerID: 171, TakerID: 180, Price: 101, Qty: 1, TS: 180, AggressorSide: order.Buy},
		{MakerID: 114, TakerID: 182, Price: 102, Qty: 1, TS: 182, AggressorSide: order.Buy},
		{MakerID: 180, TakerID: 183, Price: 101, Qty: 3, TS: 183, AggressorSide: order.Sell},
		{MakerID: 130, TakerID: 183, Price: 97, Qty: 1, TS: 183, AggressorSide: order.Sell},
		{MakerID: 159, TakerID: 187, Price: 102, Qty: 1, TS: 187, AggressorSide: order.Buy},
		{MakerID: 167, TakerID: 187, Price: 102, Qty: 1, TS: 187, AggressorSide: order.Buy},
		{MakerID: 179, TakerID: 189, Price: 102, Qty: 1, TS: 189, AggressorSide: order.Buy},
		{MakerID: 181, TakerID: 189, Price: 102, Qty: 2, TS: 189, AggressorSide: order.Buy},
		{MakerID: 185, TakerID: 191, Price: 101, Qty: 2, TS: 191, AggressorSide: order.Sell},
		{MakerID: 185, TakerID: 195, Price: 101, Qty: 2, TS: 195, AggressorSide: order.Sell},
		{MakerID: 186, TakerID: 195, Price: 101, Qty: 1, TS: 195, AggressorSide: order.Sell},
		{MakerID: 186, TakerID: 199, Price: 101, Qty: 1, TS: 199, AggressorSide: order.Sell},
	}
	got := runGolden(t)
	if len(got) != len(want) {
		t.Fatalf("trade count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("trade[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
