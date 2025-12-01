//	mockgen -source=internal/usecase/interfaces.go -destination=internal/usecase/mocks/mock_interfaces.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"
	time "time"

	domain "github.com/iho/goledger/internal/domain"
	usecase "github.com/iho/goledger/internal/usecase"
	decimal "github.com/shopspring/decimal"
	gomock "go.uber.org/mock/gomock"
)

// MockAccountRepository is a mock of AccountRepository interface.
type MockAccountRepository struct {
	ctrl     *gomock.Controller
	recorder *MockAccountRepositoryMockRecorder
	isgomock struct{}
}

// MockAccountRepositoryMockRecorder is the mock recorder for MockAccountRepository.
type MockAccountRepositoryMockRecorder struct {
	mock *MockAccountRepository
}

// NewMockAccountRepository creates a new mock instance.
func NewMockAccountRepository(ctrl *gomock.Controller) *MockAccountRepository {
	mock := &MockAccountRepository{ctrl: ctrl}
	mock.recorder = &MockAccountRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAccountRepository) EXPECT() *MockAccountRepositoryMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockAccountRepository) Create(ctx context.Context, account *domain.Account) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, account)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockAccountRepositoryMockRecorder) Create(ctx, account any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockAccountRepository)(nil).Create), ctx, account)
}

// GetByID mocks base method.
func (m *MockAccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, id)
	ret0, _ := ret[0].(*domain.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockAccountRepositoryMockRecorder) GetByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockAccountRepository)(nil).GetByID), ctx, id)
}

// GetByIDForUpdate mocks base method.
func (m *MockAccountRepository) GetByIDForUpdate(ctx context.Context, tx usecase.Transaction, id string) (*domain.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByIDForUpdate", ctx, tx, id)
	ret0, _ := ret[0].(*domain.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByIDForUpdate indicates an expected call of GetByIDForUpdate.
func (mr *MockAccountRepositoryMockRecorder) GetByIDForUpdate(ctx, tx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByIDForUpdate", reflect.TypeOf((*MockAccountRepository)(nil).GetByIDForUpdate), ctx, tx, id)
}

// GetByIDsForUpdate mocks base method.
func (m *MockAccountRepository) GetByIDsForUpdate(ctx context.Context, tx usecase.Transaction, ids []string) ([]*domain.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByIDsForUpdate", ctx, tx, ids)
	ret0, _ := ret[0].([]*domain.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByIDsForUpdate indicates an expected call of GetByIDsForUpdate.
func (mr *MockAccountRepositoryMockRecorder) GetByIDsForUpdate(ctx, tx, ids any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByIDsForUpdate", reflect.TypeOf((*MockAccountRepository)(nil).GetByIDsForUpdate), ctx, tx, ids)
}

// List mocks base method.
func (m *MockAccountRepository) List(ctx context.Context, limit, offset int) ([]*domain.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, limit, offset)
	ret0, _ := ret[0].([]*domain.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockAccountRepositoryMockRecorder) List(ctx, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockAccountRepository)(nil).List), ctx, limit, offset)
}

// UpdateBalance mocks base method.
func (m *MockAccountRepository) UpdateBalance(ctx context.Context, tx usecase.Transaction, id string, balance decimal.Decimal, updatedAt time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBalance", ctx, tx, id, balance, updatedAt)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateBalance indicates an expected call of UpdateBalance.
func (mr *MockAccountRepositoryMockRecorder) UpdateBalance(ctx, tx, id, balance, updatedAt any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBalance", reflect.TypeOf((*MockAccountRepository)(nil).UpdateBalance), ctx, tx, id, balance, updatedAt)
}

// MockTransferRepository is a mock of TransferRepository interface.
type MockTransferRepository struct {
	ctrl     *gomock.Controller
	recorder *MockTransferRepositoryMockRecorder
	isgomock struct{}
}

// MockTransferRepositoryMockRecorder is the mock recorder for MockTransferRepository.
type MockTransferRepositoryMockRecorder struct {
	mock *MockTransferRepository
}

// NewMockTransferRepository creates a new mock instance.
func NewMockTransferRepository(ctrl *gomock.Controller) *MockTransferRepository {
	mock := &MockTransferRepository{ctrl: ctrl}
	mock.recorder = &MockTransferRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTransferRepository) EXPECT() *MockTransferRepositoryMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockTransferRepository) Create(ctx context.Context, tx usecase.Transaction, transfer *domain.Transfer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, tx, transfer)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockTransferRepositoryMockRecorder) Create(ctx, tx, transfer any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockTransferRepository)(nil).Create), ctx, tx, transfer)
}

// GetByID mocks base method.
func (m *MockTransferRepository) GetByID(ctx context.Context, id string) (*domain.Transfer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, id)
	ret0, _ := ret[0].(*domain.Transfer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockTransferRepositoryMockRecorder) GetByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockTransferRepository)(nil).GetByID), ctx, id)
}

// ListByAccount mocks base method.
func (m *MockTransferRepository) ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transfer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListByAccount", ctx, accountID, limit, offset)
	ret0, _ := ret[0].([]*domain.Transfer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListByAccount indicates an expected call of ListByAccount.
func (mr *MockTransferRepositoryMockRecorder) ListByAccount(ctx, accountID, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListByAccount", reflect.TypeOf((*MockTransferRepository)(nil).ListByAccount), ctx, accountID, limit, offset)
}

// MockEntryRepository is a mock of EntryRepository interface.
type MockEntryRepository struct {
	ctrl     *gomock.Controller
	recorder *MockEntryRepositoryMockRecorder
	isgomock struct{}
}

// MockEntryRepositoryMockRecorder is the mock recorder for MockEntryRepository.
type MockEntryRepositoryMockRecorder struct {
	mock *MockEntryRepository
}

// NewMockEntryRepository creates a new mock instance.
func NewMockEntryRepository(ctrl *gomock.Controller) *MockEntryRepository {
	mock := &MockEntryRepository{ctrl: ctrl}
	mock.recorder = &MockEntryRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEntryRepository) EXPECT() *MockEntryRepositoryMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockEntryRepository) Create(ctx context.Context, tx usecase.Transaction, entry *domain.Entry) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, tx, entry)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockEntryRepositoryMockRecorder) Create(ctx, tx, entry any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockEntryRepository)(nil).Create), ctx, tx, entry)
}

// GetBalanceAtTime mocks base method.
func (m *MockEntryRepository) GetBalanceAtTime(ctx context.Context, accountID string, at time.Time) (decimal.Decimal, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBalanceAtTime", ctx, accountID, at)
	ret0, _ := ret[0].(decimal.Decimal)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBalanceAtTime indicates an expected call of GetBalanceAtTime.
func (mr *MockEntryRepositoryMockRecorder) GetBalanceAtTime(ctx, accountID, at any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBalanceAtTime", reflect.TypeOf((*MockEntryRepository)(nil).GetBalanceAtTime), ctx, accountID, at)
}

// GetByAccount mocks base method.
func (m *MockEntryRepository) GetByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.Entry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByAccount", ctx, accountID, limit, offset)
	ret0, _ := ret[0].([]*domain.Entry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByAccount indicates an expected call of GetByAccount.
func (mr *MockEntryRepositoryMockRecorder) GetByAccount(ctx, accountID, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByAccount", reflect.TypeOf((*MockEntryRepository)(nil).GetByAccount), ctx, accountID, limit, offset)
}

// GetByTransfer mocks base method.
func (m *MockEntryRepository) GetByTransfer(ctx context.Context, transferID string) ([]*domain.Entry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByTransfer", ctx, transferID)
	ret0, _ := ret[0].([]*domain.Entry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByTransfer indicates an expected call of GetByTransfer.
func (mr *MockEntryRepositoryMockRecorder) GetByTransfer(ctx, transferID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByTransfer", reflect.TypeOf((*MockEntryRepository)(nil).GetByTransfer), ctx, transferID)
}

// MockTransaction is a mock of Transaction interface.
type MockTransaction struct {
	ctrl     *gomock.Controller
	recorder *MockTransactionMockRecorder
	isgomock struct{}
}

// MockTransactionMockRecorder is the mock recorder for MockTransaction.
type MockTransactionMockRecorder struct {
	mock *MockTransaction
}

// NewMockTransaction creates a new mock instance.
func NewMockTransaction(ctrl *gomock.Controller) *MockTransaction {
	mock := &MockTransaction{ctrl: ctrl}
	mock.recorder = &MockTransactionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTransaction) EXPECT() *MockTransactionMockRecorder {
	return m.recorder
}

// Commit mocks base method.
func (m *MockTransaction) Commit(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Commit", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Commit indicates an expected call of Commit.
func (mr *MockTransactionMockRecorder) Commit(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Commit", reflect.TypeOf((*MockTransaction)(nil).Commit), ctx)
}

// Rollback mocks base method.
func (m *MockTransaction) Rollback(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Rollback", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Rollback indicates an expected call of Rollback.
func (mr *MockTransactionMockRecorder) Rollback(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Rollback", reflect.TypeOf((*MockTransaction)(nil).Rollback), ctx)
}

// MockTransactionManager is a mock of TransactionManager interface.
type MockTransactionManager struct {
	ctrl     *gomock.Controller
	recorder *MockTransactionManagerMockRecorder
	isgomock struct{}
}

// MockTransactionManagerMockRecorder is the mock recorder for MockTransactionManager.
type MockTransactionManagerMockRecorder struct {
	mock *MockTransactionManager
}

// NewMockTransactionManager creates a new mock instance.
func NewMockTransactionManager(ctrl *gomock.Controller) *MockTransactionManager {
	mock := &MockTransactionManager{ctrl: ctrl}
	mock.recorder = &MockTransactionManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTransactionManager) EXPECT() *MockTransactionManagerMockRecorder {
	return m.recorder
}

// Begin mocks base method.
func (m *MockTransactionManager) Begin(ctx context.Context) (usecase.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Begin", ctx)
	ret0, _ := ret[0].(usecase.Transaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Begin indicates an expected call of Begin.
func (mr *MockTransactionManagerMockRecorder) Begin(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Begin", reflect.TypeOf((*MockTransactionManager)(nil).Begin), ctx)
}

// MockIDGenerator is a mock of IDGenerator interface.
type MockIDGenerator struct {
	ctrl     *gomock.Controller
	recorder *MockIDGeneratorMockRecorder
	isgomock struct{}
}

// MockIDGeneratorMockRecorder is the mock recorder for MockIDGenerator.
type MockIDGeneratorMockRecorder struct {
	mock *MockIDGenerator
}

// NewMockIDGenerator creates a new mock instance.
func NewMockIDGenerator(ctrl *gomock.Controller) *MockIDGenerator {
	mock := &MockIDGenerator{ctrl: ctrl}
	mock.recorder = &MockIDGeneratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIDGenerator) EXPECT() *MockIDGeneratorMockRecorder {
	return m.recorder
}

// Generate mocks base method.
func (m *MockIDGenerator) Generate() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Generate")
	ret0, _ := ret[0].(string)
	return ret0
}

// Generate indicates an expected call of Generate.
func (mr *MockIDGeneratorMockRecorder) Generate() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Generate", reflect.TypeOf((*MockIDGenerator)(nil).Generate))
}

// MockCache is a mock of Cache interface.
type MockCache struct {
	ctrl     *gomock.Controller
	recorder *MockCacheMockRecorder
	isgomock struct{}
}

// MockCacheMockRecorder is the mock recorder for MockCache.
type MockCacheMockRecorder struct {
	mock *MockCache
}

// NewMockCache creates a new mock instance.
func NewMockCache(ctrl *gomock.Controller) *MockCache {
	mock := &MockCache{ctrl: ctrl}
	mock.recorder = &MockCacheMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCache) EXPECT() *MockCacheMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockCache) Delete(ctx context.Context, key string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, key)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockCacheMockRecorder) Delete(ctx, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockCache)(nil).Delete), ctx, key)
}

// Get mocks base method.
func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, key)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockCacheMockRecorder) Get(ctx, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCache)(nil).Get), ctx, key)
}

// Set mocks base method.
func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Set", ctx, key, value, ttl)
	ret0, _ := ret[0].(error)
	return ret0
}

// Set indicates an expected call of Set.
func (mr *MockCacheMockRecorder) Set(ctx, key, value, ttl any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Set", reflect.TypeOf((*MockCache)(nil).Set), ctx, key, value, ttl)
}

// MockIdempotencyStore is a mock of IdempotencyStore interface.
type MockIdempotencyStore struct {
	ctrl     *gomock.Controller
	recorder *MockIdempotencyStoreMockRecorder
	isgomock struct{}
}

// MockIdempotencyStoreMockRecorder is the mock recorder for MockIdempotencyStore.
type MockIdempotencyStoreMockRecorder struct {
	mock *MockIdempotencyStore
}

// NewMockIdempotencyStore creates a new mock instance.
func NewMockIdempotencyStore(ctrl *gomock.Controller) *MockIdempotencyStore {
	mock := &MockIdempotencyStore{ctrl: ctrl}
	mock.recorder = &MockIdempotencyStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIdempotencyStore) EXPECT() *MockIdempotencyStoreMockRecorder {
	return m.recorder
}

// CheckAndSet mocks base method.
func (m *MockIdempotencyStore) CheckAndSet(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckAndSet", ctx, key, response, ttl)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].([]byte)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CheckAndSet indicates an expected call of CheckAndSet.
func (mr *MockIdempotencyStoreMockRecorder) CheckAndSet(ctx, key, response, ttl any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckAndSet", reflect.TypeOf((*MockIdempotencyStore)(nil).CheckAndSet), ctx, key, response, ttl)
}

// Update mocks base method.
func (m *MockIdempotencyStore) Update(ctx context.Context, key string, response []byte, ttl time.Duration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, key, response, ttl)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockIdempotencyStoreMockRecorder) Update(ctx, key, response, ttl any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockIdempotencyStore)(nil).Update), ctx, key, response, ttl)
}
