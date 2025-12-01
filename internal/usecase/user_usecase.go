package usecase

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/iho/goledger/internal/domain"
)

// UserRepository defines the interface for user persistence
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)
}

// UserUseCase handles user management operations
type UserUseCase struct {
	userRepo UserRepository
}

// NewUserUseCase creates a new user use case
func NewUserUseCase(userRepo UserRepository) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
	}
}

// CreateUserInput represents input for creating a user
type CreateUserInput struct {
	Email    string
	Name     string
	Password string
	Role     domain.Role
}

// CreateUser creates a new user with hashed password
func (uc *UserUseCase) CreateUser(ctx context.Context, input CreateUserInput) (*domain.User, error) {
	// Validate email
	if err := domain.ValidateEmail(input.Email); err != nil {
		return nil, err
	}

	// Validate password
	if err := domain.ValidatePassword(input.Password); err != nil {
		return nil, err
	}

	// Validate role
	if !input.Role.IsValid() {
		return nil, errors.New("invalid role")
	}

	// Check if user already exists
	existing, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err == nil && existing != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := hashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:             generateID(),
		Email:          input.Email,
		Name:           input.Name,
		HashedPassword: hashedPassword,
		Role:           input.Role,
		Active:         true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Don't return hashed password
	user.HashedPassword = ""
	return user, nil
}

// AuthenticateInput represents authentication input
type AuthenticateInput struct {
	Email    string
	Password string
}

// Authenticate verifies user credentials
func (uc *UserUseCase) Authenticate(ctx context.Context, input AuthenticateInput) (*domain.User, error) {
	user, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if !user.Active {
		return nil, errors.New("user account is inactive")
	}

	// Verify password
	if err := verifyPassword(user.HashedPassword, input.Password); err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Don't return hashed password
	user.HashedPassword = ""
	return user, nil
}

// GetUser retrieves a user by ID
func (uc *UserUseCase) GetUser(ctx context.Context, id string) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.HashedPassword = ""
	return user, nil
}

// UpdateUserInput represents input for updating a user
type UpdateUserInput struct {
	ID       string
	Name     *string
	Role     *domain.Role
	Active   *bool
	Password *string
}

// UpdateUser updates user information
func (uc *UserUseCase) UpdateUser(ctx context.Context, input UpdateUserInput) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		user.Name = *input.Name
	}

	if input.Role != nil {
		if !input.Role.IsValid() {
			return nil, errors.New("invalid role")
		}
		user.Role = *input.Role
	}

	if input.Active != nil {
		user.Active = *input.Active
	}

	if input.Password != nil {
		if err := domain.ValidatePassword(*input.Password); err != nil {
			return nil, err
		}
		hashedPassword, err := hashPassword(*input.Password)
		if err != nil {
			return nil, err
		}
		user.HashedPassword = hashedPassword
	}

	user.UpdatedAt = time.Now().UTC()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	user.HashedPassword = ""
	return user, nil
}

// DeleteUser deletes a user
func (uc *UserUseCase) DeleteUser(ctx context.Context, id string) error {
	return uc.userRepo.Delete(ctx, id)
}

// ListUsers lists all users with pagination
func (uc *UserUseCase) ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	limit, offset, _ = domain.ValidatePagination(limit, offset)

	users, err := uc.userRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Remove hashed passwords
	for _, user := range users {
		user.HashedPassword = ""
	}

	return users, nil
}

// hashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// verifyPassword verifies a password against a hash
func verifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// generateID generates a unique ID for a user
func generateID() string {
	return "user-" + time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
