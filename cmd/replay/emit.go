package main

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/jayrajadeja/lob/trade"
)

// recordSize is the fixed on-wire size of one encoded trade, in bytes. This
// layout is the contract shared with the tick store (github.com/jayrajadeja/
// tickstore); it is intentionally duplicated here so the two projects share no
// code, only the byte format. TestStreamTradesGoldenHex guards against drift.
const recordSize = 25

// errShortBuffer is returned when the destination buffer is smaller than one record.
var errShortBuffer = errors.New("emit: buffer smaller than recordSize")

// encodeTrade writes t into buf (len must be >= recordSize) as 25 little-endian
// bytes: TS int64 | Price int64 | Qty uint64 | AggressorSide uint8.
func encodeTrade(t trade.Trade, buf []byte) error {
	if len(buf) < recordSize {
		return errShortBuffer
	}
	binary.LittleEndian.PutUint64(buf[0:8], uint64(t.TS))
	binary.LittleEndian.PutUint64(buf[8:16], uint64(t.Price))
	binary.LittleEndian.PutUint64(buf[16:24], t.Qty)
	buf[24] = byte(t.AggressorSide)
	return nil
}

// streamTrades encodes each trade and writes it to w as a bare record stream
// (no header, no delimiters). Buffering is the caller's responsibility.
func streamTrades(w io.Writer, trades []trade.Trade) error {
	var buf [recordSize]byte
	for _, t := range trades {
		if err := encodeTrade(t, buf[:]); err != nil {
			return err
		}
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	}
	return nil
}
