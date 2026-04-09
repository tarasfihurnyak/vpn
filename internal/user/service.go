package user

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	db "vpn/internal/db/sqlc"
)

type Service struct {
	q db.Querier
}

func NewService(q db.Querier) *Service {
	return &Service{q: q}
}

func toModel(u db.User) User {
	return User{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		PublicKey: u.PublicKey.String,
		CreatedAt: u.CreatedAt.Time,
		UpdatedAt: u.UpdatedAt.Time,
	}
}

func (s *Service) Create(ctx context.Context, username, email string) (User, error) {
	u, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Username: username,
		Email:    email,
	})
	if err != nil {
		return User{}, err
	}
	return toModel(u), nil
}

func (s *Service) GetByID(ctx context.Context, id pgtype.UUID) (User, error) {
	u, err := s.q.GetUser(ctx, id)
	if err != nil {
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

func (s *Service) UpdatePublicKey(ctx context.Context, id pgtype.UUID, publicKey string) (User, error) {
	u, err := s.q.UpdateUserPublicKey(ctx, db.UpdateUserPublicKeyParams{
		ID:        id,
		PublicKey: pgtype.Text{String: publicKey, Valid: true},
	})
	if err != nil {
		return User{}, err
	}
	return toModel(u), nil
}

func (s *Service) Delete(ctx context.Context, id pgtype.UUID) error {
	return s.q.DeleteUser(ctx, id)
}
