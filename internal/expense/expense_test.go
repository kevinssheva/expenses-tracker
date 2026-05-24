package expense

import (
	"reflect"
	"testing"
	"time"
)

func TestNewValidExpenseTrimsDescriptionAndSource(t *testing.T) {
	exp, err := New(Request{
		Description:   "  lunch  ",
		Category:      "food",
		Amount:        50000,
		PaymentMethod: "cash",
		Source:        "  whatsapp  ",
		RawMessage:    "lunch 50000",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if exp.Description != "lunch" {
		t.Fatalf("Description = %q, want %q", exp.Description, "lunch")
	}
	if exp.Source != "whatsapp" {
		t.Fatalf("Source = %q, want %q", exp.Source, "whatsapp")
	}
}

func TestNewEmptyDescriptionFails(t *testing.T) {
	_, err := New(Request{Description: "   ", Amount: 50000})
	if err == nil {
		t.Fatal("New returned nil error, want error")
	}
}

func TestNewZeroAndNegativeAmountFail(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
	}{
		{name: "zero", amount: 0},
		{name: "negative", amount: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(Request{Description: "lunch", Amount: tt.amount})
			if err == nil {
				t.Fatal("New returned nil error, want error")
			}
		})
	}
}

func TestNewEmptySourceDefaultsToNanoclaw(t *testing.T) {
	exp, err := New(Request{Description: "lunch", Amount: 50000, Source: "   "})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if exp.Source != "nanoclaw" {
		t.Fatalf("Source = %q, want %q", exp.Source, "nanoclaw")
	}
}

func TestExpenseRowMatchesColumnOrder(t *testing.T) {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("LoadLocation returned error: %v", err)
	}
	now := time.Date(2026, 5, 23, 14, 20, 0, 0, location)
	exp := Expense{
		Description:   "lunch",
		Category:      "food",
		Amount:        50000,
		PaymentMethod: "cash",
		Source:        "nanoclaw",
		RawMessage:    "lunch 50000",
	}

	got := exp.Row(now)
	want := []interface{}{
		"2026-05-23T14:20:00+07:00",
		"2026-05-23",
		"14:20",
		"lunch",
		"food",
		int64(50000),
		"cash",
		"nanoclaw",
		"lunch 50000",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Row() = %#v, want %#v", got, want)
	}
}
