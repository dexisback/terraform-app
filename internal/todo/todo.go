package todo

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("todo not found")
	ErrInvalid  = errors.New("invalid todo")
)

type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateInput struct {
	Title string `json:"title"`
}

type UpdateInput struct {
	Title     *string `json:"title,omitempty"`
	Completed *bool   `json:"completed,omitempty"`
}

// Store isolates HTTP handlers from persistence. We can replace PostgreSQL or
// add a test double without changing the API layer.
type Store interface {
	List(ctx context.Context) ([]Todo, error)
	Get(ctx context.Context, id int64) (Todo, error)
	Create(ctx context.Context, title string) (Todo, error)
	Update(ctx context.Context, id int64, input UpdateInput) (Todo, error)
	Delete(ctx context.Context, id int64) error
}
