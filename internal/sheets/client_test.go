package sheets

import (
	"strings"
	"testing"
	"time"
)

func TestExpensesFromValuesSkipsHeader(t *testing.T) {
	values := [][]interface{}{
		{"Date", "Description", "Category", "Amount", "Payment Method", "Account/Wallet", "ID"},
		{"2026-05-25", "Makan ayam geprek", "Food", "35000", "QRIS", "GoPay", "exp_1"},
	}

	got, err := expensesFromValues(values, time.UTC)
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

func TestExpenseRowsFromValuesTracksRowsWithHeader(t *testing.T) {
	values := [][]interface{}{
		{"Date", "Description", "Category", "Amount", "Payment Method", "Account/Wallet", "ID"},
		{"2026-05-25", "Makan ayam geprek", "Food", "35000", "QRIS", "GoPay", "exp_1"},
	}

	got, err := expenseRowsFromValues(values, time.UTC)
	if err != nil {
		t.Fatalf("expenseRowsFromValues returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(got))
	}
	if got[0].rowNumber != 2 {
		t.Fatalf("rowNumber = %d, want 2", got[0].rowNumber)
	}
	if got[0].expense.ID != "exp_1" {
		t.Fatalf("ID = %q, want exp_1", got[0].expense.ID)
	}
}

func TestExpenseRowsFromValuesTracksRowsWithoutHeader(t *testing.T) {
	values := [][]interface{}{
		{"2026-05-25", "Makan ayam geprek", "Food", "35000", "QRIS", "GoPay", "exp_1"},
	}

	got, err := expenseRowsFromValues(values, time.UTC)
	if err != nil {
		t.Fatalf("expenseRowsFromValues returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(got))
	}
	if got[0].rowNumber != 1 {
		t.Fatalf("rowNumber = %d, want 1", got[0].rowNumber)
	}
}

func TestExpenseRowsFromValuesIncludesRowNumberInParseError(t *testing.T) {
	values := [][]interface{}{
		{"Date", "Description", "Category", "Amount", "Payment Method", "Account/Wallet", "ID"},
		{"2026-05-25", "Makan ayam geprek", "Food", "not-a-number", "QRIS", "GoPay", "exp_1"},
	}

	_, err := expenseRowsFromValues(values, time.UTC)
	if err == nil {
		t.Fatal("expenseRowsFromValues returned nil error, want parse error")
	}
	if !strings.Contains(err.Error(), "parse expense row 2") {
		t.Fatalf("error = %q, want row number context", err.Error())
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

func TestRowRangeReturnsExactRowRange(t *testing.T) {
	got, err := rowRange("Sheet1!A:G", 2)
	if err != nil {
		t.Fatalf("rowRange returned error: %v", err)
	}
	want := "Sheet1!A2:G2"
	if got != want {
		t.Fatalf("rowRange = %q, want %q", got, want)
	}
}

func TestValidateFullColumnRangeRejectsRowNumbers(t *testing.T) {
	if err := validateFullColumnRange("Expenses!A:G"); err != nil {
		t.Fatalf("validateFullColumnRange returned error: %v", err)
	}

	err := validateFullColumnRange("Expenses!A2:G")
	if err == nil {
		t.Fatal("validateFullColumnRange returned nil error, want row-number error")
	}
	if !strings.Contains(err.Error(), "must use full columns like Expenses!A:G") {
		t.Fatalf("error = %q, want full-column guidance", err.Error())
	}
}

func TestSheetNameFromRange(t *testing.T) {
	got, err := sheetNameFromRange("Sheet1!A:G")
	if err != nil {
		t.Fatalf("sheetNameFromRange returned error: %v", err)
	}
	if got != "Sheet1" {
		t.Fatalf("sheetNameFromRange = %q, want Sheet1", got)
	}
}

func TestSheetNameFromRangeUnquotesSheetName(t *testing.T) {
	got, err := sheetNameFromRange("'Expense Log'!A:G")
	if err != nil {
		t.Fatalf("sheetNameFromRange returned error: %v", err)
	}
	if got != "Expense Log" {
		t.Fatalf("sheetNameFromRange = %q, want Expense Log", got)
	}
}

func TestSheetNameFromRangeUnescapesSingleQuote(t *testing.T) {
	got, err := sheetNameFromRange("'Kevin''s Expenses'!A:G")
	if err != nil {
		t.Fatalf("sheetNameFromRange returned error: %v", err)
	}
	if got != "Kevin's Expenses" {
		t.Fatalf("sheetNameFromRange = %q, want Kevin's Expenses", got)
	}
}

func TestColumnIndexesFromRange(t *testing.T) {
	start, end, err := columnIndexesFromRange("Expenses!B:H")
	if err != nil {
		t.Fatalf("columnIndexesFromRange returned error: %v", err)
	}
	if start != 1 || end != 8 {
		t.Fatalf("column indexes = %d, %d; want 1, 8", start, end)
	}
}

func TestDateColumnIndexFromRange(t *testing.T) {
	got, err := dateColumnIndexFromRange("Expenses!B:H")
	if err != nil {
		t.Fatalf("dateColumnIndexFromRange returned error: %v", err)
	}
	if got != 1 {
		t.Fatalf("date column index = %d, want 1", got)
	}
}

func TestDataStartIndex(t *testing.T) {
	withHeader := [][]interface{}{{"Date", "Description"}, {"2026-05-25", "Makan ayam geprek"}}
	withoutHeader := [][]interface{}{{"2026-05-25", "Makan ayam geprek"}}

	if got := dataStartIndex(withHeader); got != 1 {
		t.Fatalf("dataStartIndex(withHeader) = %d, want 1", got)
	}
	if got := dataStartIndex(withoutHeader); got != 0 {
		t.Fatalf("dataStartIndex(withoutHeader) = %d, want 0", got)
	}
}

func TestDataEndIndex(t *testing.T) {
	values := [][]interface{}{{"Date", "Description"}, {"2026-05-25", "A"}, {"2026-05-26", "B"}}

	if got := dataEndIndex(values); got != 3 {
		t.Fatalf("dataEndIndex = %d, want 3", got)
	}
}
