package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amaanworks/todo-app/internal/todo"
)

type fakeStore struct{}

func (fakeStore) List(context.Context) ([]todo.Todo, error) { return []todo.Todo{}, nil }
func (fakeStore) Get(context.Context, int64) (todo.Todo, error) {
	return todo.Todo{}, todo.ErrNotFound
}
func (fakeStore) Create(context.Context, string) (todo.Todo, error) {
	return todo.Todo{}, nil
}
func (fakeStore) Update(context.Context, int64, todo.UpdateInput) (todo.Todo, error) {
	return todo.Todo{}, todo.ErrNotFound
}
func (fakeStore) Delete(context.Context, int64) error { return todo.ErrNotFound }

func TestHealthz(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(fakeStore{}, logger).Handler()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content type, got %q", got)
	}
}
