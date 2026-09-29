# gopherd

**Production-oriented Go HTTP API for tasks, built as a systems/backend engineering project.**

`gopherd` is a real, deployable REST-style service rather than a Hello World HTTP-server exercise. It demonstrates the complete path from a Go process listening on a TCP port to an authenticated API backed by PostgreSQL, with migrations, structured logs, request IDs, rate limiting, health checks, metrics, graceful shutdown, tests, Docker, Caddy, and CI.

> **Status:** production-oriented reference implementation. Before exposing it to the public Internet, review the deployment checklist, set real secrets, configure TLS/DNS, restrict CORS, configure backups, and perform your own security review.

## What it does

- User registration and login
- Password hashing with bcrypt
- Opaque bearer sessions stored as SHA-256 token hashes
- User-scoped task CRUD
- Pagination and completion filtering
- PostgreSQL persistence
- Embedded SQL migrations
- Request IDs (`X-Request-ID`)
- JSON structured application logs
- Health and readiness endpoints
- Simple Prometheus-compatible metrics endpoint
- In-process token-bucket rate limiting
- Configurable CORS
- Request/body/time limits
- Graceful SIGINT/SIGTERM shutdown
- Docker multi-stage image
- Docker Compose with PostgreSQL
- Optional Caddy reverse proxy for HTTPS
- GitHub Actions CI
- OpenAPI 3.0 API description

## Architecture

```text
                         INTERNET
                            │
                            ▼
                       DNS / TLS
                            │
                            ▼
                    Caddy / Reverse Proxy
                            │
                            ▼
                  ┌─────────────────────┐
                  │       gopherd       │
                  │                     │
                  │ Request ID          │
                  │ Recovery            │
                  │ Logging             │
                  │ CORS / Rate limit  │
                  │        ↓            │
                  │      Router         │
                  │        ↓            │
                  │     Handlers        │
                  │        ↓            │
                  │     Services        │
                  │        ↓            │
                  │   Repositories      │
                  └─────────┬───────────┘
                            │
                            ▼
                       PostgreSQL
                            │
                            ▼
                           Disk
```

### Request flow

```text
HTTP request
    ↓
request ID
    ↓
panic recovery
    ↓
structured logging
    ↓
CORS / rate limiting
    ↓
Chi router
    ↓
auth middleware (protected routes)
    ↓
handler
    ↓
service / validation
    ↓
repository
    ↓
PostgreSQL
    ↓
JSON response
```

## Repository layout

```text
gopherd/
├── cmd/gopherd/main.go       # application entrypoint
├── internal/
│   ├── auth/                 # reserved for future auth helpers
│   ├── config/               # environment configuration
│   ├── database/              # PostgreSQL pool + embedded migrations
│   ├── handler/               # HTTP handlers
│   ├── middleware/            # request ID, recovery, logging, rate limit, CORS, metrics
│   ├── model/                 # API/domain models
│   ├── repository/             # database access
│   ├── service/                # business rules
│   └── server/                 # HTTP routing
├── migrations/                 # human-readable migration copies
├── docs/openapi.yaml           # API contract
├── deploy/Caddyfile            # production reverse-proxy example
├── .github/workflows/ci.yml    # CI
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .env.example
├── .gitignore
└── LICENSE
```

## API

Base URL: `http://localhost:8080`

### Public endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/health/live` | Process is alive |
| GET | `/health/ready` | Database is reachable |
| GET | `/metrics` | Basic Prometheus-compatible metrics |
| POST | `/api/v1/auth/register` | Create user + session |
| POST | `/api/v1/auth/login` | Login + session |

### Protected endpoints

Send:

```http
Authorization: Bearer <token>
```

| Method | Endpoint | Purpose |
|---|---|---|
| POST | `/api/v1/auth/logout` | Revoke current session |
| GET | `/api/v1/me` | Current user |
| GET | `/api/v1/tasks` | List tasks |
| POST | `/api/v1/tasks` | Create task |
| GET | `/api/v1/tasks/{id}` | Get task |
| PUT | `/api/v1/tasks/{id}` | Replace task |
| DELETE | `/api/v1/tasks/{id}` | Delete task |

See [`docs/openapi.yaml`](docs/openapi.yaml) for the complete request/response contract.

## 1. Run locally with Docker Compose

This is the fastest complete development path.

### Prerequisites

- Git
- Docker Engine
- Docker Compose v2

Clone:

```bash
git clone https://github.com/YOUR_USERNAME/gopherd.git
cd gopherd
```

Create local configuration:

```bash
cp .env.example .env
```

Start PostgreSQL and gopherd:

```bash
docker compose up --build -d db app
```

Check:

```bash
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
```

Expected live response:

```json
{"status":"ok"}
```

View logs:

```bash
docker compose logs -f app
```

Stop:

```bash
docker compose down
```

Data survives a normal `docker compose down` because PostgreSQL uses the named `postgres_data` volume. Remove it only when you intentionally want to destroy the local database:

```bash
docker compose down -v
```

## 2. Run the Go application directly

You need PostgreSQL running and Go 1.23+.

Create a database/user or use a local PostgreSQL installation, then set:

```bash
export DATABASE_URL='postgres://gopherd:gopherd@localhost:5432/gopherd?sslmode=disable'
```

Run:

```bash
go mod download
go run ./cmd/gopherd
```

Build:

```bash
make build
```

Binary:

```text
bin/gopherd
```

## 3. Exercise the API

### Register

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"a-strong-local-password"}'
```

The response contains a bearer token. Export it:

```bash
export TOKEN='PASTE_TOKEN_HERE'
```

### Current user

```bash
curl -sS http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer $TOKEN"
```

### Create a task

```bash
curl -sS -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Study HTTP","description":"Understand TCP, sockets and HTTP request flow","completed":false}'
```

### List tasks

```bash
curl -sS 'http://localhost:8080/api/v1/tasks?limit=20&offset=0' \
  -H "Authorization: Bearer $TOKEN"
```

Filter:

```bash
curl -sS 'http://localhost:8080/api/v1/tasks?completed=false' \
  -H "Authorization: Bearer $TOKEN"
```

### Get one

```bash
curl -sS http://localhost:8080/api/v1/tasks/TASK_ID \
  -H "Authorization: Bearer $TOKEN"
```

### Update

```bash
curl -sS -X PUT http://localhost:8080/api/v1/tasks/TASK_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Study HTTP deeply","description":"TCP + HTTP + Go net/http","completed":true}'
```

### Delete

```bash
curl -i -X DELETE http://localhost:8080/api/v1/tasks/TASK_ID \
  -H "Authorization: Bearer $TOKEN"
```

### Logout

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```

## 4. Observe the operating system

While gopherd is running:

```bash
ss -ltnp
```

You should see a listener on port `8080`.

Find your LAN address:

```bash
ip addr
```

If you bind the service to a reachable interface and your firewall allows it, another device on the same network can call:

```text
http://YOUR_LAN_IP:8080/health/live
```

This is the physical experiment behind the project:

```text
Go process
    ↓
TCP socket
    ↓
port 8080
    ↓
Linux network stack
    ↓
Wi-Fi/Ethernet
    ↓
router/switch
    ↓
client device
```

## 5. Production deployment: recommended path

The intended first production deployment is:

```text
Domain
  ↓
DNS A/AAAA record
  ↓
VPS with Ubuntu/Debian
  ↓
Caddy
  ↓
gopherd container
  ↓
PostgreSQL container
  ↓
Persistent volume
```

### VPS requirements

For a first deployment, a small Linux VPS is sufficient. Choose a provider/region based on your own cost, latency, support, and data-location requirements.

Install:

- Docker Engine + Compose plugin
- Git
- Firewall tooling

Create a non-root deployment user and use SSH keys rather than password authentication where practical.

### Firewall

Expose only what is required. A typical public service needs:

```text
22/tcp    SSH
80/tcp    HTTP (ACME challenge + redirect)
443/tcp   HTTPS
```

Do not expose PostgreSQL's `5432` to the public Internet for this architecture.

### DNS

Point your chosen API hostname at the VPS:

```text
api.example.com  →  VPS_PUBLIC_IP
```

Verify:

```bash
dig api.example.com
```

### Configure production secrets

Create `.env` on the server. At minimum change:

```env
POSTGRES_PASSWORD=<long-random-secret>
ENVIRONMENT=production
CORS_ORIGINS=https://your-frontend.example
LOG_LEVEL=info
```

Do not commit `.env`.

Generate a password with a password manager or a cryptographically secure generator. Never reuse your GitHub password, SSH password, or another service credential.

### Start the application

```bash
docker compose up --build -d db app
```

### Enable Caddy

Edit:

```text
deploy/Caddyfile
```

Replace:

```text
api.example.com
```

with your actual hostname.

Then:

```bash
docker compose --profile proxy up -d caddy
```

Caddy will obtain and renew certificates when DNS and ports 80/443 are correctly configured.

Verify:

```bash
curl https://api.example.com/health/live
curl https://api.example.com/health/ready
```

## 6. Production checklist

Before calling a deployment production-ready, verify:

- [ ] Real DNS configured
- [ ] HTTPS works
- [ ] PostgreSQL is not publicly exposed
- [ ] Strong database password set
- [ ] `.env` is not in Git
- [ ] CORS contains only intended origins
- [ ] Firewall allows only required ports
- [ ] SSH key authentication configured
- [ ] Root SSH login disabled where appropriate
- [ ] Database backups configured and tested
- [ ] Restore procedure documented
- [ ] Logs are retained appropriately
- [ ] Health checks monitored
- [ ] Disk space monitored
- [ ] Resource limits reviewed
- [ ] Dependency updates are reviewed regularly
- [ ] Security review performed before public launch
- [ ] Rate limiting is appropriate for the expected traffic
- [ ] If horizontally scaling, move rate limiting/session strategy to infrastructure designed for multi-instance operation

## 7. Important architectural limitations

This project deliberately has a sweet spot: it is substantially more than a tutorial but still understandable as a single-developer deployment.

### In-process rate limiter

The limiter is per application process. It is useful for a single-instance deployment. If you scale to multiple gopherd instances, each instance has its own bucket. At that point use a gateway/API gateway or shared rate-limit store.

### Session storage

Sessions are opaque random tokens. Only SHA-256 hashes are stored in PostgreSQL. The raw token is returned once at login/register. Treat it like a password: anyone holding it can authenticate until it expires or is revoked.

### PostgreSQL

The application owns migrations but does not include automated database backups. Backups are an infrastructure responsibility and must be configured for the actual production provider.

### Metrics

The built-in endpoint is intentionally small. For larger deployments, add a proper Prometheus client, dashboards, alerting, and tracing.

## 8. Testing and development workflow

Run formatting:

```bash
make fmt
```

Run tests:

```bash
make test
```

Run vet:

```bash
make vet
```

Run the race detector directly:

```bash
go test -race ./...
```

Build:

```bash
make build
```

Build and start containers:

```bash
make docker-up
```

Stop containers:

```bash
make docker-down
```

## 9. GitHub workflow

Recommended repository workflow:

```bash
git init
git add .
git commit -m "initial production-oriented Go HTTP service"
```

Create a GitHub repository, then:

```bash
git branch -M main
git remote add origin git@github.com:YOUR_USERNAME/gopherd.git
git push -u origin main
```

Suggested commits as the project evolves:

```text
initial HTTP service
add PostgreSQL persistence
add authentication
add task CRUD
add middleware and request IDs
add health and metrics endpoints
add Docker deployment
add Caddy deployment
add CI
```

## 10. CI

GitHub Actions runs formatting checks, tests, race detection, vet, and a production build. See `.github/workflows/ci.yml`.

The CI goal is simple: a commit should not be considered ready merely because it works on the developer's laptop.

## 11. Learning map

This project is intentionally a learning system as well as software.

### Layer 1 — Physical machine

CPU, RAM, storage, network interface.

### Layer 2 — Linux

Processes, file descriptors, sockets, routing, firewall, systemd.

### Layer 3 — Networking

IP, TCP, ports, DNS, TLS.

### Layer 4 — HTTP

Methods, headers, status codes, request/response bodies.

### Layer 5 — Go

`net/http`, structs, interfaces, errors, context, goroutines, tests.

### Layer 6 — Application architecture

Router → handler → service → repository.

### Layer 7 — Persistence

PostgreSQL, SQL, indexes, transactions, migrations.

### Layer 8 — Operations

Docker, Caddy, DNS, systemd, logs, metrics, backups.

The useful mental model is:

```text
Physical machine
      ↓
Operating system
      ↓
Network interface
      ↓
IP
      ↓
TCP
      ↓
Socket
      ↓
HTTP
      ↓
Go net/http
      ↓
Router
      ↓
Application
      ↓
Database
      ↓
Deployment
      ↓
Observability
```

## 12. Future extensions

Once the current system is understood and tested, possible next steps include:

- Refresh-token rotation
- Email verification
- Password reset flow
- Role-based authorization
- Full Prometheus instrumentation
- OpenTelemetry tracing
- Redis-backed rate limiting
- Background jobs
- Task labels and due dates
- Full-text search
- WebSocket/SSE updates
- Object storage
- Multi-instance deployment
- Kubernetes only when operational complexity actually justifies it

Do not add these just to make the repository larger. Add them when they solve a real requirement.

## License

MIT. See [`LICENSE`](LICENSE).
