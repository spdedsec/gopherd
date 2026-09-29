package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/spdedsec/gopherd/internal/model"
	"github.com/spdedsec/gopherd/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type AuthService struct {
	users    *repository.UserRepository
	sessions *repository.SessionRepository
	ttl      time.Duration
}

func NewAuthService(u *repository.UserRepository, s *repository.SessionRepository, ttl time.Duration) *AuthService {
	return &AuthService{users: u, sessions: s, ttl: ttl}
}
func normalizeEmail(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
func (s *AuthService) Register(ctx context.Context, email, password string) (model.User, string, error) {
	email = normalizeEmail(email)
	if err := validateCredentials(email, password); err != nil {
		return model.User{}, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, "", err
	}
	u, err := s.users.Create(ctx, email, string(hash))
	if err != nil {
		return model.User{}, "", err
	}
	token, err := s.createSession(ctx, u.ID)
	return u, token, err
}
func (s *AuthService) Login(ctx context.Context, email, password string) (model.User, string, error) {
	u, hash, err := s.users.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		return model.User{}, "", ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return model.User{}, "", ErrInvalidCredentials
	}
	token, err := s.createSession(ctx, u.ID)
	return u, token, err
}
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessions.DeleteByTokenHash(ctx, hashToken(token))
}
func (s *AuthService) Authenticate(ctx context.Context, token string) (model.User, error) {
	if token == "" {
		return model.User{}, repository.ErrNotFound
	}
	return s.sessions.UserByTokenHash(ctx, hashToken(token))
}
func (s *AuthService) createSession(ctx context.Context, userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	return token, s.sessions.Create(ctx, userID, hashToken(token), time.Now().Add(s.ttl))
}
func hashToken(v string) string { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }
func validateCredentials(email, password string) error {
	if len(email) < 5 || !strings.Contains(email, "@") {
		return errors.New("valid email is required")
	}
	if len(password) < 10 {
		return errors.New("password must be at least 10 characters")
	}
	return nil
}
