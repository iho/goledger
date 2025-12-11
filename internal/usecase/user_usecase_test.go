package usecase_test

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
)

type stubUserRepo struct {
	createFn     func(ctx context.Context, user *domain.User) error
	getByIDFn    func(ctx context.Context, id string) (*domain.User, error)
	getByEmailFn func(ctx context.Context, email string) (*domain.User, error)
	updateFn     func(ctx context.Context, user *domain.User) error
	deleteFn     func(ctx context.Context, id string) error
	listFn       func(ctx context.Context, limit, offset int) ([]*domain.User, error)
}

func (s *stubUserRepo) Create(ctx context.Context, user *domain.User) error {
	if s.createFn != nil {
		return s.createFn(ctx, user)
	}
	return nil
}

func (s *stubUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (s *stubUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if s.getByEmailFn != nil {
		return s.getByEmailFn(ctx, email)
	}
	return nil, nil
}

func (s *stubUserRepo) Update(ctx context.Context, user *domain.User) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, user)
	}
	return nil
}

func (s *stubUserRepo) Delete(ctx context.Context, id string) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

func (s *stubUserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	if s.listFn != nil {
		return s.listFn(ctx, limit, offset)
	}
	return nil, nil
}

func TestUserUseCase_CreateUser_Success(t *testing.T) {
	t.Parallel()

	var stored *domain.User
	repo := &stubUserRepo{
		getByEmailFn: func(context.Context, string) (*domain.User, error) {
			return nil, nil
		},
		createFn: func(_ context.Context, user *domain.User) error {
			if user.HashedPassword == "" {
				t.Fatal("expected user to be persisted with hashed password")
			}
			stored = &domain.User{
				ID:             user.ID,
				Email:          user.Email,
				Name:           user.Name,
				HashedPassword: user.HashedPassword,
				Role:           user.Role,
				Active:         user.Active,
			}
			return nil
		},
	}

	uc := usecase.NewUserUseCase(repo)

	user, err := uc.CreateUser(context.Background(), usecase.CreateUserInput{
		Email:    "user@example.com",
		Name:     "Alice",
		Password: "StrongPass1",
		Role:     domain.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stored == nil {
		t.Fatal("expected user to be stored")
	}

	if user.HashedPassword != "" {
		t.Fatal("expected returned user to hide hashed password")
	}
}

func TestUserUseCase_CreateUser_ValidationErrors(t *testing.T) {
	t.Parallel()

	uc := usecase.NewUserUseCase(&stubUserRepo{})

	_, err := uc.CreateUser(context.Background(), usecase.CreateUserInput{
		Email:    "invalid-email",
		Name:     "Bob",
		Password: "StrongPass1",
		Role:     domain.RoleAdmin,
	})
	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}

	_, err = uc.CreateUser(context.Background(), usecase.CreateUserInput{
		Email:    "user@example.com",
		Name:     "Bob",
		Password: "weak",
		Role:     domain.RoleAdmin,
	})
	if !errors.Is(err, domain.ErrPasswordTooWeak) {
		t.Fatalf("expected ErrPasswordTooWeak, got %v", err)
	}

	_, err = uc.CreateUser(context.Background(), usecase.CreateUserInput{
		Email:    "user@example.com",
		Name:     "Bob",
		Password: "StrongPass1",
		Role:     "invalid",
	})
	if err == nil || err.Error() != "invalid role" {
		t.Fatalf("expected invalid role error, got %v", err)
	}
}

func TestUserUseCase_CreateUser_DuplicateEmail(t *testing.T) {
	t.Parallel()

	repo := &stubUserRepo{
		getByEmailFn: func(context.Context, string) (*domain.User, error) {
			return &domain.User{ID: "existing"}, nil
		},
	}

	uc := usecase.NewUserUseCase(repo)

	_, err := uc.CreateUser(context.Background(), usecase.CreateUserInput{
		Email:    "user@example.com",
		Name:     "Bob",
		Password: "StrongPass1",
		Role:     domain.RoleAdmin,
	})
	if err == nil || err.Error() != "user with this email already exists" {
		t.Fatalf("expected duplicate email error, got %v", err)
	}
}

func TestUserUseCase_Authenticate(t *testing.T) {
	t.Parallel()

	hashed, err := bcrypt.GenerateFromPassword([]byte("StrongPass1"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	activeUser := &domain.User{
		ID:             "user-1",
		Email:          "user@example.com",
		HashedPassword: string(hashed),
		Active:         true,
	}

	repo := &stubUserRepo{
		getByEmailFn: func(context.Context, string) (*domain.User, error) {
			copied := *activeUser
			return &copied, nil
		},
	}

	uc := usecase.NewUserUseCase(repo)

	user, err := uc.Authenticate(context.Background(), usecase.AuthenticateInput{
		Email:    "user@example.com",
		Password: "StrongPass1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID != activeUser.ID {
		t.Fatalf("expected user ID %s, got %s", activeUser.ID, user.ID)
	}
	if user.HashedPassword != "" {
		t.Fatal("expected returned user to hide hashed password")
	}
}

func TestUserUseCase_AuthenticateErrors(t *testing.T) {
	t.Parallel()

	hashed, err := bcrypt.GenerateFromPassword([]byte("StrongPass1"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	inactiveUser := &domain.User{
		Email:          "user@example.com",
		HashedPassword: string(hashed),
		Active:         false,
	}

	repo := &stubUserRepo{
		getByEmailFn: func(context.Context, string) (*domain.User, error) {
			copied := *inactiveUser
			return &copied, nil
		},
	}

	uc := usecase.NewUserUseCase(repo)

	_, err = uc.Authenticate(context.Background(), usecase.AuthenticateInput{
		Email:    "user@example.com",
		Password: "StrongPass1",
	})
	if err == nil || err.Error() != "user account is inactive" {
		t.Fatalf("expected inactive error, got %v", err)
	}

	repo.getByEmailFn = func(context.Context, string) (*domain.User, error) {
		copied := *inactiveUser
		copied.Active = true
		return &copied, nil
	}

	_, err = uc.Authenticate(context.Background(), usecase.AuthenticateInput{
		Email:    "user@example.com",
		Password: "WrongPass1",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}

	repo.getByEmailFn = func(context.Context, string) (*domain.User, error) {
		return nil, errors.New("not found")
	}

	_, err = uc.Authenticate(context.Background(), usecase.AuthenticateInput{
		Email:    "missing@example.com",
		Password: "StrongPass1",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for missing user, got %v", err)
	}
}

func TestUserUseCase_GetUser(t *testing.T) {
	t.Parallel()

	repo := &stubUserRepo{
		getByIDFn: func(context.Context, string) (*domain.User, error) {
			return &domain.User{ID: "user-1", HashedPassword: "secret"}, nil
		},
	}

	uc := usecase.NewUserUseCase(repo)
	user, err := uc.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.HashedPassword != "" {
		t.Fatal("expected hashed password to be hidden")
	}
}

func TestUserUseCase_UpdateUser(t *testing.T) {
	t.Parallel()

	original := &domain.User{
		ID:             "user-1",
		Name:           "Alice",
		Role:           domain.RoleViewer,
		Active:         true,
		HashedPassword: "old",
	}

	var updated *domain.User
	repo := &stubUserRepo{
		getByIDFn: func(context.Context, string) (*domain.User, error) {
			copyUser := *original
			return &copyUser, nil
		},
		updateFn: func(_ context.Context, user *domain.User) error {
			updated = user
			return nil
		},
	}

	newName := "Bob"
	newRole := domain.RoleOperator
	active := false
	newPassword := "NewStrong1"

	uc := usecase.NewUserUseCase(repo)
	user, err := uc.UpdateUser(context.Background(), usecase.UpdateUserInput{
		ID:       "user-1",
		Name:     &newName,
		Role:     &newRole,
		Active:   &active,
		Password: &newPassword,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated == nil {
		t.Fatal("expected update to be called")
	}

	if updated.Name != newName || updated.Role != newRole || updated.Active != active {
		t.Fatalf("unexpected updated fields: %+v", updated)
	}

	if updated.HashedPassword == "old" {
		t.Fatal("expected password to be rehashed")
	}

	if user.HashedPassword != "" {
		t.Fatal("expected masked password in response")
	}
}

func TestUserUseCase_UpdateUser_InvalidRole(t *testing.T) {
	t.Parallel()

	invalidRole := domain.Role("invalid")
	repo := &stubUserRepo{
		getByIDFn: func(context.Context, string) (*domain.User, error) {
			return &domain.User{ID: "user-1"}, nil
		},
	}

	uc := usecase.NewUserUseCase(repo)
	_, err := uc.UpdateUser(context.Background(), usecase.UpdateUserInput{
		ID:   "user-1",
		Role: &invalidRole,
	})
	if err == nil || err.Error() != "invalid role" {
		t.Fatalf("expected invalid role error, got %v", err)
	}
}

func TestUserUseCase_UpdateUser_InvalidPassword(t *testing.T) {
	t.Parallel()

	repo := &stubUserRepo{
		getByIDFn: func(context.Context, string) (*domain.User, error) {
			return &domain.User{ID: "user-1"}, nil
		},
	}

	uc := usecase.NewUserUseCase(repo)
	weak := "weak"
	_, err := uc.UpdateUser(context.Background(), usecase.UpdateUserInput{
		ID:       "user-1",
		Password: &weak,
	})
	if !errors.Is(err, domain.ErrPasswordTooWeak) {
		t.Fatalf("expected ErrPasswordTooWeak, got %v", err)
	}
}

func TestUserUseCase_DeleteUser(t *testing.T) {
	t.Parallel()

	var deletedID string
	repo := &stubUserRepo{
		deleteFn: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	}

	uc := usecase.NewUserUseCase(repo)
	if err := uc.DeleteUser(context.Background(), "user-123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deletedID != "user-123" {
		t.Fatalf("expected delete to be called with user-123, got %s", deletedID)
	}
}

func TestUserUseCase_ListUsers(t *testing.T) {
	t.Parallel()

	capturedLimit := 0
	capturedOffset := 0

	repo := &stubUserRepo{
		listFn: func(_ context.Context, limit, offset int) ([]*domain.User, error) {
			capturedLimit = limit
			capturedOffset = offset
			return []*domain.User{
				{ID: "user-1", HashedPassword: "secret"},
			}, nil
		},
	}

	uc := usecase.NewUserUseCase(repo)
	users, err := uc.ListUsers(context.Background(), 0, -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const DefaultPageSize = 50
	if capturedLimit != DefaultPageSize || capturedOffset != 0 {
		t.Fatalf("expected sanitized pagination, got limit=%d offset=%d", capturedLimit, capturedOffset)
	}

	if len(users) != 1 {
		t.Fatalf("expected one user, got %d", len(users))
	}

	if users[0].HashedPassword != "" {
		t.Fatal("expected hashed password to be hidden")
	}
}
