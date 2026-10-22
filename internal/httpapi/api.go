package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/amaanworks/todo-app/internal/todo"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type API struct {
	store  todo.Store
	logger *slog.Logger
}

func New(store todo.Store, logger *slog.Logger) *API {
	return &API{store: store, logger: logger}
}

func (a *API) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(a.requestLogger)

	r.Get("/healthz", a.health)
	r.Get("/readyz", a.ready)
	r.Route("/api/v1/todos", func(r chi.Router) {
		r.Get("/", a.list)
		r.Post("/", a.create)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", a.get)
			r.Patch("/", a.update)
			r.Delete("/", a.delete)
		})
	})
	return r
}

func (a *API) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		a.logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", chimiddleware.GetReqID(r.Context()),
		)
	})
}

// Liveness means the process is alive; it intentionally does not depend on DB.
func (a *API) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readiness checks whether the application can currently serve requests.
func (a *API) ready(w http.ResponseWriter, r *http.Request) {
	if _, err := a.store.List(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *API) list(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.List(r.Context())
	if err != nil {
		a.internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) create(w http.ResponseWriter, r *http.Request) {
	var input todo.CreateInput
	if !decodeJSON(w, r, &input) {
		return
	}
	title := strings.TrimSpace(input.Title)
	if title == "" || len([]rune(title)) > 200 {
		writeError(w, http.StatusBadRequest, "title must contain between 1 and 200 characters")
		return
	}
	item, err := a.store.Create(r.Context(), title)
	if err != nil {
		a.internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	item, err := a.store.Get(r.Context(), id)
	if errors.Is(err, todo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		a.internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var input todo.UpdateInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Title == nil && input.Completed == nil {
		writeError(w, http.StatusBadRequest, "provide title and/or completed")
		return
	}
	if input.Title != nil {
		trimmed := strings.TrimSpace(*input.Title)
		if trimmed == "" || len([]rune(trimmed)) > 200 {
			writeError(w, http.StatusBadRequest, "title must contain between 1 and 200 characters")
			return
		}
		input.Title = &trimmed
	}
	item, err := a.store.Update(r.Context(), id, input)
	if errors.Is(err, todo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		a.internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := a.store.Delete(r.Context(), id); errors.Is(err, todo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	} else if err != nil {
		a.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if decoder.Decode(&struct{}{}) == nil {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return false
	}
	return true
}

func (a *API) internalError(w http.ResponseWriter, err error) {
	a.logger.Error("request failed", "error", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
