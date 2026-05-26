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

type Client struct {
	service    *sheetsapi.Service
	sheetID    string
	sheetTabID int64
	sheetRange string
	location   *time.Location
}

type expenseRow struct {
	rowNumber int64
	expense   expense.Expense
}

func NewClient(ctx context.Context, credentialsFile, sheetID, sheetRange string, location *time.Location) (*Client, error) {
	service, err := sheetsapi.NewService(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}
	if err := validateFullColumnRange(sheetRange); err != nil {
		return nil, err
	}
	sheetTabID, err := resolveSheetTabID(ctx, service, sheetID, sheetRange)
	if err != nil {
		return nil, err
	}

	return &Client{service: service, sheetID: sheetID, sheetTabID: sheetTabID, sheetRange: sheetRange, location: location}, nil
}

func (c *Client) ListExpenses(ctx context.Context) ([]expense.Expense, error) {
	rows, err := c.readExpenseRows(ctx)
	if err != nil {
		return nil, err
	}

	expenses := make([]expense.Expense, 0, len(rows))
	for _, row := range rows {
		expenses = append(expenses, row.expense)
	}
	return expenses, nil
}

func (c *Client) AppendExpense(ctx context.Context, exp expense.Expense) error {
	valueRange := &sheetsapi.ValueRange{Values: [][]interface{}{exp.Row(c.location)}}
	_, err := c.service.Spreadsheets.Values.Append(c.sheetID, c.sheetRange, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("append expense row: %w", err)
	}
	return nil
}

func (c *Client) UpdateExpense(ctx context.Context, exp expense.Expense) error {
	row, err := c.findExpenseRow(ctx, exp.ID)
	if err != nil {
		return err
	}
	updateRange, err := rowRange(c.sheetRange, row.rowNumber)
	if err != nil {
		return err
	}

	valueRange := &sheetsapi.ValueRange{Values: [][]interface{}{exp.Row(c.location)}}
	_, err = c.service.Spreadsheets.Values.Update(c.sheetID, updateRange, valueRange).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("update expense row: %w", err)
	}
	return nil
}

func (c *Client) DeleteExpense(ctx context.Context, id string) error {
	row, err := c.findExpenseRow(ctx, id)
	if err != nil {
		return err
	}

	_, err = c.service.Spreadsheets.BatchUpdate(c.sheetID, &sheetsapi.BatchUpdateSpreadsheetRequest{
		Requests: []*sheetsapi.Request{{
			DeleteDimension: &sheetsapi.DeleteDimensionRequest{
				Range: &sheetsapi.DimensionRange{
					SheetId:    c.sheetTabID,
					Dimension:  "ROWS",
					StartIndex: row.rowNumber - 1,
					EndIndex:   row.rowNumber,
				},
			},
		}},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("delete expense row: %w", err)
	}
	return nil
}

func (c *Client) SortExpenses(ctx context.Context) error {
	valueRange, err := c.service.Spreadsheets.Values.Get(c.sheetID, c.sheetRange).
		ValueRenderOption("UNFORMATTED_VALUE").
		DateTimeRenderOption("FORMATTED_STRING").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("read expenses for sort: %w", err)
	}
	if len(valueRange.Values)-int(dataStartIndex(valueRange.Values)) <= 1 {
		return nil
	}
	startColumnIndex, endColumnIndex, err := columnIndexesFromRange(c.sheetRange)
	if err != nil {
		return err
	}
	dateColumnIndex, err := dateColumnIndexFromRange(c.sheetRange)
	if err != nil {
		return err
	}

	_, err = c.service.Spreadsheets.BatchUpdate(c.sheetID, &sheetsapi.BatchUpdateSpreadsheetRequest{
		Requests: []*sheetsapi.Request{{
			SortRange: &sheetsapi.SortRangeRequest{
				Range: &sheetsapi.GridRange{
					SheetId:          c.sheetTabID,
					StartRowIndex:    dataStartIndex(valueRange.Values),
					EndRowIndex:      dataEndIndex(valueRange.Values),
					StartColumnIndex: startColumnIndex,
					EndColumnIndex:   endColumnIndex,
				},
				SortSpecs: []*sheetsapi.SortSpec{{
					DimensionIndex: dateColumnIndex,
					SortOrder:      "ASCENDING",
				}},
			},
		}},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("sort expense rows: %w", err)
	}
	return nil
}

func (c *Client) readExpenseRows(ctx context.Context) ([]expenseRow, error) {
	valueRange, err := c.service.Spreadsheets.Values.Get(c.sheetID, c.sheetRange).
		ValueRenderOption("UNFORMATTED_VALUE").
		DateTimeRenderOption("FORMATTED_STRING").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("read expenses from sheet: %w", err)
	}
	return expenseRowsFromValues(valueRange.Values, c.location)
}

func (c *Client) findExpenseRow(ctx context.Context, id string) (expenseRow, error) {
	rows, err := c.readExpenseRows(ctx)
	if err != nil {
		return expenseRow{}, err
	}
	for _, row := range rows {
		if row.expense.ID == id {
			return row, nil
		}
	}
	return expenseRow{}, expense.ErrNotFound
}

func expensesFromValues(values [][]interface{}, location *time.Location) ([]expense.Expense, error) {
	rows, err := expenseRowsFromValues(values, location)
	if err != nil {
		return nil, err
	}

	expenses := make([]expense.Expense, 0, len(rows))
	for _, row := range rows {
		expenses = append(expenses, row.expense)
	}
	return expenses, nil
}

func expenseRowsFromValues(values [][]interface{}, location *time.Location) ([]expenseRow, error) {
	start := dataStartIndex(values)
	rows := make([]expenseRow, 0, len(values)-int(start))
	for i, valueRow := range values[start:] {
		rowNumber := int64(i) + start + 1
		if len(valueRow) == 0 {
			continue
		}
		exp, err := expense.ExpenseFromRow(valueRow, location)
		if err != nil {
			return nil, fmt.Errorf("parse expense row %d: %w", rowNumber, err)
		}
		rows = append(rows, expenseRow{rowNumber: rowNumber, expense: exp})
	}
	return rows, nil
}

func isHeaderRow(row []interface{}) bool {
	if len(row) == 0 {
		return false
	}
	first := strings.TrimSpace(fmt.Sprint(row[0]))
	return strings.EqualFold(first, "Date")
}

func rowRange(sheetRange string, rowNumber int64) (string, error) {
	parts := strings.SplitN(sheetRange, "!", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("sheet range must include sheet name")
	}
	columns := strings.SplitN(parts[1], ":", 2)
	if len(columns) != 2 {
		return "", fmt.Errorf("sheet range must include start and end columns")
	}
	startColumn := columnLetters(columns[0])
	endColumn := columnLetters(columns[1])
	if startColumn == "" || endColumn == "" {
		return "", fmt.Errorf("sheet range must include column letters")
	}
	return fmt.Sprintf("%s!%s%d:%s%d", parts[0], startColumn, rowNumber, endColumn, rowNumber), nil
}

func validateFullColumnRange(sheetRange string) error {
	parts := strings.SplitN(sheetRange, "!", 2)
	if len(parts) != 2 {
		return fmt.Errorf("sheet range must include sheet name")
	}
	columns := strings.SplitN(parts[1], ":", 2)
	if len(columns) != 2 {
		return fmt.Errorf("sheet range must include start and end columns")
	}
	startColumn := columnLetters(columns[0])
	endColumn := columnLetters(columns[1])
	if startColumn == "" || endColumn == "" {
		return fmt.Errorf("sheet range must include column letters")
	}
	if strings.TrimSpace(columns[0]) != startColumn || strings.TrimSpace(columns[1]) != endColumn {
		return fmt.Errorf("sheet range must use full columns like Expenses!A:G")
	}
	return nil
}

func sheetNameFromRange(sheetRange string) (string, error) {
	parts := strings.SplitN(sheetRange, "!", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return "", fmt.Errorf("sheet range must include sheet name")
	}
	sheetName := strings.TrimSpace(parts[0])
	if strings.HasPrefix(sheetName, "'") && strings.HasSuffix(sheetName, "'") && len(sheetName) >= 2 {
		sheetName = strings.TrimPrefix(strings.TrimSuffix(sheetName, "'"), "'")
		sheetName = strings.ReplaceAll(sheetName, "''", "'")
	}
	return sheetName, nil
}

func dataStartIndex(values [][]interface{}) int64 {
	if len(values) == 0 {
		return 0
	}
	if isHeaderRow(values[0]) {
		return 1
	}
	return 0
}

func dataEndIndex(values [][]interface{}) int64 {
	return int64(len(values))
}

func resolveSheetTabID(ctx context.Context, service *sheetsapi.Service, spreadsheetID, sheetRange string) (int64, error) {
	sheetName, err := sheetNameFromRange(sheetRange)
	if err != nil {
		return 0, err
	}
	spreadsheet, err := service.Spreadsheets.Get(spreadsheetID).
		Fields("sheets(properties(sheetId,title))").
		Context(ctx).
		Do()
	if err != nil {
		return 0, fmt.Errorf("read spreadsheet metadata: %w", err)
	}
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties != nil && sheet.Properties.Title == sheetName {
			return sheet.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("sheet %q not found", sheetName)
}

func columnIndexesFromRange(sheetRange string) (int64, int64, error) {
	parts := strings.SplitN(sheetRange, "!", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("sheet range must include sheet name")
	}
	columns := strings.SplitN(parts[1], ":", 2)
	if len(columns) != 2 {
		return 0, 0, fmt.Errorf("sheet range must include start and end columns")
	}

	start, err := columnIndex(columnLetters(columns[0]))
	if err != nil {
		return 0, 0, err
	}
	end, err := columnIndex(columnLetters(columns[1]))
	if err != nil {
		return 0, 0, err
	}
	return start, end + 1, nil
}

func dateColumnIndexFromRange(sheetRange string) (int64, error) {
	start, _, err := columnIndexesFromRange(sheetRange)
	if err != nil {
		return 0, err
	}
	return start, nil
}

func columnLetters(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			builder.WriteRune(r)
		}
	}
	return strings.ToUpper(builder.String())
}

func columnIndex(column string) (int64, error) {
	if column == "" {
		return 0, fmt.Errorf("column is required")
	}
	var index int64
	for _, r := range column {
		if r < 'A' || r > 'Z' {
			return 0, fmt.Errorf("invalid column %q", column)
		}
		index = index*26 + int64(r-'A'+1)
	}
	return index - 1, nil
}
