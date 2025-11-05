# List all commands
list:
    just --list --unsorted

# Build the binary
build:
    go build -o netbird-coredns ./cmd/netbird-coredns

# Run the service locally (requires NetBird and CoreDNS installed)
run:
    go run ./cmd/netbird-coredns

# Build Docker image
docker-build:
    docker compose -f docker/compose.yml build

# Start services with Docker Compose
start:
    docker compose -f docker/compose.yml up -d --build

# Stop services
stop:
    docker compose -f docker/compose.yml down

# View logs
logs:
    docker compose -f docker/compose.yml logs -f

# View processes
ps:
    docker compose -f docker/compose.yml ps

# Stop and remove everything including volumes
clean:
    docker compose -f docker/compose.yml down -v

# Format code
fmt:
    go fmt ./...

# Tidy dependencies
tidy:
    go mod tidy
