package domain

import (
	"errors"
	"testing"
)

func TestNewOrder_TotalCalculation(t *testing.T) {
	items := []OrderItem{
		{ProductID: "p1", Quantity: 2, Price: 1000}, // 2 Ã— $10.00 = $20.00
		{ProductID: "p2", Quantity: 1, Price: 500},  // 1 Ã— $5.00  = $5.00
	}
	order := NewOrder("user-1", "USD", "idem-1", items, nil)

	if order.TotalAmount != 2500 {
		t.Errorf("expected total 2500, got %d", order.TotalAmount)
	}
	if order.Status != StatusPending {
		t.Errorf("expected status pending, got %s", order.Status)
	}
	if order.ID == "" {
		t.Error("expected non-empty ID")
	}
	for i, item := range order.Items {
		if item.ID == "" {
			t.Errorf("item[%d] has empty ID", i)
		}
		if item.OrderID != order.ID {
			t.Errorf("item[%d] OrderID mismatch", i)
		}
	}
}

func TestOrder_Transition_ValidPaths(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*Order) // get the order into the start state
		to    OrderStatus
	}{
		{"pendingâ†’confirmed", func(o *Order) {}, StatusConfirmed},
		{"pendingâ†’cancelled", func(o *Order) {}, StatusCancelled},
		{"confirmedâ†’paid", func(o *Order) { _ = o.Transition(StatusConfirmed) }, StatusPaid},
		{"confirmedâ†’cancelled", func(o *Order) { _ = o.Transition(StatusConfirmed) }, StatusCancelled},
		{"paidâ†’refunded", func(o *Order) {
			_ = o.Transition(StatusConfirmed)
			_ = o.Transition(StatusPaid)
		}, StatusRefunded},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			order := NewOrder("u", "USD", "k", nil, nil)
			tc.setup(order)
			if err := order.Transition(tc.to); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if order.Status != tc.to {
				t.Errorf("expected status %s, got %s", tc.to, order.Status)
			}
		})
	}
}

func TestOrder_Transition_InvalidPaths(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*Order)
		to    OrderStatus
	}{
		{"pendingâ†’paid", func(o *Order) {}, StatusPaid},
		{"pendingâ†’refunded", func(o *Order) {}, StatusRefunded},
		{"paidâ†’cancelled", func(o *Order) {
			_ = o.Transition(StatusConfirmed)
			_ = o.Transition(StatusPaid)
		}, StatusCancelled},
		{"cancelledâ†’paid", func(o *Order) { _ = o.Transition(StatusCancelled) }, StatusPaid},
		{"refundedâ†’paid", func(o *Order) {
			_ = o.Transition(StatusConfirmed)
			_ = o.Transition(StatusPaid)
			_ = o.Transition(StatusRefunded)
		}, StatusPaid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			order := NewOrder("u", "USD", "k", nil, nil)
			tc.setup(order)
			err := order.Transition(tc.to)
			if err == nil {
				t.Errorf("expected error for invalid transition, got nil")
			}
			var transErr *InvalidTransitionError
			if !errors.As(err, &transErr) {
				t.Errorf("expected *InvalidTransitionError, got %T", err)
			}
		})
	}
}

func TestOrder_Transition_StatusUnchangedOnError(t *testing.T) {
	order := NewOrder("u", "USD", "k", nil, nil)
	before := order.Status

	_ = order.Transition(StatusPaid) // invalid from pending

	if order.Status != before {
		t.Errorf("status should not change on invalid transition: was %s, now %s", before, order.Status)
	}
}

func TestInvalidTransitionError_Message(t *testing.T) {
	err := &InvalidTransitionError{From: StatusPending, To: StatusPaid}
	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}
