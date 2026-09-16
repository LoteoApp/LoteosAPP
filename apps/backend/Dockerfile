# syntax=docker/dockerfile:1
#
# Production image for the Go API. Build context is the repo root (not
# apps/backend) so the `migrate` stage can also copy migrations/, which lives
# outside apps/backend.
#
# Dokploy: Dockerfile Path = apps/backend/Dockerfile, Docker Context Path = .
# Default build stage (no "Docker Build Stage" set) is `server`. Set it to
# `migrate` only to build the one-off migration image; never run it as a
# long-lived service (see docs/database.md — migrations are a manual step,
# not automatic on backend startup).

FROM golang:1.26-alpine AS builder
WORKDIR /src

COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download

COPY apps/backend/. .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot AS migrate
WORKDIR /app
COPY --from=builder /out/migrate ./migrate
COPY migrations /workspace/migrations
ENV MIGRATIONS_DIR=/workspace/migrations
ENTRYPOINT ["/app/migrate"]

FROM gcr.io/distroless/static-debian12:nonroot AS server
WORKDIR /app
COPY --from=builder /out/server ./server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
