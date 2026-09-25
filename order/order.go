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
