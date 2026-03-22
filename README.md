# Todo App v2

Simple todo app — Go + PostgreSQL + Docker + Kubernetes.

━━━━━━━━━━━━━━━━━━━━
## Requirements
━━━━━━━━━━━━━━━━━━━━

**Tech stack:** Go 1.24, Gin, GORM, PostgreSQL 16, golang-migrate, Docker Compose, Kind, Helm 3, NGINX Ingress, GitHub Actions

**REST API** documented with OpenAPI 3.0 — Swagger UI served at `/docs`.

| Environment | Swagger URL |
|---|---|
| Docker Compose | http://localhost:8080/docs |
| Kubernetes | http://todo.local/docs |

━━━━━━━━━━━━━━━━━━━━
## Data Layer
━━━━━━━━━━━━━━━━━━━━

**Database:** PostgreSQL 16

**Why PostgreSQL?** Structured data with filters (completed, dueDate range, text search) and typed fields. Relational constraints, ACID, `text[]` arrays for tags out of the box. `ILIKE` covers search without extra services. For a todo app with predictable schema and filters, SQL fits better than MongoDB — no need to duplicate validation in app code.

**ORM:** GORM

**Why GORM?** Soft delete built in, connection pool management, less boilerplate for CRUD. Downside — implicit queries make it harder to spot N+1. Considered sqlc (type-safe SQL, no reflection), but it needs more code for basic CRUD. For a small app this tradeoff is fine — can always drop to raw SQL for specific queries.

━━━━━━━━━━━━━━━━━━━━
## Migrations
━━━━━━━━━━━━━━━━━━━━

**Tool:** golang-migrate — versioned SQL files (up/down) in `migrations/`.

**Why not GORM AutoMigrate?** No down-migrations, no version control on schema, can silently drop data on column renames. golang-migrate files can be reviewed in PRs, rolled back, and replayed on any environment.

━━━━━━━━━━━━━━━━━━━━
## Minimal Feature Set
━━━━━━━━━━━━━━━━━━━━

**CRUD Todos** with fields: title, completed, dueDate, tags (optional).

**Endpoints:**

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/todos` | List todos with filters |
| `POST` | `/api/v1/todos` | Create a todo |
| `GET` | `/api/v1/todos/:id` | Get one todo |
| `PUT` | `/api/v1/todos/:id` | Update a todo |
| `DELETE` | `/api/v1/todos/:id` | Soft delete |
| `POST` | `/api/v1/todos/clear-completed` | Delete all completed |

**Filters:** `?completed=true`, `?due_before=2025-01-01T00:00:00Z`, `?due_after=...`, `?search=grocery`

**Health endpoints:**

| Path | Type |
|---|---|
| `/healthz` | Liveness probe |
| `/readyz` | Readiness probe + DB ping |

**Tests:** service-level unit tests + API integration tests against real PostgreSQL (with `-race` flag).

**Frontend:** static HTML page served from the API — list / add / complete todos.

━━━━━━━━━━━━━━━━━━━━
## Deliverables
━━━━━━━━━━━━━━━━━━━━

- `openapi.yaml` — OpenAPI 3.0 spec, validated in CI
- Source code — this repository
- Architecture & tradeoffs — covered in Data Layer, Migrations, and Ingress sections of this README

━━━━━━━━━━━━━━━━━━━━
## Local Dev Environment
━━━━━━━━━━━━━━━━━━━━

Goal: one-command local run using Docker Compose.

### Prerequisites

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker
```

Verify: `docker version`

macOS: https://docs.docker.com/desktop/mac/install/

For Kubernetes — install Docker first, then:

```bash
sudo apt install make   # if not installed
make install-tools      # installs kind, kubectl, helm, golang-migrate
```

### How to run docker compose up

```bash
docker compose up --build -d
```

| Service | Description |
|---|---|
| **db** | PostgreSQL 16, port 5432 |
| **migrate** | Runs migrations and exits |
| **api** | Go server, port 8080 |

The API won't start until migrations finish. Check status: `docker compose ps`

### How to run migrations

**Docker Compose** — automatic. The `migrate` service applies all pending migrations before the API starts.

**Kubernetes:**

```bash
kubectl port-forward svc/todo-postgresql 5433:5432 &
migrate -path=./migrations -database="postgres://postgres:postgres@localhost:5433/todo_db?sslmode=disable" up
```

Or: `make migrate`

### How to seed data

Test data in `seeds/seed.sql` — 3 sample todos. Idempotent, inserts only if the table is empty. Not automatic — DB starts clean unless you explicitly seed.

| Environment | Command |
|---|---|
| Docker Compose | `docker compose --profile seed up seed` |
| Kubernetes | `make seed` |

### How to open the app locally

| Environment | URL |
|---|---|
| Docker Compose | http://localhost:8080 |
| Kubernetes | http://todo.local |

`make deploy-kind` adds `todo.local` to `/etc/hosts` automatically.

### How to open Swagger UI

| Environment | URL |
|---|---|
| Docker Compose | http://localhost:8080/docs |
| Kubernetes | http://todo.local/docs |

### Common troubleshooting

**CrashLoopBackOff** — usually PostgreSQL isn't ready yet. Check `kubectl logs deploy/todo` and `kubectl logs sts/todo-postgresql`. Give it a minute. If `password authentication failed`: `make delete-cluster && make deploy-kind`

**ErrImagePull / ImagePullBackOff** — local registry not running: `make create-registry`

**"connection refused" on docker push** — same fix: `make create-registry`

**Port already in use** — `lsof -i :8080` (Compose) or `lsof -i :80` (Kind). Kill the process or change port in `.env` / `kind-config.yaml`.

**Running both Compose and Kind** — ports don't overlap (8080 vs 80/443), but two PostgreSQL instances can confuse. Stop Compose first: `docker compose down`

━━━━━━━━━━━━━━━━━━━━
## Local Kubernetes Cluster
━━━━━━━━━━━━━━━━━━━━

Kind cluster: 1 control-plane + 1 worker node.

```bash
make deploy-kind
```

This single command:

1. Starts local Docker Registry on `localhost:5000`
2. Creates Kind cluster
3. Connects registry to Kind network
4. Installs NGINX Ingress Controller
5. Builds and pushes Docker image
6. Deploys PostgreSQL (bitnami/postgresql) and the app via Helm
7. Runs migrations
8. Adds `todo.local` to `/etc/hosts`

After that: http://todo.local

━━━━━━━━━━━━━━━━━━━━
## Build Pipeline
━━━━━━━━━━━━━━━━━━━━

CI runs on GitHub Actions (`.github/workflows/ci.yml`), triggers on push/PR to `main`.

1. **Lint** — `golangci-lint` (errcheck, staticcheck, gosimple, unused)
2. **OpenAPI validation** — `@redocly/cli lint openapi.yaml`
3. **Helm lint** — `helm lint` with bitnami dependency
4. **Tests** — unit + integration tests against real PostgreSQL, with `-race` flag
5. **Docker build & push** — multi-stage build, pushes to GHCR on `main`

━━━━━━━━━━━━━━━━━━━━
## Helm Packaging
━━━━━━━━━━━━━━━━━━━━

Chart lives in `deploy/helm/todo-app/`.

**Templates:**

- `deployment.yaml` — API Deployment with liveness/readiness probes
- `service.yaml` — ClusterIP (port 80 → 8080)
- `ingress.yaml` — NGINX Ingress, host: `todo.local`
- `configmap.yaml` — DB connection settings
- `secret.yaml` — DB credentials

**Dependency:** bitnami/postgresql 18.5.11 (persistence disabled for dev)

**Key values:**

```yaml
image:
  repository: localhost:5000/todo-api
  tag: latest
  pullPolicy: IfNotPresent

replicaCount: 1

ingress:
  enabled: true
  className: nginx
  host: todo.local

resources:
  requests: { cpu: 100m, memory: 128Mi }
  limits:   { cpu: 200m, memory: 256Mi }
```

━━━━━━━━━━━━━━━━━━━━
## Cluster Configuration
━━━━━━━━━━━━━━━━━━━━

Kubeconfig targets the Kind cluster (context: `kind-todo`).

**Database:** PostgreSQL deployed via bitnami/postgresql Helm subchart. Persistence disabled — data lives in emptyDir and dies with the pod. Fine for local dev; in production use PVC or managed DB (RDS, Cloud SQL).

━━━━━━━━━━━━━━━━━━━━
## Deploy Application
━━━━━━━━━━━━━━━━━━━━

```bash
docker build -t localhost:5000/todo-api:$(git rev-parse --short HEAD) .
docker push localhost:5000/todo-api:$(git rev-parse --short HEAD)

helm upgrade --install todo ./deploy/helm/todo-app \
  --set image.repository=localhost:5000/todo-api \
  --set image.tag=$(git rev-parse --short HEAD)
```

Or use the all-in-one: `make deploy-kind`

━━━━━━━━━━━━━━━━━━━━
## Release Workflow
━━━━━━━━━━━━━━━━━━━━

```bash
make deploy-kind
```

Steps performed:
1. Create Kind cluster (if not exists)
2. Build & push image to local registry (`localhost:5000`)
3. Run Helm deploy
4. Run migrations
5. Configure `/etc/hosts`

━━━━━━━━━━━━━━━━━━━━
## Ingress / Service Exposure
━━━━━━━━━━━━━━━━━━━━

NGINX Ingress Controller installed via Helm. Kind `extraPortMappings` routes host ports 80/443 to the control-plane node.

App is exposed at http://todo.local (Ingress rule with host `todo.local`).

━━━━━━━━━━━━━━━━━━━━
## Explain Choice: NGINX vs MetalLB
━━━━━━━━━━━━━━━━━━━━

**NGINX Ingress** — used in this project. For a local Kind cluster with `extraPortMappings`, ports 80/443 go straight to the control-plane node. NodePort/localhost access is enough for solo dev.

**MetalLB** — would make sense if you need a real LoadBalancer IP on your LAN (access from other machines). For solo dev it's overkill and more config to maintain.

━━━━━━━━━━━━━━━━━━━━
## Cleanup
━━━━━━━━━━━━━━━━━━━━

| Environment | Command | What it does |
|---|---|---|
| Docker Compose | `docker compose down` | Stop containers |
| Docker Compose | `docker compose down -v` | Stop + delete DB data |
| Kubernetes | `make delete-cluster` | Delete Kind cluster |
| Kubernetes | `make clean` | Cluster + registry |
