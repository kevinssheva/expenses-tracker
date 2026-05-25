package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kevinssheva/expenses-tracker/internal/expense"
)

var ErrNotFound = errors.New("expense not found")
var ErrInvalid = errors.New("invalid expense")

type Store interface {
	ListExpenses(ctx context.Context) ([]expense.Expense, error)
	SaveExpenses(ctx context.Context, expenses []expense.Expense) error
}

type Filter struct {
	ID   string
	From *time.Time
	To   *time.Time
}

type ExpenseService struct {
	store    Store
	location *time.Location
	now      func() time.Time
	newID    func() string
	mu       *sync.Mutex
}

func NewExpenseService(store Store, location *time.Location) ExpenseService {
	return NewExpenseServiceWithClockAndID(store, location, time.Now, func() string {
		return "exp_" + uuid.NewString()
	})
}

func NewExpenseServiceWithClockAndID(store Store, location *time.Location, now func() time.Time, newID func() string) ExpenseService {
	return ExpenseService{store: store, location: location, now: now, newID: newID, mu: &sync.Mutex{}}
}

func (s ExpenseService) Create(ctx context.Context, req expense.CreateRequest) (expense.Expense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expenses, err := s.store.ListExpenses(ctx)
	if err != nil {
		return expense.Expense{}, err
	}

	created, err := expense.New(s.newID(), req, s.now().In(s.location))
	if err != nil {
		return expense.Expense{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	expenses = append(expenses, created)
	sortExpenses(expenses)
	if err := s.store.SaveExpenses(ctx, expenses); err != nil {
		return expense.Expense{}, err
	}
	return created, nil
}

func (s ExpenseService) List(ctx context.Context, filter Filter) ([]expense.Expense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expenses, err := s.store.ListExpenses(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]expense.Expense, 0, len(expenses))
	for _, exp := range expenses {
		if filter.ID != "" && exp.ID != filter.ID {
			continue
		}
		if filter.From != nil && exp.Timestamp.Before(*filter.From) {
			continue
		}
		if filter.To != nil && exp.Timestamp.After(*filter.To) {
			continue
		}
		filtered = append(filtered, exp)
	}
	sortExpenses(filtered)
	return filtered, nil
}

func (s ExpenseService) Update(ctx context.Context, id string, req expense.UpdateRequest) (expense.Expense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expenses, err := s.store.ListExpenses(ctx)
	if err != nil {
		return expense.Expense{}, err
	}

	for i, exp := range expenses {
		if exp.ID != id {
			continue
		}
		updated, err := exp.ApplyUpdate(req)
		if err != nil {
			return expense.Expense{}, fmt.Errorf("%w: %v", ErrInvalid, err)
		}
		expenses[i] = updated
		sortExpenses(expenses)
		if err := s.store.SaveExpenses(ctx, expenses); err != nil {
			return expense.Expense{}, err
		}
		return updated, nil
	}

	return expense.Expense{}, ErrNotFound
}

func (s ExpenseService) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	expenses, err := s.store.ListExpenses(ctx)
	if err != nil {
		return err
	}

	for i, exp := range expenses {
		if exp.ID != id {
			continue
		}
		expenses = append(expenses[:i], expenses[i+1:]...)
		sortExpenses(expenses)
		return s.store.SaveExpenses(ctx, expenses)
	}

	return ErrNotFound
}

func sortExpenses(expenses []expense.Expense) {
	sort.SliceStable(expenses, func(i, j int) bool {
		return expenses[i].Timestamp.Before(expenses[j].Timestamp)
	})
}
