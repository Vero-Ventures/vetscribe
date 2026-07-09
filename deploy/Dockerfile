# syntax=docker/dockerfile:1

# ---- Stage 1: build the frontend into the Go embed directory ----
FROM node:22-alpine AS frontend
RUN corepack enable
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile
COPY web ./
# vite creates ../internal/api/dist (relative to /src/web) with the built assets.
RUN pnpm build

# ---- Stage 2: build the static Go binary with the embedded frontend ----
FROM golang:1.26-alpine AS backend
WORKDIR /src
# go.sum is absent while the backend is stdlib-only; the glob keeps the COPY valid.
COPY go.mod go.su[m] ./
RUN go mod download
COPY . .
COPY --from=frontend /src/internal/api/dist ./internal/api/dist
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /vetscribe ./cmd/vetscribe

# ---- Stage 3: minimal runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /vetscribe /vetscribe
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/vetscribe"]
