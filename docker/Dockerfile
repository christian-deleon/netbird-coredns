# Stage: Clone CoreDNS (cache this separately - only invalidates if CoreDNS version changes)
FROM golang:1.25-alpine AS coredns-source
RUN apk add --no-cache git
WORKDIR /coredns
RUN git clone --depth 1 --branch v1.13.1 https://github.com/coredns/coredns.git . && \
    git checkout v1.13.1

# Stage: Build netbird-coredns binary
FROM golang:1.25-alpine AS netbird-builder
WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies with cache
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY plugin.go ./

# Build the main binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o netbird-coredns ./cmd/netbird-coredns

# Stage: Build CoreDNS with plugin
FROM golang:1.25-alpine AS coredns-builder
WORKDIR /app

# Copy the netbird-coredns binary from previous stage
COPY --from=netbird-builder /app/netbird-coredns /app/netbird-coredns

# Copy CoreDNS source (cached layer)
COPY --from=coredns-source /coredns /coredns

# Copy plugin source (only this invalidates when plugin code changes)
# Need go.mod and go.sum for the replace directive to work
COPY go.mod go.sum /app/
COPY plugin.go /app/plugin.go
COPY internal/ /app/internal/
COPY pkg/ /app/pkg/
COPY cmd/ /app/cmd/

WORKDIR /coredns

# Modify plugin.cfg and build CoreDNS
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    sed -i '/^forward:/i netbird:netbird-coredns' plugin.cfg && \
    go mod edit -require=netbird-coredns@v0.0.0 && \
    go mod edit -replace=netbird-coredns=/app && \
    go generate && \
    go mod tidy && \
    go build -o coredns .

# Runtime stage
FROM alpine:3.22

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    iptables \
    ip6tables \
    curl \
    jq && \
    mkdir -p \
    /state \
    /etc/nb-dns/records

# Install NetBird
# See latest version at: https://github.com/netbirdio/netbird/releases/latest
ENV NETBIRD_VERSION=0.59.12
RUN ARCH=$(uname -m) && \
    case $ARCH in \
        x86_64) ARCH_NAME=amd64 ;; \
        aarch64) ARCH_NAME=arm64 ;; \
        *) echo "Unsupported architecture: $ARCH" && exit 1 ;; \
    esac && \
    curl -fsSL "https://github.com/netbirdio/netbird/releases/download/v${NETBIRD_VERSION}/netbird_${NETBIRD_VERSION}_linux_${ARCH_NAME}.tar.gz" | \
    tar -xzf - -C /usr/local/bin/ netbird

# Copy binaries from builder stages
COPY --from=netbird-builder /app/netbird-coredns /usr/local/bin/
COPY --from=coredns-builder /coredns/coredns /usr/local/bin/

# Set working directory
WORKDIR /

# Expose DNS and API ports
# Note: The actual DNS port is configurable via NBDNS_DNS_PORT (default: 5053)
EXPOSE 5053/udp 5053/tcp 8080/tcp

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/netbird-coredns"]
