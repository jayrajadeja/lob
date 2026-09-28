// Command replay drives the matching engine with the synthetic generator and
// prints the resulting trades, final book depth, and throughput. This is the
// only package in the project that performs I/O.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
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
	emit := flag.Bool("emit", false, "emit trades as a binary record stream to stdout (summary goes to stderr)")
	flag.Parse()

	// When emitting, stdout carries the raw trade stream, so the human-readable
	// summary is routed to stderr.
	sumOut := io.Writer(os.Stdout)
	var bw *bufio.Writer
	if *emit {
		sumOut = os.Stderr
		bw = bufio.NewWriter(os.Stdout)
	}

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
	var emitErr error
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
			if *emit {
				if err := streamTrades(bw, trades); err != nil {
					emitErr = err
				}
			}
		case gen.CancelOrder:
			b.Cancel(a.CancelID)
		}
		if emitErr != nil {
			break
		}
	}
	elapsed := time.Since(start)

	if bw != nil {
		if err := bw.Flush(); err != nil && emitErr == nil {
			emitErr = err
		}
	}
	if emitErr != nil {
		fmt.Fprintf(os.Stderr, "emit error: %v\n", emitErr)
	}

	bids, asks := b.Depth(5)
	fmt.Fprintf(sumOut, "symbol=%s actions=%d trades=%d\n", b.Symbol(), *orders, totalTrades)
	fmt.Fprintln(sumOut, "top bids (price x qty):")
	for _, l := range bids {
		fmt.Fprintf(sumOut, "  %d x %d\n", l.Price, l.Qty)
	}
	fmt.Fprintln(sumOut, "top asks (price x qty):")
	for _, l := range asks {
		fmt.Fprintf(sumOut, "  %d x %d\n", l.Price, l.Qty)
	}
	if elapsed > 0 {
		fmt.Fprintf(sumOut, "throughput=%.0f actions/sec\n", float64(*orders)/elapsed.Seconds())
	}
	if addErr != nil || emitErr != nil {
		os.Exit(1)
	}
}
