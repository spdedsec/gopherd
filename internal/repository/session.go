package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spdedsec/gopherd/internal/model"
)

type SessionRepository struct{ db *pgxpool.Pool }

func NewSessionRepository(db *pgxpool.Pool) *SessionRepository { return &SessionRepository{db: db} }
func (r *SessionRepository) Create(ctx context.Context, userID, tokenHash string, expires time.Time) error {
	_, err := r.db.Exec(ctx, `INSERT INTO sessions(id,user_id,token_hash,expires_at) VALUES($1,$2,$3,$4)`, uuid.New(), userID, tokenHash, expires)
	return err
}
func (r *SessionRepository) UserByTokenHash(ctx context.Context, tokenHash string) (model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx, `SELECT u.id,u.email,u.created_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>NOW()`, tokenHash).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return model.User{}, ErrNotFound
	}
	return u, err
}
func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, tokenHash)
	return err
}
