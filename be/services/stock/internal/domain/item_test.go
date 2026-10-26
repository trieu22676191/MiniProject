package domain

import (
	"errors"
	"math"
	"testing"
)

func TestAdjustOnHand(t *testing.T) {
	tests := []struct {
		name     string
		onHand   int64
		reserved int64
		delta    int64
		want     int64
		wantErr  error
	}{
		{name: "increase", onHand: 5, reserved: 2, delta: 3, want: 8},
		{name: "decrease to reserved floor", onHand: 5, reserved: 2, delta: -3, want: 2},
		{name: "cannot cross reserved floor", onHand: 5, reserved: 2, delta: -4, wantErr: ErrBelowReserved},
		{name: "zero adjustment", onHand: 5, reserved: 2, wantErr: ErrInvalidAdjustment},
		{name: "overflow", onHand: math.MaxInt64, reserved: 0, delta: 1, wantErr: ErrOnHandOverflow},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AdjustOnHand(tc.onHand, tc.reserved, tc.delta)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if err == nil && got != tc.want {
				t.Fatalf("on hand = %d, want %d", got, tc.want)
			}
		})
	}
}
