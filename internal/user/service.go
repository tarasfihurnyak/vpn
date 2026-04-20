package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	db "vpn/internal/db/sqlc"
)

type Service struct {
	q          db.Querier
	bcryptCost int
}

func NewService(q db.Querier) *Service {
	return &Service{q: q, bcryptCost: bcrypt.DefaultCost}
}

// NewServiceWithMinBcryptCost is a test helper that creates a Service with bcrypt.MinCost to speed up tests
func NewServiceWithMinBcryptCost(q db.Querier) *Service {
	return &Service{q: q, bcryptCost: bcrypt.MinCost}
}

func toModel(u db.User) User {
	var pk string
	if u.PublicKey != nil {
		pk = *u.PublicKey
	}
	return User{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		PublicKey: pk,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func (s *Service) Create(ctx context.Context, username, email, password string) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	u, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
	})
	if err != nil {
		return User{}, err
	}
	return toModel(u), nil
}

// Authenticate looks up a user by email or username and verifies the password.
func (s *Service) Authenticate(ctx context.Context, login, password string) (User, error) {
	var dbUser db.User
	var err error

	if strings.Contains(login, "@") {
		dbUser, err = s.q.GetUserByEmail(ctx, login)
	} else {
		dbUser, err = s.q.GetUserByUsername(ctx, login)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}

	return toModel(dbUser), nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	u, err := s.q.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return toModel(u), nil
}

func (s *Service) GetByEmail(ctx context.Context, email string) (User, error) {
	u, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, err
	}
	return toModel(u), nil
}

func (s *Service) GetByUsername(ctx context.Context, username string) (User, error) {
	u, err := s.q.GetUserByUsername(ctx, username)
	if err != nil {
		return User{}, err
	}
	return toModel(u), nil
}

func (s *Service) List(ctx context.Context) ([]User, error) {
	users, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]User, len(users))
	for i, u := range users {
		result[i] = toModel(u)
	}
	return result, nil
}

func (s *Service) UpdatePublicKey(ctx context.Context, id uuid.UUID, publicKey string) (User, error) {
	u, err := s.q.UpdateUserPublicKey(ctx, db.UpdateUserPublicKeyParams{
		ID:        id,
		PublicKey: &publicKey,
	})
	if err != nil {
		return User{}, err
	}
	return toModel(u), nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteUser(ctx, id)
}
