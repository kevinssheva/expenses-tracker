package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kevinssheva/expenses-tracker/internal/expense"
	"github.com/kevinssheva/expenses-tracker/internal/service"
)

type ExpenseService interface {
	Create(ctx context.Context, req expense.CreateRequest) (expense.Expense, error)
	List(ctx context.Context, filter service.Filter) ([]expense.Expense, error)
	Update(ctx context.Context, id string, req expense.UpdateRequest) (expense.Expense, error)
	Delete(ctx context.Context, id string) error
}

type expensesHandler struct {
	apiKey   string
	service  ExpenseService
	location *time.Location
}

type expenseResponse struct {
	ID            string `json:"id"`
	Date          string `json:"date"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	AccountWallet string `json:"account_wallet"`
}

func NewExpensesHandler(apiKey string, service ExpenseService, location *time.Location) expensesHandler {
	return expensesHandler{apiKey: apiKey, service: service, location: location}
}

func RegisterExpensesRoutes(mux *http.ServeMux, apiKey string, service ExpenseService, location *time.Location) {
	h := NewExpensesHandler(apiKey, service, location)

	mux.HandleFunc("GET /expenses", h.requireAPIKey(h.listExpenses))
	mux.HandleFunc("POST /expenses", h.requireAPIKey(h.createExpense))
	mux.HandleFunc("PATCH /expenses/{id}", h.requireAPIKey(h.updateExpense))
	mux.HandleFunc("DELETE /expenses/{id}", h.requireAPIKey(h.deleteExpense))
}

func (h expensesHandler) requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != h.apiKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (h expensesHandler) createExpense(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req expense.CreateRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	exp, err := h.service.Create(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, "create expense", err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]expenseResponse{"expense": toExpenseResponse(exp)})
}

func (h expensesHandler) listExpenses(w http.ResponseWriter, r *http.Request) {
	filter, err := parseFilter(r, h.location)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	expenses, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, "list expenses", err)
		return
	}

	response := make([]expenseResponse, 0, len(expenses))
	for _, exp := range expenses {
		response = append(response, toExpenseResponse(exp))
	}
	writeJSON(w, http.StatusOK, map[string][]expenseResponse{"expenses": response})
}

func (h expensesHandler) updateExpense(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	id := r.PathValue("id")
	var req expense.UpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	exp, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		h.handleServiceError(w, "update expense", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]expenseResponse{"expense": toExpenseResponse(exp)})
}

func (h expensesHandler) deleteExpense(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.service.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, "delete expense", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h expensesHandler) handleServiceError(w http.ResponseWriter, action string, err error) {
	if errors.Is(err, service.ErrInvalid) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, service.ErrNotFound) {
		http.Error(w, "expense not found", http.StatusNotFound)
		return
	}

	log.Printf("%s: %v", action, err)
	http.Error(w, "failed to handle expense", http.StatusInternalServerError)
}

func decodeJSON(r *http.Request, value interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}

func parseFilter(r *http.Request, location *time.Location) (service.Filter, error) {
	query := r.URL.Query()
	for key := range query {
		if key != "id" && key != "from" && key != "to" {
			return service.Filter{}, fmt.Errorf("unsupported query parameter %q", key)
		}
	}
	filter := service.Filter{ID: query.Get("id")}

	if query.Get("from") != "" {
		from, err := time.ParseInLocation("2006-01-02", query.Get("from"), location)
		if err != nil {
			return service.Filter{}, fmt.Errorf("from must use YYYY-MM-DD")
		}
		filter.From = &from
	}
	if query.Get("to") != "" {
		to, err := time.ParseInLocation("2006-01-02", query.Get("to"), location)
		if err != nil {
			return service.Filter{}, fmt.Errorf("to must use YYYY-MM-DD")
		}
		filter.To = &to
	}

	return filter, nil
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func toExpenseResponse(exp expense.Expense) expenseResponse {
	return expenseResponse{
		ID:            exp.ID,
		Date:          exp.Date.Format("2006-01-02"),
		Description:   exp.Description,
		Category:      exp.Category,
		Amount:        exp.Amount,
		PaymentMethod: exp.PaymentMethod,
		AccountWallet: exp.AccountWallet,
	}
}
