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

func newTestMux(apiKey string, svc ExpenseService, location *time.Location) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterExpensesRoutes(mux, apiKey, svc, location)
	return mux
}

func TestRegisterExpensesRoutesRoutesCollectionRequests(t *testing.T) {
	svc := &fakeExpenseService{expenses: []expense.Expense{sampleExpense()}}
	mux := http.NewServeMux()
	RegisterExpensesRoutes(mux, "secret", svc, time.UTC)

	createRecorder := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader([]byte(`{
		"date":"2026-05-25",
		"description":"Makan ayam geprek",
		"category":"Food",
		"amount":35000,
		"payment_method":"QRIS",
		"account_wallet":"GoPay"
	}`)))
	createRequest.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(createRecorder, createRequest)

	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d, body %s", createRecorder.Code, http.StatusCreated, createRecorder.Body.String())
	}
	if svc.createReq.Description != "Makan ayam geprek" {
		t.Fatalf("create description = %q, want Makan ayam geprek", svc.createReq.Description)
	}

	listRecorder := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	listRequest.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(listRecorder, listRequest)

	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body %s", listRecorder.Code, http.StatusOK, listRecorder.Body.String())
	}
}

func TestRegisterExpensesRoutesRoutesItemRequests(t *testing.T) {
	svc := &fakeExpenseService{expenses: []expense.Expense{sampleExpense()}}
	mux := http.NewServeMux()
	RegisterExpensesRoutes(mux, "secret", svc, time.UTC)

	description := "Spotify"
	body, err := json.Marshal(map[string]string{"description": description})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	updateRecorder := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPatch, "/expenses/exp_1", bytes.NewReader(body))
	updateRequest.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(updateRecorder, updateRequest)

	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d, body %s", updateRecorder.Code, http.StatusOK, updateRecorder.Body.String())
	}
	if svc.updateID != "exp_1" {
		t.Fatalf("update ID = %q, want exp_1", svc.updateID)
	}

	deleteRecorder := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/expenses/exp_1", nil)
	deleteRequest.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(deleteRecorder, deleteRequest)

	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d", deleteRecorder.Code, http.StatusOK)
	}
	if svc.deleteID != "exp_1" {
		t.Fatalf("delete ID = %q, want exp_1", svc.deleteID)
	}
}

func TestExpensesHandlerRequiresAPIKey(t *testing.T) {
	mux := newTestMux("secret", &fakeExpenseService{}, time.UTC)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses", nil)

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestExpensesHandlerCreatesExpense(t *testing.T) {
	svc := &fakeExpenseService{expenses: []expense.Expense{sampleExpense()}}
	mux := newTestMux("secret", svc, time.UTC)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader([]byte(`{
		"date":"2026-05-25",
		"description":"Makan ayam geprek",
		"category":"Food",
		"amount":35000,
		"payment_method":"QRIS",
		"account_wallet":"GoPay"
	}`)))
	request.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if svc.createReq.Date != "2026-05-25" {
		t.Fatalf("Date = %q, want provided date", svc.createReq.Date)
	}
	if svc.createReq.AccountWallet != "GoPay" {
		t.Fatalf("AccountWallet = %q, want GoPay", svc.createReq.AccountWallet)
	}
	var response map[string]expenseResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if response["expense"].ID != "exp_1" {
		t.Fatalf("response id = %q, want exp_1", response["expense"].ID)
	}
	if response["expense"].Date != "2026-05-25" {
		t.Fatalf("response date = %q, want 2026-05-25", response["expense"].Date)
	}
}

func TestExpensesHandlerListsExpensesWithFilters(t *testing.T) {
	svc := &fakeExpenseService{expenses: []expense.Expense{sampleExpense()}}
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	mux := newTestMux("secret", svc, location)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses?id=exp_1&from=2026-05-25&to=2026-05-25", nil)
	request.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if svc.listFilter.ID != "exp_1" {
		t.Fatalf("filter ID = %q, want exp_1", svc.listFilter.ID)
	}
	if svc.listFilter.From == nil || svc.listFilter.To == nil {
		t.Fatalf("filter from/to must be parsed")
	}
	if !svc.listFilter.From.Equal(time.Date(2026, 5, 25, 0, 0, 0, 0, location)) {
		t.Fatalf("from = %v, want Asia/Jakarta date midnight", svc.listFilter.From)
	}
	if !svc.listFilter.To.Equal(time.Date(2026, 5, 25, 0, 0, 0, 0, location)) {
		t.Fatalf("to = %v, want Asia/Jakarta date midnight", svc.listFilter.To)
	}
}

func TestExpensesHandlerRejectsInvalidFilterDate(t *testing.T) {
	mux := newTestMux("secret", &fakeExpenseService{}, time.UTC)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses?from=not-a-date", nil)
	request.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestExpensesHandlerRejectsUnknownQueryParam(t *testing.T) {
	mux := newTestMux("secret", &fakeExpenseService{}, time.UTC)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses?summary=true", nil)
	request.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestExpensesHandlerMapsInvalidExpenseTo400(t *testing.T) {
	svc := &fakeExpenseService{err: service.ErrInvalid}
	mux := newTestMux("secret", svc, time.UTC)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader([]byte(`{"date":"2026-05-25","description":"","category":"Salary","amount":0}`)))
	request.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestExpensesHandlerUpdatesExpense(t *testing.T) {
	svc := &fakeExpenseService{expenses: []expense.Expense{sampleExpense()}}
	mux := newTestMux("secret", svc, time.UTC)
	description := "Spotify"
	body, err := json.Marshal(map[string]string{"description": description})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/expenses/exp_1", bytes.NewReader(body))
	request.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if svc.updateID != "exp_1" {
		t.Fatalf("update ID = %q, want exp_1", svc.updateID)
	}
	if svc.updateReq.Description == nil || *svc.updateReq.Description != "Spotify" {
		t.Fatalf("description update was not decoded")
	}
}

func TestExpensesHandlerDeletesExpense(t *testing.T) {
	svc := &fakeExpenseService{}
	mux := newTestMux("secret", svc, time.UTC)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/expenses/exp_1", nil)
	request.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if svc.deleteID != "exp_1" {
		t.Fatalf("delete ID = %q, want exp_1", svc.deleteID)
	}
}

func TestExpensesHandlerMapsNotFoundTo404(t *testing.T) {
	svc := &fakeExpenseService{err: service.ErrNotFound}
	mux := newTestMux("secret", svc, time.UTC)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/expenses/missing", nil)
	request.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestExpensesHandlerMapsServiceFailureTo500(t *testing.T) {
	svc := &fakeExpenseService{err: errors.New("store failed")}
	mux := newTestMux("secret", svc, time.UTC)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	request.Header.Set("X-API-Key", "secret")

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func sampleExpense() expense.Expense {
	return expense.Expense{
		ID:            "exp_1",
		Date:          time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
		Description:   "Makan ayam geprek",
		Category:      "Food",
		Amount:        35000,
		PaymentMethod: "QRIS",
		AccountWallet: "GoPay",
	}
}
