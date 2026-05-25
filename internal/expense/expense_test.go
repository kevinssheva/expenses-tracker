package expense

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewUsesProvidedTimestamp(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	defaultTime := time.Date(2026, 5, 25, 9, 0, 0, 0, location)

	exp, err := New("exp_1", CreateRequest{
		Timestamp:     "2026-05-24T20:30:00+07:00",
		Description:   "  kopi susu  ",
		Category:      "food",
		Amount:        28000,
		PaymentMethod: "QRIS",
		RawMessage:    "kopi susu 28k qris",
	}, defaultTime)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if exp.ID != "exp_1" {
		t.Fatalf("ID = %q, want exp_1", exp.ID)
	}
	if !exp.Timestamp.Equal(time.Date(2026, 5, 24, 20, 30, 0, 0, location)) {
		t.Fatalf("Timestamp = %v, want provided timestamp", exp.Timestamp)
	}
	if exp.Description != "kopi susu" {
		t.Fatalf("Description = %q, want kopi susu", exp.Description)
	}
	if exp.Source != "nanoclaw" {
		t.Fatalf("Source = %q, want nanoclaw", exp.Source)
	}
}

func TestNewUsesDefaultTimestampWhenMissing(t *testing.T) {
	defaultTime := time.Date(2026, 5, 25, 9, 0, 0, 0, time.UTC)

	exp, err := New("exp_1", CreateRequest{Description: "lunch", Amount: 50000}, defaultTime)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if !exp.Timestamp.Equal(defaultTime) {
		t.Fatalf("Timestamp = %v, want %v", exp.Timestamp, defaultTime)
	}
}

func TestNewRejectsInvalidExpense(t *testing.T) {
	tests := []struct {
		name string
		req  CreateRequest
	}{
		{name: "empty description", req: CreateRequest{Description: "   ", Amount: 50000}},
		{name: "zero amount", req: CreateRequest{Description: "lunch", Amount: 0}},
		{name: "negative amount", req: CreateRequest{Description: "lunch", Amount: -1}},
		{name: "invalid timestamp", req: CreateRequest{Timestamp: "not-a-time", Description: "lunch", Amount: 50000}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New("exp_1", tt.req, time.Now())
			if err == nil {
				t.Fatal("New returned nil error, want error")
			}
		})
	}
}

func TestApplyUpdateChangesOnlyProvidedFields(t *testing.T) {
	originalTime := time.Date(2026, 5, 25, 9, 0, 0, 0, time.UTC)
	exp := Expense{
		ID:            "exp_1",
		Timestamp:     originalTime,
		Description:   "lunch",
		Category:      "food",
		Amount:        50000,
		PaymentMethod: "cash",
		Source:        "nanoclaw",
		RawMessage:    "lunch 50000",
	}
	description := "dinner"
	amount := int64(75000)
	timestamp := "2026-05-25T19:00:00Z"

	updated, err := exp.ApplyUpdate(UpdateRequest{
		Timestamp:   &timestamp,
		Description: &description,
		Amount:      &amount,
	})
	if err != nil {
		t.Fatalf("ApplyUpdate returned error: %v", err)
	}

	if updated.Description != "dinner" {
		t.Fatalf("Description = %q, want dinner", updated.Description)
	}
	if updated.Amount != 75000 {
		t.Fatalf("Amount = %d, want 75000", updated.Amount)
	}
	if updated.Category != "food" {
		t.Fatalf("Category = %q, want food", updated.Category)
	}
	if !updated.Timestamp.Equal(time.Date(2026, 5, 25, 19, 0, 0, 0, time.UTC)) {
		t.Fatalf("Timestamp = %v, want updated timestamp", updated.Timestamp)
	}
}

func TestApplyUpdateRejectsInvalidResult(t *testing.T) {
	exp := Expense{ID: "exp_1", Timestamp: time.Now(), Description: "lunch", Amount: 50000, Source: "nanoclaw"}
	emptyDescription := "   "
	zeroAmount := int64(0)
	invalidTimestamp := "not-a-time"

	tests := []struct {
		name string
		req  UpdateRequest
	}{
		{name: "empty description", req: UpdateRequest{Description: &emptyDescription}},
		{name: "zero amount", req: UpdateRequest{Amount: &zeroAmount}},
		{name: "invalid timestamp", req: UpdateRequest{Timestamp: &invalidTimestamp}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := exp.ApplyUpdate(tt.req)
			if err == nil {
				t.Fatal("ApplyUpdate returned nil error, want error")
			}
		})
	}
}

func TestExpenseRowMatchesColumnOrder(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	exp := Expense{
		ID:            "exp_1",
		Timestamp:     time.Date(2026, 5, 25, 14, 20, 0, 0, location),
		Description:   "lunch",
		Category:      "food",
		Amount:        50000,
		PaymentMethod: "cash",
		Source:        "nanoclaw",
		RawMessage:    "lunch 50000",
	}

	got := exp.Row(location)
	want := []interface{}{
		"exp_1",
		"2026-05-25T14:20:00+07:00",
		"2026-05-25",
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

func TestExpenseFromRowParsesSheetRow(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	row := []interface{}{"exp_1", "2026-05-25T14:20:00+07:00", "2026-05-25", "14:20", "lunch", "food", "50000", "cash", "nanoclaw", "lunch 50000"}

	exp, err := ExpenseFromRow(row, location)
	if err != nil {
		t.Fatalf("ExpenseFromRow returned error: %v", err)
	}

	if exp.ID != "exp_1" {
		t.Fatalf("ID = %q, want exp_1", exp.ID)
	}
	if !exp.Timestamp.Equal(time.Date(2026, 5, 25, 14, 20, 0, 0, location)) {
		t.Fatalf("Timestamp = %v, want parsed timestamp", exp.Timestamp)
	}
	if exp.Amount != 50000 {
		t.Fatalf("Amount = %d, want 50000", exp.Amount)
	}
}

func TestExpenseFromRowAllowsMissingTrailingOptionalCells(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	row := []interface{}{"exp_1", "2026-05-25T14:20:00+07:00", "2026-05-25", "14:20", "lunch", "food", "50000", "cash", "nanoclaw"}

	exp, err := ExpenseFromRow(row, location)
	if err != nil {
		t.Fatalf("ExpenseFromRow returned error: %v", err)
	}

	if exp.RawMessage != "" {
		t.Fatalf("RawMessage = %q, want empty", exp.RawMessage)
	}
}

func TestExpenseFromRowParsesFormattedIntegerAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount interface{}
	}{
		{name: "comma thousands", amount: "50,000"},
		{name: "dot thousands", amount: "50.000"},
		{name: "float", amount: float64(50000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := []interface{}{"exp_1", "2026-05-25T14:20:00Z", "2026-05-25", "14:20", "lunch", "food", tt.amount, "cash", "nanoclaw", ""}

			exp, err := ExpenseFromRow(row, time.UTC)
			if err != nil {
				t.Fatalf("ExpenseFromRow returned error: %v", err)
			}
			if exp.Amount != 50000 {
				t.Fatalf("Amount = %d, want 50000", exp.Amount)
			}
		})
	}
}

func TestExpenseFromRowParsesLegacyRowWithoutID(t *testing.T) {
	row := []interface{}{"2026-05-25T14:20:00+07:00", "2026-05-25", "14:20", "lunch", "food", "50000", "cash", "nanoclaw", "lunch 50000"}

	exp, err := ExpenseFromRow(row, time.UTC)
	if err != nil {
		t.Fatalf("ExpenseFromRow returned error: %v", err)
	}
	again, err := ExpenseFromRow(row, time.UTC)
	if err != nil {
		t.Fatalf("ExpenseFromRow returned error on second parse: %v", err)
	}

	if !strings.HasPrefix(exp.ID, "exp_legacy_") {
		t.Fatalf("ID = %q, want exp_legacy_ prefix", exp.ID)
	}
	if exp.ID != again.ID {
		t.Fatalf("legacy ID = %q, want stable ID %q", again.ID, exp.ID)
	}
	if exp.Description != "lunch" || exp.Amount != 50000 {
		t.Fatalf("legacy expense = %#v, want parsed description and amount", exp)
	}
}
