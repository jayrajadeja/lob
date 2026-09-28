package main

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/jayrajadeja/lob/order"
	"github.com/jayrajadeja/lob/trade"
)

// decodeTrade reverses encodeTrade for test purposes only. It recovers the four
// fields the wire format carries (TS, Price, Qty, AggressorSide); maker/taker
// IDs are not on the wire.
func decodeTrade(buf []byte) trade.Trade {
	return trade.Trade{
		TS:            int64(le64(buf[0:8])),
		Price:         int64(le64(buf[8:16])),
		Qty:           le64(buf[16:24]),
		AggressorSide: order.Side(buf[24]),
	}
}

func le64(b []byte) uint64 {
	var v uint64
	for i := 7; i >= 0; i-- {
		v = v<<8 | uint64(b[i])
	}
	return v
}

func TestEncodeTradeRoundTrip(t *testing.T) {
	in := []trade.Trade{
		{TS: 1, Price: 100, Qty: 5, AggressorSide: order.Buy},
		{TS: 7, Price: -1500, Qty: 9, AggressorSide: order.Sell},
	}
	for _, want := range in {
		var buf [recordSize]byte
		if err := encodeTrade(want, buf[:]); err != nil {
			t.Fatalf("encodeTrade: %v", err)
		}
		if got := decodeTrade(buf[:]); got != want {
			t.Fatalf("round trip = %+v, want %+v", got, want)
		}
	}
}

func TestEncodeTradeShortBuffer(t *testing.T) {
	small := make([]byte, recordSize-1)
	if err := encodeTrade(trade.Trade{}, small); err != errShortBuffer {
		t.Fatalf("err = %v, want errShortBuffer", err)
	}
}

func TestEncodeTradeSideByte(t *testing.T) {
	var buf [recordSize]byte
	_ = encodeTrade(trade.Trade{AggressorSide: order.Sell}, buf[:])
	if buf[24] != 1 {
		t.Fatalf("sell side byte = %d, want 1", buf[24])
	}
	_ = encodeTrade(trade.Trade{AggressorSide: order.Buy}, buf[:])
	if buf[24] != 0 {
		t.Fatalf("buy side byte = %d, want 0", buf[24])
	}
}

// TestStreamTradesGoldenHex pins the exact 25-byte little-endian layout against
// the tick store's wire contract. If these bytes drift, ingestion breaks.
func TestStreamTradesGoldenHex(t *testing.T) {
	trades := []trade.Trade{
		{MakerID: 10, TakerID: 20, TS: 1, Price: 100, Qty: 5, AggressorSide: order.Buy},
		{MakerID: 11, TakerID: 21, TS: 2, Price: 101, Qty: 2, AggressorSide: order.Sell},
	}
	var out bytes.Buffer
	if err := streamTrades(&out, trades); err != nil {
		t.Fatalf("streamTrades: %v", err)
	}
	want := "01000000000000006400000000000000050000000000000000" +
		"02000000000000006500000000000000020000000000000001"
	if got := hex.EncodeToString(out.Bytes()); got != want {
		t.Fatalf("stream bytes =\n%s\nwant\n%s", got, want)
	}
	if out.Len() != 2*recordSize {
		t.Fatalf("stream len = %d, want %d", out.Len(), 2*recordSize)
	}
}
