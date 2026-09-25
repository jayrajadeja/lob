package order

import (
	"errors"
	"testing"
)

func TestSideString(t *testing.T) {
	if Buy.String() != "BUY" {
		t.Errorf("Buy.String() = %q, want BUY", Buy.String())
	}
	if Sell.String() != "SELL" {
		t.Errorf("Sell.String() = %q, want SELL", Sell.String())
	}
}

func TestOrderValidate(t *testing.T) {
	tests := []struct {
		name string
		o    Order
		want error
	}{
		{"ok", Order{ID: 1, Side: Buy, Price: 100, Qty: 5, TS: 1}, nil},
		{"zero qty", Order{ID: 2, Side: Buy, Price: 100, Qty: 0, TS: 1}, ErrInvalidQty},
		{"zero price", Order{ID: 3, Side: Sell, Price: 0, Qty: 5, TS: 1}, ErrInvalidPrice},
		{"negative price", Order{ID: 4, Side: Sell, Price: -1, Qty: 5, TS: 1}, ErrInvalidPrice},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.Validate(); !errors.Is(got, tt.want) {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
