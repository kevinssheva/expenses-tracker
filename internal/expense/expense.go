package expense

import (
	"errors"
	"strings"
	"time"
)

const defaultSource = "nanoclaw"

type Request struct {
	Description   string `json:"description"`
	Category      string `json:"category"`
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Source        string `json:"source"`
	RawMessage    string `json:"raw_message"`
}

type Expense struct {
	Description   string
	Category      string
	Amount        int64
	PaymentMethod string
	Source        string
	RawMessage    string
}

func New(req Request) (Expense, error) {
	description := strings.TrimSpace(req.Description)
	if description == "" {
		return Expense{}, errors.New("description is required")
	}
	if req.Amount <= 0 {
		return Expense{}, errors.New("amount must be greater than 0")
	}

	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = defaultSource
	}

	return Expense{
		Description:   description,
		Category:      req.Category,
		Amount:        req.Amount,
		PaymentMethod: req.PaymentMethod,
		Source:        source,
		RawMessage:    req.RawMessage,
	}, nil
}

func (e Expense) Row(now time.Time) []interface{} {
	return []interface{}{
		now.Format(time.RFC3339),
		now.Format("2006-01-02"),
		now.Format("15:04"),
		e.Description,
		e.Category,
		e.Amount,
		e.PaymentMethod,
		e.Source,
		e.RawMessage,
	}
}
