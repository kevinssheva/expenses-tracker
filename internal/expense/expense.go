package expense

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const defaultSource = "nanoclaw"

type CreateRequest struct {
	Timestamp     string `json:"timestamp"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Source        string `json:"source"`
	RawMessage    string `json:"raw_message"`
}

type UpdateRequest struct {
	Timestamp     *string `json:"timestamp"`
	Description   *string `json:"description"`
	Category      *string `json:"category"`
	Amount        *int64  `json:"amount"`
	PaymentMethod *string `json:"payment_method"`
	Source        *string `json:"source"`
	RawMessage    *string `json:"raw_message"`
}

type Expense struct {
	ID            string
	Timestamp     time.Time
	Description   string
	Category      string
	Amount        int64
	PaymentMethod string
	Source        string
	RawMessage    string
}

func New(id string, req CreateRequest, defaultTimestamp time.Time) (Expense, error) {
	timestamp := defaultTimestamp
	if strings.TrimSpace(req.Timestamp) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.Timestamp))
		if err != nil {
			return Expense{}, fmt.Errorf("timestamp must be RFC3339: %w", err)
		}
		timestamp = parsed
	}

	exp := Expense{
		ID:            strings.TrimSpace(id),
		Timestamp:     timestamp,
		Description:   strings.TrimSpace(req.Description),
		Category:      req.Category,
		Amount:        req.Amount,
		PaymentMethod: req.PaymentMethod,
		Source:        sourceOrDefault(req.Source),
		RawMessage:    req.RawMessage,
	}
	if err := exp.validate(); err != nil {
		return Expense{}, err
	}
	return exp, nil
}

func (e Expense) ApplyUpdate(req UpdateRequest) (Expense, error) {
	updated := e

	if req.Timestamp != nil {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.Timestamp))
		if err != nil {
			return Expense{}, fmt.Errorf("timestamp must be RFC3339: %w", err)
		}
		updated.Timestamp = parsed
	}
	if req.Description != nil {
		updated.Description = strings.TrimSpace(*req.Description)
	}
	if req.Category != nil {
		updated.Category = *req.Category
	}
	if req.Amount != nil {
		updated.Amount = *req.Amount
	}
	if req.PaymentMethod != nil {
		updated.PaymentMethod = *req.PaymentMethod
	}
	if req.Source != nil {
		updated.Source = sourceOrDefault(*req.Source)
	}
	if req.RawMessage != nil {
		updated.RawMessage = *req.RawMessage
	}

	if err := updated.validate(); err != nil {
		return Expense{}, err
	}
	return updated, nil
}

func (e Expense) Row(location *time.Location) []interface{} {
	timestamp := e.Timestamp.In(location)
	return []interface{}{
		e.ID,
		timestamp.Format(time.RFC3339),
		timestamp.Format("2006-01-02"),
		timestamp.Format("15:04"),
		e.Description,
		e.Category,
		e.Amount,
		e.PaymentMethod,
		e.Source,
		e.RawMessage,
	}
}

func ExpenseFromRow(row []interface{}, location *time.Location) (Expense, error) {
	if len(row) >= 6 && isRFC3339(cellString(cellAt(row, 0))) {
		return legacyExpenseFromRow(row, location)
	}

	if len(row) < 7 {
		return Expense{}, fmt.Errorf("expense row has %d columns, want at least 7", len(row))
	}

	timestamp, err := time.Parse(time.RFC3339, cellString(cellAt(row, 1)))
	if err != nil {
		return Expense{}, fmt.Errorf("parse timestamp: %w", err)
	}
	amount, err := cellInt64(cellAt(row, 6))
	if err != nil {
		return Expense{}, fmt.Errorf("parse amount: %w", err)
	}

	exp := Expense{
		ID:            cellString(cellAt(row, 0)),
		Timestamp:     timestamp.In(location),
		Description:   cellString(cellAt(row, 4)),
		Category:      cellString(cellAt(row, 5)),
		Amount:        amount,
		PaymentMethod: cellString(cellAt(row, 7)),
		Source:        sourceOrDefault(cellString(cellAt(row, 8))),
		RawMessage:    cellString(cellAt(row, 9)),
	}
	if err := exp.validate(); err != nil {
		return Expense{}, err
	}
	return exp, nil
}

func legacyExpenseFromRow(row []interface{}, location *time.Location) (Expense, error) {
	timestamp, err := time.Parse(time.RFC3339, cellString(cellAt(row, 0)))
	if err != nil {
		return Expense{}, fmt.Errorf("parse timestamp: %w", err)
	}
	amount, err := cellInt64(cellAt(row, 5))
	if err != nil {
		return Expense{}, fmt.Errorf("parse amount: %w", err)
	}

	exp := Expense{
		ID:            legacyID(row),
		Timestamp:     timestamp.In(location),
		Description:   cellString(cellAt(row, 3)),
		Category:      cellString(cellAt(row, 4)),
		Amount:        amount,
		PaymentMethod: cellString(cellAt(row, 6)),
		Source:        sourceOrDefault(cellString(cellAt(row, 7))),
		RawMessage:    cellString(cellAt(row, 8)),
	}
	if err := exp.validate(); err != nil {
		return Expense{}, err
	}
	return exp, nil
}

func (e Expense) validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return errors.New("id is required")
	}
	if strings.TrimSpace(e.Description) == "" {
		return errors.New("description is required")
	}
	if e.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if e.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	return nil
}

func sourceOrDefault(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return defaultSource
	}
	return trimmed
}

func cellString(value interface{}) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func cellAt(row []interface{}, index int) interface{} {
	if index >= len(row) {
		return ""
	}
	return row[index]
}

func cellInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		amount := cellString(value)
		parsed, err := strconv.ParseInt(amount, 10, 64)
		if err == nil {
			return parsed, nil
		}
		amount = strings.NewReplacer(",", "", ".", "").Replace(amount)
		return strconv.ParseInt(amount, 10, 64)
	}
}

func isRFC3339(value string) bool {
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}

func legacyID(row []interface{}) string {
	parts := make([]string, 0, len(row))
	for _, value := range row {
		parts = append(parts, cellString(value))
	}
	sum := sha1.Sum([]byte(strings.Join(parts, "\x00")))
	return "exp_legacy_" + hex.EncodeToString(sum[:])[:12]
}
