package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

type fakeWriter struct {
	row []interface{}
	err error
}

func (f *fakeWriter) AppendExpense(_ context.Context, row []interface{}) error {
	f.row = row
	return f.err
}

func TestExpensesHandlerRequiresAPIKey(t *testing.T) {
	handler := NewExpensesHandler("secret", time.UTC, &fakeWriter{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader([]byte(`{}`)))

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestExpensesHandlerRejectsWrongAPIKey(t *testing.T) {
	handler := NewExpensesHandler("secret", time.UTC, &fakeWriter{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader([]byte(`{}`)))
	request.Header.Set("X-API-Key", "wrong")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestExpensesHandlerRejectsWrongMethod(t *testing.T) {
	handler := NewExpensesHandler("secret", time.UTC, &fakeWriter{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestExpensesHandlerRejectsInvalidJSON(t *testing.T) {
	handler := NewExpensesHandler("secret", time.UTC, &fakeWriter{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader([]byte(`{`)))
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestExpensesHandlerRejectsInvalidExpense(t *testing.T) {
	body := map[string]interface{}{"description": "kopi susu", "amount": 0}
	requestBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	handler := NewExpensesHandler("secret", time.UTC, &fakeWriter{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader(requestBody))
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestExpensesHandlerRecordsExpense(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	writer := &fakeWriter{}
	handler := NewExpensesHandlerWithClock("secret", location, writer, func() time.Time {
		return time.Date(2026, 5, 23, 14, 20, 0, 0, location)
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader([]byte(`{
		"description":"kopi susu",
		"category":"food",
		"amount":28000,
		"payment_method":"QRIS",
		"source":"",
		"raw_message":"kopi susu 28k qris"
	}`)))
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", recorder.Header().Get("Content-Type"))
	}
	want := []interface{}{"2026-05-23T14:20:00+07:00", "2026-05-23", "14:20", "kopi susu", "food", int64(28000), "QRIS", "nanoclaw", "kopi susu 28k qris"}
	if !reflect.DeepEqual(writer.row, want) {
		t.Fatalf("row = %#v, want %#v", writer.row, want)
	}
}

func TestExpensesHandlerReturnsServerErrorWhenWriterFails(t *testing.T) {
	writer := &fakeWriter{err: errors.New("append failed")}
	handler := NewExpensesHandler("secret", time.UTC, writer)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader([]byte(`{"description":"kopi susu","amount":28000}`)))
	request.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
