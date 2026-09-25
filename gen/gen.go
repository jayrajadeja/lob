// Package gen produces a deterministic, seedable stream of order-book actions
// (limit placements and cancels) for driving and load-testing the engine. It
// depends only on the order data types and never touches the book internals.
package gen

import (
	"math/rand"

	"github.com/jayrajadeja/lob/order"
)

// Kind distinguishes a placement from a cancel.
type Kind uint8

const (
	Place Kind = iota
	CancelOrder
)

// Action is one generated instruction: a Place carries Order; a CancelOrder
// carries CancelID.
type Action struct {
	Kind     Kind
	Order    order.Order
	CancelID uint64
}

// Params configures a generator run.
type Params struct {
	Seed       int64
	Count      int
	Mid        int64
	Tick       int64
	MaxQty     uint64
	CancelProb float64
}

// Generator emits a reproducible action stream for a given seed.
type Generator struct {
	p       Params
	rng     *rand.Rand
	emitted int
	nextID  uint64
	nextTS  int64
	live    []uint64
}

// New creates a generator. A MaxQty of 0 is treated as 1, and a Tick <= 0 as 1.
func New(p Params) *Generator {
	if p.MaxQty == 0 {
		p.MaxQty = 1
	}
	if p.Tick <= 0 {
		p.Tick = 1
	}
	return &Generator{
		p:      p,
		rng:    rand.New(rand.NewSource(p.Seed)),
		nextID: 1,
		nextTS: 1,
	}
}

// Next returns the next action, or ok=false once Count actions are produced.
func (g *Generator) Next() (Action, bool) {
	if g.emitted >= g.p.Count {
		return Action{}, false
	}
	g.emitted++

	if len(g.live) > 0 && g.rng.Float64() < g.p.CancelProb {
		idx := g.rng.Intn(len(g.live))
		id := g.live[idx]
		g.live[idx] = g.live[len(g.live)-1]
		g.live = g.live[:len(g.live)-1]
		return Action{Kind: CancelOrder, CancelID: id}, true
	}

	side := order.Buy
	if g.rng.Intn(2) == 1 {
		side = order.Sell
	}
	step := int64(g.rng.Intn(11) - 5) // [-5, 5]
	price := g.p.Mid + step*g.p.Tick
	if price < g.p.Tick {
		price = g.p.Tick
	}
	qty := uint64(g.rng.Intn(int(g.p.MaxQty))) + 1

	o := order.Order{
		ID:    g.nextID,
		Side:  side,
		Price: price,
		Qty:   qty,
		TS:    g.nextTS,
	}
	g.nextID++
	g.nextTS++
	g.live = append(g.live, o.ID)
	return Action{Kind: Place, Order: o}, true
}
