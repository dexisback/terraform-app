# Build stage: compile a static Go binary.
FROM golang:1.23-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/todo-api ./cmd/api

# Runtime stage: no Go compiler or source tree in the final image.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/todo-api /todo-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/todo-api"]
