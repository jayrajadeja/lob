package book

import "github.com/jayrajadeja/lob/order"

// priceLevel is the set of resting orders at one price, held as a FIFO queue:
// the head (index 0) is the oldest order and matches first, enforcing time
// priority within a price.
type priceLevel struct {
	price  int64
	orders []order.Order
}

func newPriceLevel(price int64) *priceLevel {
	return &priceLevel{price: price}
}

// push adds an order to the back of the queue.
func (l *priceLevel) push(o order.Order) {
	l.orders = append(l.orders, o)
}

// front returns the head order without removing it; ok is false if empty.
func (l *priceLevel) front() (order.Order, bool) {
	if len(l.orders) == 0 {
		return order.Order{}, false
	}
	return l.orders[0], true
}

// reduceFront subtracts qty from the head order, popping it if it reaches zero.
// The caller guarantees qty <= head quantity.
func (l *priceLevel) reduceFront(qty uint64) {
	if len(l.orders) == 0 {
		return
	}
	l.orders[0].Qty -= qty
	if l.orders[0].Qty == 0 {
		l.orders = l.orders[1:]
	}
}

// removeByID removes the first order with the given id, preserving FIFO order.
// It reports whether an order was removed.
func (l *priceLevel) removeByID(id uint64) bool {
	for i := range l.orders {
		if l.orders[i].ID == id {
			l.orders = append(l.orders[:i], l.orders[i+1:]...)
			return true
		}
	}
	return false
}

func (l *priceLevel) totalQty() uint64 {
	var sum uint64
	for i := range l.orders {
		sum += l.orders[i].Qty
	}
	return sum
}

func (l *priceLevel) isEmpty() bool {
	return len(l.orders) == 0
}
