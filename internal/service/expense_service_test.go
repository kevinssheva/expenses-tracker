package service

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/kevinssheva/expenses-tracker/internal/expense"
)

type fakeStore struct {
	expenses   []expense.Expense
	appended   []expense.Expense
	updated    []expense.Expense
	deletedIDs []string
	sortCount  int
	err        error
	deleteErr  error
	appendErr  error
	updateErr  error
	sortErr    error
}

func (f *fakeStore) ListExpenses(context.Context) ([]expense.Expense, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]expense.Expense(nil), f.expenses...), nil
}

func TestCreateAppendsExpenseAndSorts(t *testing.T) {
	store := &fakeStore{}
	svc := newExpenseServiceWithID(store, time.UTC, func() string { return "exp_new" })

	created, err := svc.Create(context.Background(), expense.CreateRequest{
		Date:          "2026-05-25",
		Description:   "Makan ayam geprek",
		Category:      "Food",
		Amount:        35000,
		PaymentMethod: "QRIS",
		AccountWallet: "GoPay",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if created.ID != "exp_new" {
		t.Fatalf("ID = %q, want exp_new", created.ID)
	}
	if got := ids(store.appended); !reflect.DeepEqual(got, []string{"exp_new"}) {
		t.Fatalf("appended ids = %#v, want exp_new", got)
	}
	if store.sortCount != 1 {
		t.Fatalf("sortCount = %d, want 1", store.sortCount)
	}
}

func TestCreateReturnsInvalidForValidationError(t *testing.T) {
	svc := newExpenseServiceWithID(&fakeStore{}, time.UTC, func() string { return "exp_new" })

	_, err := svc.Create(context.Background(), expense.CreateRequest{Date: "2026-05-25", Description: "coffee", Category: "Salary", Amount: 20000})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Create error = %v, want ErrInvalid", err)
	}
}

func TestCreateReturnsStoreErrorWithoutSorting(t *testing.T) {
	store := &fakeStore{appendErr: errors.New("append failed")}
	svc := newExpenseServiceWithID(store, time.UTC, func() string { return "exp_new" })

	_, err := svc.Create(context.Background(), expense.CreateRequest{Date: "2026-05-25", Description: "coffee", Category: "Food", Amount: 20000})
	if err == nil {
		t.Fatal("Create returned nil error, want append error")
	}
	if store.sortCount != 0 {
		t.Fatalf("sortCount = %d, want 0", store.sortCount)
	}
}

func TestCreateSerializesConcurrentWrites(t *testing.T) {
	store := &concurrentStore{}
	var idMu sync.Mutex
	ids := []string{"exp_1", "exp_2"}
	svc := newExpenseServiceWithID(store, time.UTC, func() string {
		idMu.Lock()
		defer idMu.Unlock()
		id := ids[0]
		ids = ids[1:]
		return id
	})

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Create(context.Background(), expense.CreateRequest{Date: "2026-05-25", Description: "coffee", Category: "Food", Amount: 20000})
			if err != nil {
				t.Errorf("Create returned error: %v", err)
			}
		}()
	}
	wg.Wait()

	if len(store.expenses) != 2 {
		t.Fatalf("len(expenses) = %d, want 2", len(store.expenses))
	}
}

func TestListFiltersByIDAndDateRange(t *testing.T) {
	expenses := []expense.Expense{
		{ID: "exp_1", Date: time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC), Description: "coffee", Category: "Food", Amount: 20000},
		{ID: "exp_2", Date: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), Description: "lunch", Category: "Food", Amount: 50000},
		{ID: "exp_3", Date: time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC), Description: "dinner", Category: "Food", Amount: 75000},
	}
	store := &fakeStore{expenses: expenses}
	svc := newExpenseServiceWithID(store, time.UTC, func() string { return "exp_new" })
	from := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)

	got, err := svc.List(context.Background(), Filter{From: &from, To: &to})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if gotIDs := ids(got); !reflect.DeepEqual(gotIDs, []string{"exp_2"}) {
		t.Fatalf("filtered ids = %#v, want exp_2", gotIDs)
	}

	got, err = svc.List(context.Background(), Filter{ID: "exp_3"})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if gotIDs := ids(got); !reflect.DeepEqual(gotIDs, []string{"exp_3"}) {
		t.Fatalf("filtered ids = %#v, want exp_3", gotIDs)
	}
}

func TestUpdateEditsMatchingExpenseAndSorts(t *testing.T) {
	store := &fakeStore{expenses: []expense.Expense{
		{ID: "exp_1", Date: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), Description: "coffee", Category: "Food", Amount: 20000},
		{ID: "exp_2", Date: time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC), Description: "lunch", Category: "Food", Amount: 50000},
	}}
	svc := newExpenseServiceWithID(store, time.UTC, func() string { return "exp_new" })
	description := "brunch"
	date := "2026-05-24"

	updated, err := svc.Update(context.Background(), "exp_2", expense.UpdateRequest{Date: &date, Description: &description})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if updated.Description != "brunch" {
		t.Fatalf("Description = %q, want brunch", updated.Description)
	}
	if got := ids(store.updated); !reflect.DeepEqual(got, []string{"exp_2"}) {
		t.Fatalf("updated ids = %#v, want exp_2", got)
	}
	if store.sortCount != 1 {
		t.Fatalf("sortCount = %d, want 1", store.sortCount)
	}
}

func TestUpdateReturnsNotFound(t *testing.T) {
	svc := newExpenseServiceWithID(&fakeStore{}, time.UTC, func() string { return "exp_new" })

	_, err := svc.Update(context.Background(), "missing", expense.UpdateRequest{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update error = %v, want ErrNotFound", err)
	}
}

func TestUpdateReturnsInvalidForValidationError(t *testing.T) {
	store := &fakeStore{expenses: []expense.Expense{{ID: "exp_1", Date: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), Description: "coffee", Category: "Food", Amount: 20000}}}
	svc := newExpenseServiceWithID(store, time.UTC, func() string { return "exp_new" })
	amount := int64(0)

	_, err := svc.Update(context.Background(), "exp_1", expense.UpdateRequest{Amount: &amount})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Update error = %v, want ErrInvalid", err)
	}
}

func TestDeleteRemovesMatchingExpense(t *testing.T) {
	store := &fakeStore{}
	svc := newExpenseServiceWithID(store, time.UTC, func() string { return "exp_new" })

	if err := svc.Delete(context.Background(), "exp_1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if !reflect.DeepEqual(store.deletedIDs, []string{"exp_1"}) {
		t.Fatalf("deleted IDs = %#v, want exp_1", store.deletedIDs)
	}
	if store.sortCount != 0 {
		t.Fatalf("sortCount = %d, want 0", store.sortCount)
	}
}

func TestDeleteReturnsNotFound(t *testing.T) {
	svc := newExpenseServiceWithID(&fakeStore{deleteErr: ErrNotFound}, time.UTC, func() string { return "exp_new" })

	err := svc.Delete(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete error = %v, want ErrNotFound", err)
	}
}

func ids(expenses []expense.Expense) []string {
	ids := make([]string, 0, len(expenses))
	for _, exp := range expenses {
		ids = append(ids, exp.ID)
	}
	return ids
}

type concurrentStore struct {
	mu       sync.Mutex
	expenses []expense.Expense
}

func (s *concurrentStore) ListExpenses(context.Context) ([]expense.Expense, error) {
	s.mu.Lock()
	expenses := append([]expense.Expense(nil), s.expenses...)
	s.mu.Unlock()
	time.Sleep(10 * time.Millisecond)
	return expenses, nil
}

func (f *fakeStore) AppendExpense(_ context.Context, exp expense.Expense) error {
	if f.appendErr != nil {
		return f.appendErr
	}
	f.appended = append(f.appended, exp)
	return nil
}

func (f *fakeStore) UpdateExpense(_ context.Context, exp expense.Expense) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = append(f.updated, exp)
	return nil
}

func (f *fakeStore) DeleteExpense(_ context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletedIDs = append(f.deletedIDs, id)
	return nil
}

func (f *fakeStore) SortExpenses(context.Context) error {
	if f.sortErr != nil {
		return f.sortErr
	}
	f.sortCount++
	return nil
}

func (s *concurrentStore) AppendExpense(_ context.Context, exp expense.Expense) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expenses = append(s.expenses, exp)
	return nil
}

func (s *concurrentStore) UpdateExpense(_ context.Context, exp expense.Expense) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.expenses {
		if s.expenses[i].ID == exp.ID {
			s.expenses[i] = exp
			return nil
		}
	}
	return ErrNotFound
}

func (s *concurrentStore) DeleteExpense(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.expenses {
		if s.expenses[i].ID == id {
			s.expenses = append(s.expenses[:i], s.expenses[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (s *concurrentStore) SortExpenses(context.Context) error { return nil }
