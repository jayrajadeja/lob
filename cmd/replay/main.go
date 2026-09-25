// Command replay drives the matching engine with the synthetic generator and
// prints the resulting trades, final book depth, and throughput. This is the
// only package in the project that performs I/O.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jayrajadeja/lob/book"
	"github.com/jayrajadeja/lob/gen"
)

func main() {
	seed := flag.Int64("seed", 1, "generator seed")
	orders := flag.Int("orders", 10000, "number of actions to generate")
	symbol := flag.String("symbol", "SYNTH", "symbol name")
	mid := flag.Int64("mid", 10000, "mid price in ticks")
	tick := flag.Int64("tick", 1, "tick size")
	maxQty := flag.Uint64("maxqty", 10, "max order quantity")
	cancelProb := flag.Float64("cancelprob", 0.1, "cancel probability")
	flag.Parse()

	b := book.New(*symbol)
	g := gen.New(gen.Params{
		Seed:       *seed,
		Count:      *orders,
		Mid:        *mid,
		Tick:       *tick,
		MaxQty:     *maxQty,
		CancelProb: *cancelProb,
	})

	var totalTrades int
	var addErr error
	start := time.Now()
	for {
		a, ok := g.Next()
		if !ok {
			break
		}
		switch a.Kind {
		case gen.Place:
			trades, err := b.Add(a.Order)
			if err != nil {
				fmt.Fprintf(os.Stderr, "add error: %v\n", err)
				addErr = err
				continue
			}
			totalTrades += len(trades)
		case gen.CancelOrder:
			b.Cancel(a.CancelID)
		}
	}
	elapsed := time.Since(start)

	bids, asks := b.Depth(5)
	fmt.Printf("symbol=%s actions=%d trades=%d\n", b.Symbol(), *orders, totalTrades)
	fmt.Println("top bids (price x qty):")
	for _, l := range bids {
		fmt.Printf("  %d x %d\n", l.Price, l.Qty)
	}
	fmt.Println("top asks (price x qty):")
	for _, l := range asks {
		fmt.Printf("  %d x %d\n", l.Price, l.Qty)
	}
	if elapsed > 0 {
		fmt.Printf("throughput=%.0f actions/sec\n", float64(*orders)/elapsed.Seconds())
	}
	if addErr != nil {
		os.Exit(1)
	}
}
