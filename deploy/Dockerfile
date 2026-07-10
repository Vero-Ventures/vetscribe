# syntax=docker/dockerfile:1

# ---- Stage 1: build the static Go binary (frontend embedded from source) ----
# The HTML template and vanilla JS/CSS assets live in internal/api and are embedded
# via embed.FS at compile time — there is no separate frontend build step.
FROM golang:1.26-alpine AS backend
WORKDIR /src
# chromedp is a test-only (e2e-tagged) dependency, so go.sum exists; the glob keeps
# the COPY valid regardless.
COPY go.mod go.su[m] ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /vetscribe ./cmd/vetscribe

# ---- Stage 2: minimal runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /vetscribe /vetscribe
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/vetscribe"]
