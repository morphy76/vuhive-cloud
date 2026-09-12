# ==============================================================================
# Stage 1: Frontend build (React 19 PWA with Vite & Tailwind CSS v4)
# ==============================================================================
FROM node:22-alpine AS frontend-builder

WORKDIR /src/web

# Enable pnpm package manager via corepack
RUN corepack enable && corepack prepare pnpm@latest --activate

# Cache frontend dependencies
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml* ./
RUN pnpm install --frozen-lockfile

# Copy documentation referenced by frontend (cookbook)
COPY docs/ /src/docs/

# Copy frontend source code and configuration files
COPY web/ .

# Build production static assets into /src/web/dist
RUN pnpm build

# ==============================================================================
# Stage 2: Go binary compilation (BFF service embedding frontend assets)
# ==============================================================================
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache Go module dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source tree
COPY . .

# Copy compiled frontend assets from Stage 1 into web/dist
COPY --from=frontend-builder /src/web/dist ./web/dist

# Build static binary for BFF service
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown
ARG MODULE=github.com/morphy76/vuhive-cloud

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w \
      -X '${MODULE}/internal/version.Version=${VERSION}' \
      -X '${MODULE}/internal/version.Commit=${COMMIT}' \
      -X '${MODULE}/internal/version.BuildTime=${BUILD_TIME}'" \
    -o /bin/bff ./cmd/bff

# ==============================================================================
# Stage 3: Runtime image (Restricted Pod Security Standard compliant)
# ==============================================================================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Create non-root system user and group (UID/GID 10001)
RUN addgroup -g 10001 vuhive && \
    adduser -u 10001 -G vuhive -s /bin/sh -D vuhive

# Copy statically compiled binary
COPY --from=builder /bin/bff /usr/local/bin/bff

RUN chmod 0755 /usr/local/bin/bff

USER 10001:10001

ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/bff"]
