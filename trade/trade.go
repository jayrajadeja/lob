// Package trade defines the Trade type the matching engine emits when two
// orders match. It is the output stream later projects (the tick store) consume.
package trade

import "github.com/jayrajadeja/lob/order"

// Trade is a single execution: TakerID crossed the book and matched against the
// resting MakerID. Price is always the maker (resting) order's price.
// AggressorSide is the side of the incoming (taker) order that crossed.
type Trade struct {
	MakerID       uint64
	TakerID       uint64
	Price         int64
	Qty           uint64
	TS            int64
	AggressorSide order.Side
}
