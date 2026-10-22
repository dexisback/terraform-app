package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/amaanworks/todo-app/internal/todo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS todos (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
			completed BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("ensure database schema: %w", err)
	}
	return nil
}

func (s *Store) List(ctx context.Context) ([]todo.Todo, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, completed, created_at, updated_at
		FROM todos ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}
	defer rows.Close()

	items := make([]todo.Todo, 0)
	for rows.Next() {
		var item todo.Todo
		if err := rows.Scan(&item.ID, &item.Title, &item.Completed, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan todo: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate todos: %w", err)
	}
	return items, nil
}

func (s *Store) Get(ctx context.Context, id int64) (todo.Todo, error) {
	var item todo.Todo
	err := s.pool.QueryRow(ctx, `
		SELECT id, title, completed, created_at, updated_at
		FROM todos WHERE id = $1
	`, id).Scan(&item.ID, &item.Title, &item.Completed, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return todo.Todo{}, todo.ErrNotFound
	}
	if err != nil {
		return todo.Todo{}, fmt.Errorf("get todo: %w", err)
	}
	return item, nil
}

func (s *Store) Create(ctx context.Context, title string) (todo.Todo, error) {
	var item todo.Todo
	err := s.pool.QueryRow(ctx, `
		INSERT INTO todos (title) VALUES ($1)
		RETURNING id, title, completed, created_at, updated_at
	`, title).Scan(&item.ID, &item.Title, &item.Completed, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return todo.Todo{}, fmt.Errorf("create todo: %w", err)
	}
	return item, nil
}

func (s *Store) Update(ctx context.Context, id int64, input todo.UpdateInput) (todo.Todo, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return todo.Todo{}, err
	}
	if input.Title != nil {
		current.Title = *input.Title
	}
	if input.Completed != nil {
		current.Completed = *input.Completed
	}

	var item todo.Todo
	err = s.pool.QueryRow(ctx, `
		UPDATE todos SET title = $2, completed = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, title, completed, created_at, updated_at
	`, id, strings.TrimSpace(current.Title), current.Completed).
		Scan(&item.ID, &item.Title, &item.Completed, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return todo.Todo{}, fmt.Errorf("update todo: %w", err)
	}
	return item, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM todos WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return todo.ErrNotFound
	}
	return nil
}
