package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spdedsec/gopherd/internal/model"
)

type TaskRepository struct{ db *pgxpool.Pool }

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository { return &TaskRepository{db: db} }
func (r *TaskRepository) Create(ctx context.Context, userID, title, description string) (model.Task, error) {
	var t model.Task
	id := uuid.New()
	err := r.db.QueryRow(ctx, `INSERT INTO tasks(id,user_id,title,description) VALUES($1,$2,$3,$4) RETURNING id,title,description,completed,created_at,updated_at`, id, userID, title, description).Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}
func (r *TaskRepository) List(ctx context.Context, userID string, completed *bool, limit, offset int) ([]model.Task, error) {
	args := []any{userID}
	where := `WHERE user_id=$1`
	n := 2
	if completed != nil {
		where += fmt.Sprintf(` AND completed=$%d`, n)
		args = append(args, *completed)
		n++
	}
	args = append(args, limit, offset)
	q := fmt.Sprintf(`SELECT id,title,description,completed,created_at,updated_at FROM tasks %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, n, n+1)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Task
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (r *TaskRepository) Get(ctx context.Context, userID, id string) (model.Task, error) {
	var t model.Task
	err := r.db.QueryRow(ctx, `SELECT id,title,description,completed,created_at,updated_at FROM tasks WHERE id=$1 AND user_id=$2`, id, userID).Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return model.Task{}, ErrNotFound
	}
	return t, err
}
func (r *TaskRepository) Update(ctx context.Context, userID, id, title, description string, completed bool) (model.Task, error) {
	var t model.Task
	err := r.db.QueryRow(ctx, `UPDATE tasks SET title=$1,description=$2,completed=$3,updated_at=NOW() WHERE id=$4 AND user_id=$5 RETURNING id,title,description,completed,created_at,updated_at`, title, description, completed, id, userID).Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return model.Task{}, ErrNotFound
	}
	return t, err
}
func (r *TaskRepository) Delete(ctx context.Context, userID, id string) error {
	res, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
