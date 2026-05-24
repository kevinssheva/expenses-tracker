package sheets

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	sheetsapi "google.golang.org/api/sheets/v4"
)

type Client struct {
	service *sheetsapi.Service
	sheetID string
	range_  string
}

func NewClient(ctx context.Context, credentialsFile, sheetID, sheetRange string) (*Client, error) {
	service, err := sheetsapi.NewService(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}

	return &Client{service: service, sheetID: sheetID, range_: sheetRange}, nil
}

func (c *Client) AppendExpense(ctx context.Context, row []interface{}) error {
	valueRange := &sheetsapi.ValueRange{Values: [][]interface{}{row}}
	_, err := c.service.Spreadsheets.Values.Append(c.sheetID, c.range_, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("append row to sheet: %w", err)
	}
	return nil
}
