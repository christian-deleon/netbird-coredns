# NetBird CoreDNS Integration

A CoreDNS plugin for managing custom DNS records via HTTP API. This service is containerized and works with Kubernetes.

## Features

- **Custom DNS Records API**: Manage A and CNAME records via HTTP API
- **Forward to External DNS**: Forward unresolved queries to external DNS servers (e.g., Cloudflare, Google DNS)
- **Docker Support**: Containerized deployment with Docker Compose
- **Kubernetes Support**: Designed to run in Kubernetes environments
- **Health Endpoint**: `/health` endpoint for monitoring and K8s compatibility
- **High Availability Compatible**: Designed for multi-instance deployments

## Quick Start

### Prerequisites

- Docker and Docker Compose (for local testing)
- A domain for DNS resolution

### Installation

1. **Clone the repository**:

```bash
git clone https://github.com/christian-deleon/netbird-coredns.git
cd netbird-coredns/docker
```

2. **Create environment file**:

```bash
cp example.env .env
```

3. **Edit `.env` with your configuration**:

```bash
# Required settings
NBDNS_DOMAINS=mydomain.com

# Optional settings
NBDNS_FORWARD_TO=8.8.8.8
NBDNS_API_PORT=8080
NBDNS_DNS_PORT=5053
NBDNS_REFRESH_INTERVAL=30
```

4. **Start the service**:

```bash
docker compose up -d
```

Or using the Justfile:

```bash
just start
```

5. **Check logs**:

```bash
docker compose logs -f
```

Or:

```bash
just logs
```

## Configuration

### Environment Variables

All environment variables are prefixed with `NBDNS_`:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NBDNS_DOMAINS` | Yes | - | Comma-separated domains for DNS resolution |
| `NBDNS_FORWARD_TO` | No | `8.8.8.8` | Forward server for unresolved queries |
| `NBDNS_DNS_PORT` | No | `5053` | DNS server port (use different port if 53 is in use) |
| `NBDNS_API_PORT` | No | `8080` | API server port |
| `NBDNS_REFRESH_INTERVAL` | No | `30` | Refresh interval in seconds |
| `NBDNS_RECORDS_FILE` | No | `/etc/nb-dns/records/records.json` | Path to DNS records file |
| `NBDNS_LOG_LEVEL` | No | `info` | Log level for the entire service (debug, info, warn, error) |

### Domain Configuration

The `NBDNS_DOMAINS` environment variable specifies which domains this DNS server will handle. The configured domains determine which DNS queries will be processed by this service. Queries for other domains will be forwarded to the external DNS server specified in `NBDNS_FORWARD_TO`.

**Note**: The domain configured in `NBDNS_DOMAINS` is independent of any NetBird peer configuration. If you're using NetBird, the peer domain (determined by your NetBird Management server - whether official or self-hosted) can be different from `NBDNS_DOMAINS`.

## DNS Records API

The service provides an HTTP API for managing custom DNS records.

### API Endpoints

#### Health Check

```bash
GET /health
```

Returns `200 OK` when the service is healthy.

**Example**:

```bash
curl http://localhost:8080/health
```

**Response**:

```json
{
  "status": "ok"
}
```

#### List All Records

```bash
GET /api/v1/records
```

Returns all DNS records organized by domain.

**Example**:

```bash
curl http://localhost:8080/api/v1/records
```

**Response**:

```json
{
  "example.com": {
    "web": {
      "name": "web",
      "domain": "example.com",
      "type": "A",
      "value": "192.168.1.100",
      "ttl": 60
    },
    "api": {
      "name": "api",
      "domain": "example.com",
      "type": "CNAME",
      "value": "web.example.com",
      "ttl": 60
    }
  }
}
```

#### Create a Record

```bash
POST /api/v1/records
Content-Type: application/json

{
  "name": "web",
  "domain": "example.com",
  "type": "A",
  "value": "192.168.1.100"
}
```

**Supported record types**: `A`, `CNAME`

**Example**:

```bash
# Create A record
curl -X POST http://localhost:8080/api/v1/records \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web",
    "domain": "example.com",
    "type": "A",
    "value": "192.168.1.100"
  }'

# Create CNAME record
curl -X POST http://localhost:8080/api/v1/records \
  -H "Content-Type: application/json" \
  -d '{
    "name": "www",
    "domain": "example.com",
    "type": "CNAME",
    "value": "web.example.com"
  }'
```

#### Update a Record

```bash
PUT /api/v1/records/{domain}/{name}
Content-Type: application/json

{
  "type": "A",
  "value": "192.168.1.101"
}
```

**Example**:

```bash
curl -X PUT http://localhost:8080/api/v1/records/example.com/web \
  -H "Content-Type: application/json" \
  -d '{
    "type": "A",
    "value": "192.168.1.101"
  }'
```

#### Delete a Record

```bash
DELETE /api/v1/records/{domain}/{name}
```

**Example**:

```bash
curl -X DELETE http://localhost:8080/api/v1/records/example.com/web
```

## Usage

### DNS Resolution

Once the service is running, you can resolve DNS queries using the configured DNS port (default 5053).

**Testing with localhost**:

```bash
# Resolve custom A record
dig +short web.example.com @localhost -p 5053

# Resolve CNAME record
dig +short www.example.com @localhost -p 5053 CNAME
```

**Testing with DNS hostname**:

If the service is accessible via a hostname (e.g., in a Kubernetes cluster or via DNS), you can query it directly:

```bash
# Query using the DNS server hostname
dig +short web.example.com @dns-server.example.com -p 5053
```

### Testing

This service can be tested locally without any NetBird peer connection. It operates independently and only requires:

1. A configured domain in `NBDNS_DOMAINS`
2. Access to the DNS port (default 5053)
3. Access to the API port (default 8080) for managing records

You can test the service locally using `localhost` or via a DNS hostname if the service is exposed in your network or Kubernetes cluster.

## High Availability

### Multiple Instances

Deploy multiple instances for high availability:

1. Each instance serves the same DNS records
2. Load balance DNS queries across instances
3. Use a shared persistent volume for the records file in Kubernetes

## Docker Compose Commands

The project includes a `Justfile` with convenient commands. See `just list` for all available commands.

**Using Docker Compose directly**:

```bash
# Start services
docker compose up -d

# View logs
docker compose logs -f

# Stop services
docker compose down

# Rebuild and restart
docker compose up -d --build

# Remove everything including volumes
docker compose down -v
```

**Using Justfile**:

```bash
# Start services
just start

# View logs
just logs

# Stop services
just stop

# View processes
just ps

# Clean everything
just clean

# Build binary
just build

# Run locally
just run
```

## Troubleshooting

### DNS Queries Not Resolving

1. Verify domain configuration matches
2. Test with the correct port: `dig +short @localhost -p 5053 hostname.domain.com`
3. Check records exist: `curl http://localhost:8080/api/v1/records`

### DNS Port Conflicts

If port 53 is already in use:

1. Change the DNS port in `.env`: `NBDNS_DNS_PORT=5053`
2. Update the port mapping in `docker/compose.yml` to match
3. Test with: `dig +short @localhost -p 5053 hostname.domain.com`

### API Not Responding

1. Check if API server is running: `curl http://localhost:8080/health`
2. Verify `NBDNS_API_PORT` is not in use
3. Check firewall rules for port access

### Custom Records Not Persisting

1. Verify records file path: `NBDNS_RECORDS_FILE`
2. Check volume mount in `compose.yml` or Kubernetes PersistentVolume
3. Ensure write permissions on records directory

## Architecture

### Components

1. **CoreDNS**: DNS server with custom plugin
2. **API Server**: HTTP API for managing custom DNS records
3. **Process Manager**: Orchestrates all components and handles shutdown

### DNS Resolution Priority

1. **Custom CNAME records** (from API)
2. **Custom A records** (from API)
3. **Forward to external DNS** (configured forward server)

### Data Flow

```text
DNS Query → CoreDNS → Custom Plugin → Check Custom Records
                                     ↓
                               Forward to External DNS
```

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/christian-deleon/netbird-coredns.git
cd netbird-coredns

# Build binary
go build -o netbird-coredns ./cmd/netbird-coredns

# Or use Justfile
just build
```

### Running Locally

```bash
# Set required environment variables
export NBDNS_DOMAINS=mydomain.com

# Run the service
./netbird-coredns

# Or use Justfile
just run
```

### Project Structure

```text
netbird-coredns/
├── cmd/netbird-coredns/   # Main application
├── internal/
│   ├── api/               # HTTP API server
│   ├── config/            # Configuration management
│   ├── plugin/            # CoreDNS plugin
│   ├── process/           # Process management
│   └── template/          # Corefile generation
├── pkg/
│   └── dns/               # DNS record types
├── docker/                # Docker deployment files
└── Justfile               # Build and development commands
```

## Kubernetes

This service is containerized and works with Kubernetes. It includes:

- **Health endpoint**: `/health` for liveness/readiness probes
- **Graceful shutdown**: Handles SIGTERM properly
- **Container best practices**: Proper signal handling and resource management

## Roadmap

### Future Features

- Kubernetes manifests and Helm charts
- Authentication for API endpoints
- Metrics and Prometheus integration
- Comprehensive test suite
- Web UI for managing DNS records

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Author

Created by [Christian De Leon](https://github.com/christian-deleon)

## Related Projects

- [CoreDNS](https://github.com/coredns/coredns) - DNS server in Go
- [tailscale-coredns](https://github.com/christian-deleon/tailscale-coredns) - Similar project for Tailscale

## Support

For issues and questions:

- [GitHub Issues](https://github.com/christian-deleon/netbird-coredns/issues)
- [CoreDNS Documentation](https://coredns.io/manual/toc/)
