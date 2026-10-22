# Build stage
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/todo-api ./cmd/api

# Runtime stage
FROM alpine:3.22
WORKDIR /
RUN apk add --no-cache ca-certificates \
    && adduser -D -H -s /sbin/nologin nonroot
COPY --from=build /out/todo-api /todo-api
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/todo-api"]
