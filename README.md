# Todo API

A small Go + PostgreSQL API used as the application repository for a platform-engineering learning project. The app stays intentionally simple; the infrastructure and operating practices will grow around it.

## Prerequisites

- Go 1.23+
- Docker Engine and Docker Compose plugin

## Run locally with Go

Start PostgreSQL:

```bash
docker compose up -d db
```

Download dependencies and run the API:

```bash
go mod tidy
go run ./cmd/api
```

The API listens on `http://localhost:8080`.

## Run everything in containers

```bash
docker compose up --build
```

View logs:

```bash
docker compose logs -f api
```

Stop containers (keep database data):

```bash
docker compose down
```

Reset local database data (destructive):

```bash
docker compose down -v
```

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| GET | `/healthz` | Process liveness |
| GET | `/readyz` | Readiness / database check |
| GET | `/api/v1/todos/` | List todos |
| POST | `/api/v1/todos/` | Create a todo |
| GET | `/api/v1/todos/{id}/` | Get one todo |
| PATCH | `/api/v1/todos/{id}/` | Update title and/or completed |
| DELETE | `/api/v1/todos/{id}/` | Delete a todo |

Create:

```bash
curl -i -X POST http://localhost:8080/api/v1/todos/ \
  -H 'Content-Type: application/json' \
  -d '{"title":"Learn Kubernetes probes"}'
```

List:

```bash
curl -s http://localhost:8080/api/v1/todos/ | jq
```

Complete todo 1:

```bash
curl -i -X PATCH http://localhost:8080/api/v1/todos/1/ \
  -H 'Content-Type: application/json' \
  -d '{"completed":true}'
```

Delete todo 1:

```bash
curl -i -X DELETE http://localhost:8080/api/v1/todos/1/
```

## Initial design decisions

- `internal/todo.Store` separates the HTTP API from persistence.
- `internal/store/postgres` owns PostgreSQL access.
- `/healthz` and `/readyz` are separate so Kubernetes can later use liveness and readiness probes differently.
- JSON structured logs and request IDs support later log aggregation and incident diagnosis.
- Graceful shutdown is important when Kubernetes replaces pods during deployments.
- The initial database schema is created on startup to keep the first local run simple. Before production use, replace this with a versioned migration tool and expand/contract migration practices.
- The API container has no shell or curl because it uses a minimal distroless runtime image; check it through `/healthz` and `/readyz` from the host.
- The local database credentials are for development only. Never reuse them in a deployed environment.
