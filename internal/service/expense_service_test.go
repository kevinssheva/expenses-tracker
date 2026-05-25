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
	expenses []expense.Expense
	saved    []expense.Expense
	err      error
}

func (f *fakeStore) ListExpenses(context.Context) ([]expense.Expense, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]expense.Expense(nil), f.expenses...), nil
}

func (f *fakeStore) SaveExpenses(_ context.Context, expenses []expense.Expense) error {
	if f.err != nil {
		return f.err
	}
	f.saved = append([]expense.Expense(nil), expenses...)
	return nil
}

func TestCreateUsesProvidedTimestampAndSavesSorted(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	existing := expense.Expense{ID: "exp_existing", Timestamp: time.Date(2026, 5, 25, 12, 0, 0, 0, location), Description: "lunch", Amount: 50000, Source: "nanoclaw"}
	store := &fakeStore{expenses: []expense.Expense{existing}}
	svc := NewExpenseServiceWithClockAndID(store, location, func() time.Time {
		return time.Date(2026, 5, 25, 9, 0, 0, 0, location)
	}, func() string { return "exp_new" })

	created, err := svc.Create(context.Background(), expense.CreateRequest{
		Timestamp:   "2026-05-25T08:00:00+07:00",
		Description: "breakfast",
		Amount:      25000,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if created.ID != "exp_new" {
		t.Fatalf("ID = %q, want exp_new", created.ID)
	}
	if !created.Timestamp.Equal(time.Date(2026, 5, 25, 8, 0, 0, 0, location)) {
		t.Fatalf("Timestamp = %v, want provided timestamp", created.Timestamp)
	}
	wantOrder := []string{"exp_new", "exp_existing"}
	if gotOrder := ids(store.saved); !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("saved order = %#v, want %#v", gotOrder, wantOrder)
	}
}

func TestCreateUsesCurrentTimeWhenTimestampMissing(t *testing.T) {
	now := time.Date(2026, 5, 25, 9, 0, 0, 0, time.UTC)
	store := &fakeStore{}
	svc := NewExpenseServiceWithClockAndID(store, time.UTC, func() time.Time { return now }, func() string { return "exp_new" })

	created, err := svc.Create(context.Background(), expense.CreateRequest{Description: "lunch", Amount: 50000})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if !created.Timestamp.Equal(now) {
		t.Fatalf("Timestamp = %v, want %v", created.Timestamp, now)
	}
}

func TestCreateReturnsInvalidForValidationError(t *testing.T) {
	svc := NewExpenseServiceWithClockAndID(&fakeStore{}, time.UTC, time.Now, func() string { return "exp_new" })

	_, err := svc.Create(context.Background(), expense.CreateRequest{Description: "", Amount: 50000})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Create error = %v, want ErrInvalid", err)
	}
}

func TestCreateSerializesConcurrentWrites(t *testing.T) {
	store := &concurrentStore{}
	var idMu sync.Mutex
	ids := []string{"exp_1", "exp_2"}
	svc := NewExpenseServiceWithClockAndID(store, time.UTC, func() time.Time {
		return time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	}, func() string {
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
			_, err := svc.Create(context.Background(), expense.CreateRequest{Description: "coffee", Amount: 20000})
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

func TestListFiltersByIDAndTimeRange(t *testing.T) {
	expenses := []expense.Expense{
		{ID: "exp_1", Timestamp: time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC), Description: "coffee", Amount: 20000, Source: "nanoclaw"},
		{ID: "exp_2", Timestamp: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC), Description: "lunch", Amount: 50000, Source: "nanoclaw"},
		{ID: "exp_3", Timestamp: time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC), Description: "dinner", Amount: 75000, Source: "nanoclaw"},
	}
	store := &fakeStore{expenses: expenses}
	svc := NewExpenseServiceWithClockAndID(store, time.UTC, time.Now, func() string { return "exp_new" })
	from := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 25, 23, 59, 59, 0, time.UTC)

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

func TestUpdateEditsMatchingExpenseAndSavesSorted(t *testing.T) {
	store := &fakeStore{expenses: []expense.Expense{
		{ID: "exp_1", Timestamp: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC), Description: "coffee", Amount: 20000, Source: "nanoclaw"},
		{ID: "exp_2", Timestamp: time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC), Description: "lunch", Amount: 50000, Source: "nanoclaw"},
	}}
	svc := NewExpenseServiceWithClockAndID(store, time.UTC, time.Now, func() string { return "exp_new" })
	description := "brunch"
	timestamp := "2026-05-25T09:00:00Z"

	updated, err := svc.Update(context.Background(), "exp_2", expense.UpdateRequest{Timestamp: &timestamp, Description: &description})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if updated.Description != "brunch" {
		t.Fatalf("Description = %q, want brunch", updated.Description)
	}
	if gotOrder := ids(store.saved); !reflect.DeepEqual(gotOrder, []string{"exp_2", "exp_1"}) {
		t.Fatalf("saved order = %#v, want exp_2 then exp_1", gotOrder)
	}
}

func TestUpdateReturnsNotFound(t *testing.T) {
	svc := NewExpenseServiceWithClockAndID(&fakeStore{}, time.UTC, time.Now, func() string { return "exp_new" })

	_, err := svc.Update(context.Background(), "missing", expense.UpdateRequest{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update error = %v, want ErrNotFound", err)
	}
}

func TestUpdateReturnsInvalidForValidationError(t *testing.T) {
	store := &fakeStore{expenses: []expense.Expense{{ID: "exp_1", Timestamp: time.Now(), Description: "coffee", Amount: 20000, Source: "nanoclaw"}}}
	svc := NewExpenseServiceWithClockAndID(store, time.UTC, time.Now, func() string { return "exp_new" })
	amount := int64(0)

	_, err := svc.Update(context.Background(), "exp_1", expense.UpdateRequest{Amount: &amount})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Update error = %v, want ErrInvalid", err)
	}
}

func TestDeleteRemovesMatchingExpense(t *testing.T) {
	store := &fakeStore{expenses: []expense.Expense{
		{ID: "exp_1", Timestamp: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC), Description: "coffee", Amount: 20000, Source: "nanoclaw"},
		{ID: "exp_2", Timestamp: time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC), Description: "lunch", Amount: 50000, Source: "nanoclaw"},
	}}
	svc := NewExpenseServiceWithClockAndID(store, time.UTC, time.Now, func() string { return "exp_new" })

	if err := svc.Delete(context.Background(), "exp_1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if gotIDs := ids(store.saved); !reflect.DeepEqual(gotIDs, []string{"exp_2"}) {
		t.Fatalf("saved ids = %#v, want exp_2", gotIDs)
	}
}

func TestDeleteReturnsNotFound(t *testing.T) {
	svc := NewExpenseServiceWithClockAndID(&fakeStore{}, time.UTC, time.Now, func() string { return "exp_new" })

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

func (s *concurrentStore) SaveExpenses(_ context.Context, expenses []expense.Expense) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expenses = append([]expense.Expense(nil), expenses...)
	return nil
}
