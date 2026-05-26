package expense

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrNotFound = errors.New("expense not found")

var allowedCategories = map[string]struct{}{
	"Food":          {},
	"Subscription":  {},
	"Transport":     {},
	"Shopping":      {},
	"Bills":         {},
	"Health":        {},
	"Entertainment": {},
	"Education":     {},
	"Other":         {},
}

type CreateRequest struct {
	Date          string `json:"date"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	AccountWallet string `json:"account_wallet"`
}

type UpdateRequest struct {
	Date          *string `json:"date"`
	Description   *string `json:"description"`
	Category      *string `json:"category"`
	Amount        *int64  `json:"amount"`
	PaymentMethod *string `json:"payment_method"`
	AccountWallet *string `json:"account_wallet"`
}

type Expense struct {
	ID            string
	Date          time.Time
	Description   string
	Category      string
	Amount        int64
	PaymentMethod string
	AccountWallet string
}

func New(id string, req CreateRequest, location *time.Location) (Expense, error) {
	date, err := parseDate(req.Date, location)
	if err != nil {
		return Expense{}, err
	}

	exp := Expense{
		ID:            strings.TrimSpace(id),
		Date:          date,
		Description:   strings.TrimSpace(req.Description),
		Category:      strings.TrimSpace(req.Category),
		Amount:        req.Amount,
		PaymentMethod: req.PaymentMethod,
		AccountWallet: req.AccountWallet,
	}
	if err := exp.validate(); err != nil {
		return Expense{}, err
	}
	return exp, nil
}

func (e Expense) ApplyUpdate(req UpdateRequest, location *time.Location) (Expense, error) {
	updated := e

	if req.Date != nil {
		date, err := parseDate(*req.Date, location)
		if err != nil {
			return Expense{}, err
		}
		updated.Date = date
	}
	if req.Description != nil {
		updated.Description = strings.TrimSpace(*req.Description)
	}
	if req.Category != nil {
		updated.Category = strings.TrimSpace(*req.Category)
	}
	if req.Amount != nil {
		updated.Amount = *req.Amount
	}
	if req.PaymentMethod != nil {
		updated.PaymentMethod = *req.PaymentMethod
	}
	if req.AccountWallet != nil {
		updated.AccountWallet = *req.AccountWallet
	}

	if err := updated.validate(); err != nil {
		return Expense{}, err
	}
	return updated, nil
}

func (e Expense) Row(location *time.Location) []interface{} {
	return []interface{}{
		e.Date.In(location).Format("2006-01-02"),
		e.Description,
		e.Category,
		e.Amount,
		e.PaymentMethod,
		e.AccountWallet,
		e.ID,
	}
}

func ExpenseFromRow(row []interface{}, location *time.Location) (Expense, error) {
	if len(row) < 7 {
		return Expense{}, fmt.Errorf("expense row has %d columns, want 7", len(row))
	}

	date, err := parseDate(cellString(cellAt(row, 0)), location)
	if err != nil {
		return Expense{}, err
	}
	amount, err := cellInt64(cellAt(row, 3))
	if err != nil {
		return Expense{}, fmt.Errorf("parse amount: %w", err)
	}

	exp := Expense{
		Date:          date,
		Description:   cellString(cellAt(row, 1)),
		Category:      cellString(cellAt(row, 2)),
		Amount:        amount,
		PaymentMethod: cellString(cellAt(row, 4)),
		AccountWallet: cellString(cellAt(row, 5)),
		ID:            cellString(cellAt(row, 6)),
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
	if e.Date.IsZero() {
		return errors.New("date is required")
	}
	if strings.TrimSpace(e.Description) == "" {
		return errors.New("description is required")
	}
	if _, ok := allowedCategories[e.Category]; !ok {
		return fmt.Errorf("category must be one of Food, Subscription, Transport, Shopping, Bills, Health, Entertainment, Education, Other")
	}
	if e.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	return nil
}

func parseDate(value string, location *time.Location) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, errors.New("date is required")
	}
	date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), location)
	if err != nil {
		return time.Time{}, fmt.Errorf("date must use YYYY-MM-DD: %w", err)
	}
	return date, nil
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
