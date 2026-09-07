# ==============================================================================
# Build stage
# ==============================================================================
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree
COPY . .

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
# Runtime image (Restricted Pod Security Standard compliant)
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

EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/bff"]
