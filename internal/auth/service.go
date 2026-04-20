package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "vpn/internal/db/sqlc"
	"vpn/internal/user"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrTokenExpired       = errors.New("token expired")
)

type claims struct {
	jwt.RegisteredClaims
}

// Service handles JWT issuance/validation and refresh token lifecycle.
type Service struct {
	users      *user.Service
	q          db.Querier
	privateKey *ecdsa.PrivateKey
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewService(
	users *user.Service,
	q db.Querier,
	privateKey *ecdsa.PrivateKey,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *Service {
	return &Service{
		users:      users,
		q:          q,
		privateKey: privateKey,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Login authenticates a user by login (username or email) and password,
// returning a JWT access token and a raw refresh token.
func (s *Service) Login(ctx context.Context, login, password string) (TokenPair, string, error) {
	u, err := s.users.Authenticate(ctx, login, password)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			return TokenPair{}, "", ErrInvalidCredentials
		}
		return TokenPair{}, "", fmt.Errorf("authenticate: %w", err)
	}
	return s.issueTokens(ctx, u.ID)
}

// Refresh validates the raw refresh token, invalidates it, and issues a new token pair.
// If the token is already revoked (replay attack), all user tokens are revoked.
func (s *Service) Refresh(ctx context.Context, rawToken string) (TokenPair, string, error) {
	tokenHash := hashToken(rawToken)

	rt, err := s.q.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenPair{}, "", ErrInvalidToken
		}
		return TokenPair{}, "", fmt.Errorf("get refresh token: %w", err)
	}

	if rt.RevokedAt != nil {
		// replay attack detected — revoke all tokens for this user
		_ = s.q.RevokeAllUserRefreshTokens(ctx, rt.UserID)
		return TokenPair{}, "", ErrTokenRevoked
	}

	if rt.ExpiresAt.Before(time.Now()) {
		return TokenPair{}, "", ErrTokenExpired
	}

	if err := s.q.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return TokenPair{}, "", fmt.Errorf("revoke refresh token: %w", err)
	}

	return s.issueTokens(ctx, rt.UserID)
}

// Logout revokes the refresh token (best-effort; errors are ignored by the handler).
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	return s.q.RevokeRefreshToken(ctx, hashToken(rawToken))
}

// ValidateAccessToken parses and validates an ES256-signed JWT, returning the user ID.
// The key function explicitly rejects any algorithm other than ES256.
func (s *Service) ValidateAccessToken(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodES256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return &s.privateKey.PublicKey, nil
	})
	if err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}

	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return uuid.UUID{}, ErrInvalidToken
	}

	parsed, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}
	return parsed, nil
}

func (s *Service) issueTokens(ctx context.Context, userID uuid.UUID) (TokenPair, string, error) {
	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return TokenPair{}, "", fmt.Errorf("generate access token: %w", err)
	}

	rawRefresh, err := s.generateRefreshToken(ctx, userID)
	if err != nil {
		return TokenPair{}, "", fmt.Errorf("generate refresh token: %w", err)
	}

	return TokenPair{AccessToken: accessToken}, rawRefresh, nil
}

func (s *Service) generateAccessToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodES256, c).SignedString(s.privateKey)
}

func (s *Service) generateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	rawStr := hex.EncodeToString(raw)

	if _, err := s.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: hashToken(rawStr),
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}); err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}

	return rawStr, nil
}

// hashToken returns the hex-encoded SHA-256 hash of the raw token string.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// LoadPrivateKey reads and parses an EC private key from a PEM file.
func LoadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block from private key file")
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse EC private key: %w", err)
	}
	return key, nil
}
