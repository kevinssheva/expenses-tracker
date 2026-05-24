package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/kevinssheva/expenses-tracker/internal/expense"
)

type ExpenseWriter interface {
	AppendExpense(ctx context.Context, row []interface{}) error
}

type clock func() time.Time

type expensesHandler struct {
	apiKey   string
	location *time.Location
	writer   ExpenseWriter
	now      clock
}

func NewExpensesHandler(apiKey string, location *time.Location, writer ExpenseWriter) http.Handler {
	return NewExpensesHandlerWithClock(apiKey, location, writer, time.Now)
}

func NewExpensesHandlerWithClock(apiKey string, location *time.Location, writer ExpenseWriter, now clock) http.Handler {
	return expensesHandler{
		apiKey:   apiKey,
		location: location,
		writer:   writer,
		now:      now,
	}
}

func (h expensesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-API-Key") != h.apiKey {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	var req expense.Request
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	exp, err := expense.New(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.writer.AppendExpense(r.Context(), exp.Row(h.now().In(h.location))); err != nil {
		log.Printf("append expense: %v", err)
		http.Error(w, "failed to record expense", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}
