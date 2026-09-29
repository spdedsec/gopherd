package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spdedsec/gopherd/internal/model"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

type UserRepository struct{ db *pgxpool.Pool }

func NewUserRepository(db *pgxpool.Pool) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (model.User, error) {
	id := uuid.New()
	var u model.User
	err := r.db.QueryRow(ctx, `INSERT INTO users(id,email,password_hash) VALUES($1,$2,$3) RETURNING id,email,created_at`, id, email, passwordHash).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.User{}, ErrConflict
		}
		return model.User{}, err
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (model.User, string, error) {
	var u model.User
	var hash string
	err := r.db.QueryRow(ctx, `SELECT id,email,password_hash,created_at FROM users WHERE email=$1`, email).Scan(&u.ID, &u.Email, &hash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, "", ErrNotFound
	}
	return u, hash, err
}
