# List all commands
list:
    just --list --unsorted

########################################################
# Go

# Build the binary
build:
    go build -o netbird-coredns ./cmd/netbird-coredns

# Run the service locally (requires NetBird and CoreDNS installed)
run:
    go run ./cmd/netbird-coredns

########################################################
# Docker

# Build Docker image
docker-build:
    docker build -t nb-dns:dev . -f docker/Dockerfile

_docker-compose *args:
    docker compose -f docker/compose.yml {{ args }}

# Start services with Docker Compose
start:
    just docker-build
    just _docker-compose up -d

# Stop services
stop:
    just _docker-compose down

# View logs
logs:
    just _docker-compose logs -f

# View processes
ps:
    just _docker-compose ps

# Stop and remove everything including volumes
clean:
    just _docker-compose down -v

# Format code
fmt:
    go fmt ./...

# Tidy dependencies
tidy:
    go mod tidy
