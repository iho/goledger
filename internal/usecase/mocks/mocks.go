package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
	"github.com/shopspring/decimal"
)

// MockAccountRepository is a mock implementation of AccountRepository.
type MockAccountRepository struct {
	mu       sync.RWMutex
	accounts map[string]*domain.Account

	CreateFunc            func(ctx context.Context, account *domain.Account) error
	GetByIDFunc           func(ctx context.Context, id string) (*domain.Account, error)
	GetByIDForUpdateFunc  func(ctx context.Context, tx usecase.Transaction, id string) (*domain.Account, error)
	GetByIDsForUpdateFunc func(ctx context.Context, tx usecase.Transaction, ids []string) ([]*domain.Account, error)
	UpdateBalanceFunc     func(ctx context.Context, tx usecase.Transaction, id string, balance decimal.Decimal, updatedAt time.Time) error
	ListFunc              func(ctx context.Context, limit, offset int) ([]*domain.Account, error)
}

func NewMockAccountRepository() *MockAccountRepository {
	return &MockAccountRepository{
		accounts: make(map[string]*domain.Account),
	}
}

func (m *MockAccountRepository) Create(ctx context.Context, account *domain.Account) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, account)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.accounts[account.ID] = account
	return nil
}

func (m *MockAccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if acc, ok := m.accounts[id]; ok {
		return acc, nil
	}
	return nil, domain.ErrAccountNotFound
}

func (m *MockAccountRepository) GetByIDForUpdate(ctx context.Context, tx usecase.Transaction, id string) (*domain.Account, error) {
	if m.GetByIDForUpdateFunc != nil {
		return m.GetByIDForUpdateFunc(ctx, tx, id)
	}
	return m.GetByID(ctx, id)
}

func (m *MockAccountRepository) GetByIDsForUpdate(ctx context.Context, tx usecase.Transaction, ids []string) ([]*domain.Account, error) {
	if m.GetByIDsForUpdateFunc != nil {
		return m.GetByIDsForUpdateFunc(ctx, tx, ids)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var accounts []*domain.Account
	for _, id := range ids {
		if acc, ok := m.accounts[id]; ok {
			accounts = append(accounts, acc)
		}
	}
	return accounts, nil
}

func (m *MockAccountRepository) UpdateBalance(ctx context.Context, tx usecase.Transaction, id string, balance decimal.Decimal, updatedAt time.Time) error {
	if m.UpdateBalanceFunc != nil {
		return m.UpdateBalanceFunc(ctx, tx, id, balance, updatedAt)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if acc, ok := m.accounts[id]; ok {
		acc.Balance = balance
		acc.Version++
		acc.UpdatedAt = updatedAt
	}
	return nil
}

func (m *MockAccountRepository) List(ctx context.Context, limit, offset int) ([]*domain.Account, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, limit, offset)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var accounts []*domain.Account
	for _, acc := range m.accounts {
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

// MockTransferRepository is a mock implementation of TransferRepository.
type MockTransferRepository struct {
	mu        sync.RWMutex
	transfers map[string]*domain.Transfer

	CreateFunc        func(ctx context.Context, tx usecase.Transaction, transfer *domain.Transfer) error
	GetByIDFunc       func(ctx context.Context, id string) (*domain.Transfer, error)
	ListByAccountFunc func(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transfer, error)
}

func NewMockTransferRepository() *MockTransferRepository {
	return &MockTransferRepository{
		transfers: make(map[string]*domain.Transfer),
	}
}

func (m *MockTransferRepository) Create(ctx context.Context, tx usecase.Transaction, transfer *domain.Transfer) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, tx, transfer)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transfers[transfer.ID] = transfer
	return nil
}

func (m *MockTransferRepository) GetByID(ctx context.Context, id string) (*domain.Transfer, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if t, ok := m.transfers[id]; ok {
		return t, nil
	}
	return nil, domain.ErrTransferNotFound
}

func (m *MockTransferRepository) ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transfer, error) {
	if m.ListByAccountFunc != nil {
		return m.ListByAccountFunc(ctx, accountID, limit, offset)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var transfers []*domain.Transfer
	for _, t := range m.transfers {
		if t.FromAccountID == accountID || t.ToAccountID == accountID {
			transfers = append(transfers, t)
		}
	}
	return transfers, nil
}

// MockEntryRepository is a mock implementation of EntryRepository.
type MockEntryRepository struct {
	mu      sync.RWMutex
	entries map[string]*domain.Entry

	CreateFunc           func(ctx context.Context, tx usecase.Transaction, entry *domain.Entry) error
	GetByTransferFunc    func(ctx context.Context, transferID string) ([]*domain.Entry, error)
	GetByAccountFunc     func(ctx context.Context, accountID string, limit, offset int) ([]*domain.Entry, error)
	GetBalanceAtTimeFunc func(ctx context.Context, accountID string, at time.Time) (decimal.Decimal, error)
}

func NewMockEntryRepository() *MockEntryRepository {
	return &MockEntryRepository{
		entries: make(map[string]*domain.Entry),
	}
}

func (m *MockEntryRepository) Create(ctx context.Context, tx usecase.Transaction, entry *domain.Entry) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, tx, entry)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[entry.ID] = entry
	return nil
}

func (m *MockEntryRepository) GetByTransfer(ctx context.Context, transferID string) ([]*domain.Entry, error) {
	if m.GetByTransferFunc != nil {
		return m.GetByTransferFunc(ctx, transferID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var entries []*domain.Entry
	for _, e := range m.entries {
		if e.TransferID == transferID {
			entries = append(entries, e)
		}
	}
	return entries, nil
}

func (m *MockEntryRepository) GetByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.Entry, error) {
	if m.GetByAccountFunc != nil {
		return m.GetByAccountFunc(ctx, accountID, limit, offset)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var entries []*domain.Entry
	for _, e := range m.entries {
		if e.AccountID == accountID {
			entries = append(entries, e)
		}
	}
	return entries, nil
}

func (m *MockEntryRepository) GetBalanceAtTime(ctx context.Context, accountID string, at time.Time) (decimal.Decimal, error) {
	if m.GetBalanceAtTimeFunc != nil {
		return m.GetBalanceAtTimeFunc(ctx, accountID, at)
	}
	return decimal.Zero, nil
}

// MockTransactionManager is a mock implementation of TransactionManager.
type MockTransactionManager struct {
	BeginFunc func(ctx context.Context) (usecase.Transaction, error)
}

func NewMockTransactionManager() *MockTransactionManager {
	return &MockTransactionManager{}
}

func (m *MockTransactionManager) Begin(ctx context.Context) (usecase.Transaction, error) {
	if m.BeginFunc != nil {
		return m.BeginFunc(ctx)
	}
	return &MockTransaction{}, nil
}

// MockTransaction is a mock implementation of Transaction.
type MockTransaction struct {
	CommitFunc   func(ctx context.Context) error
	RollbackFunc func(ctx context.Context) error
}

func (m *MockTransaction) Commit(ctx context.Context) error {
	if m.CommitFunc != nil {
		return m.CommitFunc(ctx)
	}
	return nil
}

func (m *MockTransaction) Rollback(ctx context.Context) error {
	if m.RollbackFunc != nil {
		return m.RollbackFunc(ctx)
	}
	return nil
}

// MockIDGenerator is a mock implementation of IDGenerator.
type MockIDGenerator struct {
	GenerateFunc func() string
	counter      int
	mu           sync.Mutex
}

func NewMockIDGenerator() *MockIDGenerator {
	return &MockIDGenerator{}
}

func (m *MockIDGenerator) Generate() string {
	if m.GenerateFunc != nil {
		return m.GenerateFunc()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "mock-id-" + string(rune('0'+m.counter))
}

// MockIdempotencyStore is a mock implementation of IdempotencyStore.
type MockIdempotencyStore struct {
	mu   sync.RWMutex
	data map[string][]byte

	CheckAndSetFunc func(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error)
	UpdateFunc      func(ctx context.Context, key string, response []byte, ttl time.Duration) error
}

func NewMockIdempotencyStore() *MockIdempotencyStore {
	return &MockIdempotencyStore{
		data: make(map[string][]byte),
	}
}

func (m *MockIdempotencyStore) CheckAndSet(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error) {
	if m.CheckAndSetFunc != nil {
		return m.CheckAndSetFunc(ctx, key, response, ttl)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.data[key]; ok {
		return true, existing, nil
	}
	if response != nil {
		m.data[key] = response
	} else {
		m.data[key] = []byte("processing")
	}
	return false, nil, nil
}

func (m *MockIdempotencyStore) Update(ctx context.Context, key string, response []byte, ttl time.Duration) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, key, response, ttl)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = response
	return nil
}
