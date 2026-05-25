package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kevinssheva/expenses-tracker/internal/expense"
	"github.com/kevinssheva/expenses-tracker/internal/service"
)

type fakeExpenseService struct {
	createReq  expense.CreateRequest
	listFilter service.Filter
	updateID   string
	updateReq  expense.UpdateRequest
	deleteID   string
	expenses   []expense.Expense
	err        error
}

func (f *fakeExpenseService) Create(_ context.Context, req expense.CreateRequest) (expense.Expense, error) {
	f.createReq = req
	if f.err != nil {
		return expense.Expense{}, f.err
	}
	return f.expenses[0], nil
}

func (f *fakeExpenseService) List(_ context.Context, filter service.Filter) ([]expense.Expense, error) {
	f.listFilter = filter
	if f.err != nil {
		return nil, f.err
	}
	return f.expenses, nil
}

func (f *fakeExpenseService) Update(_ context.Context, id string, req expense.UpdateRequest) (expense.Expense, error) {
	f.updateID = id
	f.updateReq = req
	if f.err != nil {
		return expense.Expense{}, f.err
	}
	return f.expenses[0], nil
}

func (f *fakeExpenseService) Delete(_ context.Context, id string) error {
	f.deleteID = id
	return f.err
}

func TestExpensesHandlerRequiresAPIKey(t *testing.T) {
	handler := NewExpensesHandler("secret", &fakeExpenseService{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses", nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestExpensesHandlerCreatesExpense(t *testing.T) {
	svc := &fakeExpenseService{expenses: []expense.Expense{sampleExpense()}}
	handler := NewExpensesHandler("secret", svc)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader([]byte(`{
		"timestamp":"2026-05-25T10:00:00+07:00",
		"description":"kopi susu",
		"category":"food",
		"amount":28000,
		"payment_method":"QRIS",
		"raw_message":"kopi susu 28k qris"
	}`)))
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if svc.createReq.Timestamp != "2026-05-25T10:00:00+07:00" {
		t.Fatalf("Timestamp = %q, want provided timestamp", svc.createReq.Timestamp)
	}
	var response map[string]expenseResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if response["expense"].ID != "exp_1" {
		t.Fatalf("response id = %q, want exp_1", response["expense"].ID)
	}
}

func TestExpensesHandlerListsExpensesWithFilters(t *testing.T) {
	svc := &fakeExpenseService{expenses: []expense.Expense{sampleExpense()}}
	handler := NewExpensesHandler("secret", svc)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses?id=exp_1&from=2026-05-25T00:00:00Z&to=2026-05-25T23:59:59Z", nil)
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if svc.listFilter.ID != "exp_1" {
		t.Fatalf("filter ID = %q, want exp_1", svc.listFilter.ID)
	}
	if svc.listFilter.From == nil || svc.listFilter.To == nil {
		t.Fatalf("filter from/to must be parsed")
	}
}

func TestExpensesHandlerRejectsInvalidFilterTimestamp(t *testing.T) {
	handler := NewExpensesHandler("secret", &fakeExpenseService{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses?from=not-a-time", nil)
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestExpensesHandlerRejectsUnknownQueryParam(t *testing.T) {
	handler := NewExpensesHandler("secret", &fakeExpenseService{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses?summary=true", nil)
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestExpensesHandlerMapsInvalidExpenseTo400(t *testing.T) {
	svc := &fakeExpenseService{err: service.ErrInvalid}
	handler := NewExpensesHandler("secret", svc)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader([]byte(`{"description":"","amount":0}`)))
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestExpensesHandlerUpdatesExpense(t *testing.T) {
	svc := &fakeExpenseService{expenses: []expense.Expense{sampleExpense()}}
	handler := NewExpensesHandler("secret", svc)
	description := "kopi hitam"
	body, err := json.Marshal(map[string]string{"description": description})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/expenses/exp_1", bytes.NewReader(body))
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if svc.updateID != "exp_1" {
		t.Fatalf("update ID = %q, want exp_1", svc.updateID)
	}
	if svc.updateReq.Description == nil || *svc.updateReq.Description != "kopi hitam" {
		t.Fatalf("description update was not decoded")
	}
}

func TestExpensesHandlerDeletesExpense(t *testing.T) {
	svc := &fakeExpenseService{}
	handler := NewExpensesHandler("secret", svc)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/expenses/exp_1", nil)
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if svc.deleteID != "exp_1" {
		t.Fatalf("delete ID = %q, want exp_1", svc.deleteID)
	}
}

func TestExpensesHandlerMapsNotFoundTo404(t *testing.T) {
	svc := &fakeExpenseService{err: service.ErrNotFound}
	handler := NewExpensesHandler("secret", svc)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/expenses/missing", nil)
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestExpensesHandlerMapsServiceFailureTo500(t *testing.T) {
	svc := &fakeExpenseService{err: errors.New("store failed")}
	handler := NewExpensesHandler("secret", svc)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func sampleExpense() expense.Expense {
	return expense.Expense{
		ID:            "exp_1",
		Timestamp:     time.Date(2026, 5, 25, 10, 0, 0, 0, time.FixedZone("Asia/Jakarta", 7*60*60)),
		Description:   "kopi susu",
		Category:      "food",
		Amount:        28000,
		PaymentMethod: "QRIS",
		Source:        "nanoclaw",
		RawMessage:    "kopi susu 28k qris",
	}
}
