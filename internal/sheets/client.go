package sheets

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kevinssheva/expenses-tracker/internal/expense"
	"google.golang.org/api/option"
	sheetsapi "google.golang.org/api/sheets/v4"
)

var headerRow = []interface{}{"ID", "Timestamp", "Date", "Time", "Description", "Category", "Amount", "Payment Method", "Source", "Raw Message"}

type Client struct {
	service  *sheetsapi.Service
	sheetID  string
	range_   string
	location *time.Location
}

func NewClient(ctx context.Context, credentialsFile, sheetID, sheetRange string, location *time.Location) (*Client, error) {
	service, err := sheetsapi.NewService(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}

	return &Client{service: service, sheetID: sheetID, range_: sheetRange, location: location}, nil
}

func (c *Client) ListExpenses(ctx context.Context) ([]expense.Expense, error) {
	valueRange, err := c.service.Spreadsheets.Values.Get(c.sheetID, c.range_).
		ValueRenderOption("UNFORMATTED_VALUE").
		DateTimeRenderOption("FORMATTED_STRING").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("read expenses from sheet: %w", err)
	}

	expenses, err := expensesFromValues(valueRange.Values, c.location)
	if err != nil {
		return nil, err
	}
	return expenses, nil

}

func (c *Client) SaveExpenses(ctx context.Context, expenses []expense.Expense) error {
	oldValueRange, err := c.service.Spreadsheets.Values.Get(c.sheetID, c.range_).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("read existing expenses sheet: %w", err)
	}

	values := valuesForExpenses(expenses, c.location)
	valueRange := &sheetsapi.ValueRange{Values: values}
	_, err = c.service.Spreadsheets.Values.Update(c.sheetID, c.range_, valueRange).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("write expenses to sheet: %w", err)
	}

	if clearRange, ok := tailRange(c.range_, len(values), len(oldValueRange.Values)); ok {
		_, err := c.service.Spreadsheets.Values.Clear(c.sheetID, clearRange, &sheetsapi.ClearValuesRequest{}).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("clear old expense rows: %w", err)
		}
	}
	return nil
}

func valuesForExpenses(expenses []expense.Expense, location *time.Location) [][]interface{} {
	values := make([][]interface{}, 0, len(expenses)+1)
	values = append(values, headerRow)
	for _, exp := range expenses {
		values = append(values, exp.Row(location))
	}
	return values
}

func expensesFromValues(values [][]interface{}, location *time.Location) ([]expense.Expense, error) {
	if len(values) == 0 {
		return nil, nil
	}

	start := 0
	if isHeaderRow(values[0]) {
		start = 1
	}
	expenses := make([]expense.Expense, 0, len(values)-start)
	for _, row := range values[start:] {
		if len(row) == 0 {
			continue
		}
		exp, err := expense.ExpenseFromRow(row, location)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, exp)
	}
	return expenses, nil
}

func isHeaderRow(row []interface{}) bool {
	if len(row) == 0 {
		return false
	}
	first := strings.TrimSpace(fmt.Sprint(row[0]))
	return strings.EqualFold(first, "ID") || strings.EqualFold(first, "Timestamp")
}

func tailRange(sheetRange string, newRowCount, oldRowCount int) (string, bool) {
	if newRowCount >= oldRowCount {
		return "", false
	}

	parts := strings.SplitN(sheetRange, "!", 2)
	if len(parts) != 2 {
		return "", false
	}
	columns := strings.SplitN(parts[1], ":", 2)
	if len(columns) != 2 {
		return "", false
	}

	return fmt.Sprintf("%s!%s%d:%s%d", parts[0], columns[0], newRowCount+1, columns[1], oldRowCount), true
}
