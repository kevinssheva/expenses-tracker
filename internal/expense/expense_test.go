package expense

import (
	"reflect"
	"testing"
	"time"
)

func TestNewValidExpenseUsesDateAndAccountWallet(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)

	exp, err := New("exp_1", CreateRequest{
		Date:          "2026-05-25",
		Description:   "  Makan ayam geprek  ",
		Category:      "Food",
		Amount:        35000,
		PaymentMethod: "QRIS",
		AccountWallet: "GoPay",
	}, location)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if exp.ID != "exp_1" {
		t.Fatalf("ID = %q, want exp_1", exp.ID)
	}
	if !exp.Date.Equal(time.Date(2026, 5, 25, 0, 0, 0, 0, location)) {
		t.Fatalf("Date = %v, want 2026-05-25 in location", exp.Date)
	}
	if exp.Description != "Makan ayam geprek" {
		t.Fatalf("Description = %q, want trimmed description", exp.Description)
	}
	if exp.AccountWallet != "GoPay" {
		t.Fatalf("AccountWallet = %q, want GoPay", exp.AccountWallet)
	}
}

func TestNewRejectsInvalidExpense(t *testing.T) {
	tests := []struct {
		name string
		req  CreateRequest
	}{
		{name: "missing date", req: CreateRequest{Description: "lunch", Category: "Food", Amount: 50000}},
		{name: "invalid date", req: CreateRequest{Date: "25-05-2026", Description: "lunch", Category: "Food", Amount: 50000}},
		{name: "empty description", req: CreateRequest{Date: "2026-05-25", Description: "   ", Category: "Food", Amount: 50000}},
		{name: "zero amount", req: CreateRequest{Date: "2026-05-25", Description: "lunch", Category: "Food", Amount: 0}},
		{name: "negative amount", req: CreateRequest{Date: "2026-05-25", Description: "lunch", Category: "Food", Amount: -1}},
		{name: "missing category", req: CreateRequest{Date: "2026-05-25", Description: "lunch", Amount: 50000}},
		{name: "invalid category", req: CreateRequest{Date: "2026-05-25", Description: "lunch", Category: "Salary", Amount: 50000}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New("exp_1", tt.req, time.UTC)
			if err == nil {
				t.Fatal("New returned nil error, want error")
			}
		})
	}
}

func TestAllowedCategoriesPass(t *testing.T) {
	categories := []string{"Food", "Subscription", "Transport", "Shopping", "Bills", "Health", "Entertainment", "Education", "Other"}

	for _, category := range categories {
		t.Run(category, func(t *testing.T) {
			_, err := New("exp_1", CreateRequest{Date: "2026-05-25", Description: "expense", Category: category, Amount: 1000}, time.UTC)
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}
		})
	}
}

func TestApplyUpdateChangesOnlyProvidedFields(t *testing.T) {
	exp := Expense{
		ID:            "exp_1",
		Date:          time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
		Description:   "lunch",
		Category:      "Food",
		Amount:        50000,
		PaymentMethod: "cash",
		AccountWallet: "BCA",
	}
	date := "2026-05-26"
	description := "dinner"
	amount := int64(75000)
	accountWallet := "GoPay"

	updated, err := exp.ApplyUpdate(UpdateRequest{
		Date:          &date,
		Description:   &description,
		Amount:        &amount,
		AccountWallet: &accountWallet,
	}, time.UTC)
	if err != nil {
		t.Fatalf("ApplyUpdate returned error: %v", err)
	}

	if !updated.Date.Equal(time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("Date = %v, want 2026-05-26", updated.Date)
	}
	if updated.Description != "dinner" {
		t.Fatalf("Description = %q, want dinner", updated.Description)
	}
	if updated.Amount != 75000 {
		t.Fatalf("Amount = %d, want 75000", updated.Amount)
	}
	if updated.Category != "Food" {
		t.Fatalf("Category = %q, want unchanged Food", updated.Category)
	}
	if updated.AccountWallet != "GoPay" {
		t.Fatalf("AccountWallet = %q, want GoPay", updated.AccountWallet)
	}
}

func TestApplyUpdateRejectsInvalidResult(t *testing.T) {
	exp := Expense{ID: "exp_1", Date: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), Description: "lunch", Category: "Food", Amount: 50000}
	invalidDate := "not-a-date"
	emptyDescription := "   "
	zeroAmount := int64(0)
	invalidCategory := "Salary"

	tests := []struct {
		name string
		req  UpdateRequest
	}{
		{name: "invalid date", req: UpdateRequest{Date: &invalidDate}},
		{name: "empty description", req: UpdateRequest{Description: &emptyDescription}},
		{name: "zero amount", req: UpdateRequest{Amount: &zeroAmount}},
		{name: "invalid category", req: UpdateRequest{Category: &invalidCategory}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := exp.ApplyUpdate(tt.req, time.UTC)
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
		Date:          time.Date(2026, 5, 25, 0, 0, 0, 0, location),
		Description:   "Makan ayam geprek",
		Category:      "Food",
		Amount:        35000,
		PaymentMethod: "QRIS",
		AccountWallet: "GoPay",
	}

	got := exp.Row(location)
	want := []interface{}{"2026-05-25", "Makan ayam geprek", "Food", int64(35000), "QRIS", "GoPay", "exp_1"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Row() = %#v, want %#v", got, want)
	}
}

func TestExpenseFromRowParsesSheetRow(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	row := []interface{}{"2026-05-25", "Makan ayam geprek", "Food", "35000", "QRIS", "GoPay", "exp_1"}

	exp, err := ExpenseFromRow(row, location)
	if err != nil {
		t.Fatalf("ExpenseFromRow returned error: %v", err)
	}

	if exp.ID != "exp_1" {
		t.Fatalf("ID = %q, want exp_1", exp.ID)
	}
	if !exp.Date.Equal(time.Date(2026, 5, 25, 0, 0, 0, 0, location)) {
		t.Fatalf("Date = %v, want parsed date", exp.Date)
	}
	if exp.Amount != 35000 {
		t.Fatalf("Amount = %d, want 35000", exp.Amount)
	}
}
