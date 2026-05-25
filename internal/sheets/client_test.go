package sheets

import (
	"reflect"
	"testing"
	"time"

	"github.com/kevinssheva/expenses-tracker/internal/expense"
)

func TestValuesForExpensesIncludesHeaderAndRows(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	expenses := []expense.Expense{{
		ID:            "exp_1",
		Timestamp:     time.Date(2026, 5, 25, 14, 20, 0, 0, location),
		Description:   "lunch",
		Category:      "food",
		Amount:        50000,
		PaymentMethod: "cash",
		Source:        "nanoclaw",
		RawMessage:    "lunch 50000",
	}}

	got := valuesForExpenses(expenses, location)
	want := [][]interface{}{
		{"ID", "Timestamp", "Date", "Time", "Description", "Category", "Amount", "Payment Method", "Source", "Raw Message"},
		{"exp_1", "2026-05-25T14:20:00+07:00", "2026-05-25", "14:20", "lunch", "food", int64(50000), "cash", "nanoclaw", "lunch 50000"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("valuesForExpenses() = %#v, want %#v", got, want)
	}
}

func TestExpensesFromValuesSkipsHeader(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	values := [][]interface{}{
		{"ID", "Timestamp", "Date", "Time", "Description", "Category", "Amount", "Payment Method", "Source", "Raw Message"},
		{"exp_1", "2026-05-25T14:20:00+07:00", "2026-05-25", "14:20", "lunch", "food", "50000", "cash", "nanoclaw", "lunch 50000"},
	}

	got, err := expensesFromValues(values, location)
	if err != nil {
		t.Fatalf("expensesFromValues returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(expenses) = %d, want 1", len(got))
	}
	if got[0].ID != "exp_1" {
		t.Fatalf("ID = %q, want exp_1", got[0].ID)
	}
}

func TestExpensesFromValuesReadsLegacyRowsWithoutHeader(t *testing.T) {
	values := [][]interface{}{
		{"2026-05-25T14:20:00+07:00", "2026-05-25", "14:20", "lunch", "food", "50000", "cash", "nanoclaw", "lunch 50000"},
	}

	got, err := expensesFromValues(values, time.UTC)
	if err != nil {
		t.Fatalf("expensesFromValues returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(expenses) = %d, want 1", len(got))
	}
	if got[0].Description != "lunch" {
		t.Fatalf("Description = %q, want lunch", got[0].Description)
	}
}

func TestExpensesFromValuesReturnsEmptyForEmptySheet(t *testing.T) {
	got, err := expensesFromValues(nil, time.UTC)
	if err != nil {
		t.Fatalf("expensesFromValues returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(expenses) = %d, want 0", len(got))
	}
}

func TestTailRangeReturnsRowsAfterReplacementValues(t *testing.T) {
	got, ok := tailRange("Sheet1!A:J", 3, 5)
	if !ok {
		t.Fatal("tailRange ok = false, want true")
	}
	want := "Sheet1!A4:J5"
	if got != want {
		t.Fatalf("tailRange = %q, want %q", got, want)
	}
}

func TestTailRangeReturnsFalseWhenNoOldRowsRemain(t *testing.T) {
	_, ok := tailRange("Sheet1!A:J", 5, 5)
	if ok {
		t.Fatal("tailRange ok = true, want false")
	}
}
