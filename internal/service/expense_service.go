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

var ErrNotFound = expense.ErrNotFound
var ErrInvalid = errors.New("invalid expense")

type Store interface {
	ListExpenses(ctx context.Context) ([]expense.Expense, error)
	AppendExpense(ctx context.Context, exp expense.Expense) error
	UpdateExpense(ctx context.Context, exp expense.Expense) error
	DeleteExpense(ctx context.Context, id string) error
	SortExpenses(ctx context.Context) error
}

type Filter struct {
	ID   string
	From *time.Time
	To   *time.Time
}

type ExpenseService struct {
	store    Store
	location *time.Location
	newID    func() string
	mu       *sync.Mutex
}

func NewExpenseService(store Store, location *time.Location) ExpenseService {
	return newExpenseServiceWithID(store, location, func() string {
		return "exp_" + uuid.NewString()
	})
}

func newExpenseServiceWithID(store Store, location *time.Location, newID func() string) ExpenseService {
	return ExpenseService{store: store, location: location, newID: newID, mu: &sync.Mutex{}}
}

func (s ExpenseService) Create(ctx context.Context, req expense.CreateRequest) (expense.Expense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	created, err := expense.New(s.newID(), req, s.location)
	if err != nil {
		return expense.Expense{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if err := s.store.AppendExpense(ctx, created); err != nil {
		return expense.Expense{}, err
	}
	if err := s.store.SortExpenses(ctx); err != nil {
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
		if filter.From != nil && exp.Date.Before(*filter.From) {
			continue
		}
		if filter.To != nil && exp.Date.After(*filter.To) {
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

	for _, exp := range expenses {
		if exp.ID != id {
			continue
		}
		updated, err := exp.ApplyUpdate(req, s.location)
		if err != nil {
			return expense.Expense{}, fmt.Errorf("%w: %v", ErrInvalid, err)
		}
		if err := s.store.UpdateExpense(ctx, updated); err != nil {
			return expense.Expense{}, err
		}
		if err := s.store.SortExpenses(ctx); err != nil {
			return expense.Expense{}, err
		}
		return updated, nil
	}

	return expense.Expense{}, ErrNotFound
}

func (s ExpenseService) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.store.DeleteExpense(ctx, id)
}

func sortExpenses(expenses []expense.Expense) {
	sort.SliceStable(expenses, func(i, j int) bool {
		return expenses[i].Date.Before(expenses[j].Date)
	})
}
