# Todo App v2

A small REST-based todo service built in Go, designed as a portfolio project to showcase a full path from local development to production. The same codebase ships in four ways: Docker Compose for fast iteration, a local Kind cluster for Kubernetes practice, a GKE cluster managed by Terraform with FluxCD running GitOps, and a single-node VPS deployment driven by Ansible.

[![CI](https://github.com/Subcore/todo-app/actions/workflows/ci.yml/badge.svg)](.github/workflows/ci.yml)
![Go](https://img.shields.io/badge/go-1.24.1-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-blue)

[Русская версия](README.ru.md)

---

## Tech Stack

| Layer | Tool |
|---|---|
| Language / runtime | Go 1.24.1 |
| HTTP framework | Gin |
| ORM | GORM |
| Database | PostgreSQL 16.2 |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) v4.19.1 |
| Frontend | Static HTML + Alpine.js + Tailwind CSS |
| API spec | OpenAPI 3.0 (Swagger UI at `/docs`) |
| Containers | Docker, Docker Compose |
| Local Kubernetes | Kind v0.31.0 |
| Kubernetes tools | kubectl v1.35.3, Helm v4.1.3 |
| Ingress | NGINX Ingress v1.12.1 |
| Cloud infra (IaC) | Terraform (GCP / GKE) |
| GitOps | FluxCD (Helm + image automation) |
| VPS deployment | Ansible |
| CI | GitHub Actions, GHCR |

---

## Application Architecture

The app is split into three independent components behind a single entry point. The frontend is pure static content — it never touches the database directly; all data access goes through the backend API.

```
                       ┌──────────────┐
                       │   Browser    │
                       └──────┬───────┘
                              │ HTTP
                              ▼
                  ┌───────────────────────┐
                  │      API Gateway      │   single entry point
                  │  (nginx / k8s ingress)│   routes /web and /api
                  └─────┬───────────┬─────┘
                  /web/ │           │ /api/
                        ▼           ▼
              ┌─────────────────┐  ┌─────────────────┐
              │    Frontend     │  │     Backend     │
              │   (todo-web)    │  │   (todo-api)    │
              │ static HTML +   │  │  Go + Gin REST  │
              │ Alpine.js + Tw  │  │      API        │
              └─────────────────┘  └────────┬────────┘
                                            │ SQL (only path to DB)
                                            ▼
                                   ┌─────────────────┐
                                   │   PostgreSQL    │
                                   └─────────────────┘
```

- **API Gateway** — the only component reachable from the outside. In Compose it's the nginx inside `todo-web`; in Kubernetes it's NGINX Ingress; on a VPS it's the host nginx ([ansible/roles/nginx/templates/todo.conf.j2](ansible/roles/nginx/templates/todo.conf.j2)).
- **Frontend (`todo-web`)** — static HTML + Alpine.js + Tailwind, served by nginx. Has no database credentials and no DB driver. Talks to the backend over HTTP through the gateway.
- **Backend (`todo-api`)** — Go/Gin REST service. The only component that holds DB credentials and opens connections to PostgreSQL.
- **PostgreSQL** — not exposed via the gateway. Reachable only from the backend pod/container on the internal network.

The same three-tier split is preserved in every deployment target (Compose, Kind, GKE, VPS); only the gateway implementation and the DB hosting change.

---

## Deployment Architecture

End-to-end GitOps flow for the GKE deployment:

```
 ┌────────────┐    push    ┌──────────────────┐   build & push   ┌─────────┐
 │  Developer │ ─────────▶ │  GitHub Actions  │ ───────────────▶ │  GHCR   │
 └────────────┘            │   (CI workflow)  │                  └────┬────┘
                           └──────────────────┘                       │
                                                                      │ image
                                                                      ▼
 ┌──────────────┐  reconcile  ┌─────────────┐  HelmRelease   ┌─────────────────┐
 │ k8s/cluster/ │ ◀─────────  │   FluxCD    │ ─────────────▶ │  GKE (todo-app) │
 │  (manifests) │   git pull  │  on cluster │                │  Postgres + API │
 └──────┬───────┘             └─────┬───────┘                │  + web + nginx  │
        ▲                           │                        └─────────────────┘
        │ patch HelmRelease         │ image automation
        └───────────────────────────┘  detects new tag in GHCR
```

- Code merges to `main` → CI builds two images (`todo-api`, `todo-web`) and pushes them to `ghcr.io`.
- Flux's image-reflector + image-automation controllers watch GHCR, find a newer tag matching `^main-<timestamp>-<sha>$`, and rewrite [k8s/cluster/todo-app/helmrelease.yaml](k8s/cluster/todo-app/helmrelease.yaml) on `main`.
- Flux reconciles the new commit and rolls out the new HelmRelease against the cluster.

---

## Quick Start

Pick one of the four paths below.

### Option 1 — Docker Compose (local dev)

The fastest way to run the app on your laptop. Brings up PostgreSQL, runs migrations, then starts the API and the web container.

```bash
cp .env.example .env
docker compose up --build -d
```

| URL | What |
|---|---|
| http://localhost:8080 | API (`/api/v1/todos`, `/healthz`, `/readyz`) |
| http://localhost:8080/docs | Swagger UI |
| http://localhost | Static frontend (separate `todo-web` container) |

Useful commands:

```bash
docker compose logs -f api          # tail API logs
docker compose --profile seed up seed   # load seeds/seed.sql (3 sample todos)
docker compose down                 # stop containers
docker compose down -v              # stop + drop the postgres volume
```

---

### Option 2 — Local Kubernetes (Kind)

A self-contained local cluster: Kind + a local Docker registry, NGINX Ingress, PostgreSQL via the bitnami Helm chart, and our own Helm chart for the app.

```bash
make install-tools    # installs kind, kubectl, helm, golang-migrate
make deploy-kind      # one-shot 11-step pipeline
```

What `make deploy-kind` does:

1. Spawns a local Docker registry on `localhost:5000`
2. Creates the Kind cluster (control-plane + worker)
3. Connects the registry to Kind's network
4. Configures containerd on each node to pull from the registry
5. Installs NGINX Ingress Controller
6. Builds and pushes API + web Docker images
7. Deploys PostgreSQL (bitnami) and the app via Helm
8. Runs database migrations
9. Adds `todo.local` to `/etc/hosts`
10. Runs a smoke test against `/api/v1/healthz`

Once finished:

| URL | What |
|---|---|
| http://todo.local/web | Frontend |
| http://todo.local/api | API |
| http://todo.local/api/docs | Swagger UI |

Troubleshooting:

- **`CrashLoopBackOff`** — Postgres usually isn't ready yet. `kubectl logs deploy/todo` and `kubectl logs sts/todo-postgresql`. If you see `password authentication failed`, recreate: `make delete-cluster && make deploy-kind`.
- **`ErrImagePull` / `ImagePullBackOff`** — local registry isn't running: `make create-registry`.
- **`port already in use`** — `lsof -i :8080` (Compose) or `lsof -i :80` (Kind). Either kill the process or change the port in `.env` / [kind-config.yaml](kind-config.yaml).
- **Both Compose and Kind running** — both expose Postgres on 5432. Stop Compose first: `docker compose down`.

---

### Option 3 — GCP / GKE with Terraform + FluxCD (production path)

This is the main production-style deployment. Terraform provisions the GKE cluster and bootstraps FluxCD; from that point on, the cluster reconciles itself from this repo.

**Prerequisites:**

- `gcloud` CLI (authenticated: `gcloud auth login && gcloud auth application-default login`)
- `kubectl`, `helm`, `terraform` ≥ 1.5
- A GCP project with billing enabled
- A GitHub Personal Access Token — see Step 0 below

**Step 0 — create a GitHub Personal Access Token**

Terraform uses one token for two things ([terraform/flux.tf](terraform/flux.tf)):

1. The `github` provider creates a repository deploy key that Flux uses to clone and push to this repo (image-automation commits).
2. The same token is written into a Kubernetes `ghcr-secret` so the cluster can pull `ghcr.io/<owner>/todo-{api,web}` images.

Create a **classic** PAT at https://github.com/settings/tokens/new with these scopes:

| Scope | Why |
|---|---|
| `repo` | Manage the deploy key on the repository |
| `read:packages` | Pull images from GHCR (used in `ghcr-secret`) |
| `write:packages` | Only needed if you push from this token; CI uses `GITHUB_TOKEN` instead, so usually skip |

> Fine-grained PATs currently don't authenticate against GHCR reliably — use a classic PAT.

Pass the token via an environment variable, **not** through `terraform.tfvars` (the file is gitignored, but it's easy to leak it via screen-shares, backups, or accidental edits to the example file):

```bash
export TF_VAR_github_token=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

The token ends up in two places after `tf-apply`:

- Terraform state in your GCS bucket — keep that bucket private.
- The `ghcr-secret` Kubernetes Secret (in `flux-system` and `todo-app` namespaces).

If the token leaks, revoke it on GitHub, then re-export `TF_VAR_github_token` and run `terraform apply` again — the `ghcr-secret` and deploy key get rotated in place.

**Step 1 — provision infrastructure with Terraform**

```bash
cp terraform/terraform.tfvars.example terraform/terraform.tfvars
# Edit terraform.tfvars: set project_id (region/zone are optional).
# Pass secrets through env vars instead of writing them into terraform.tfvars:
export TF_VAR_db_password=$(openssl rand -hex 16)
export TF_VAR_github_owner=<your-gh-user-or-org>
export TF_VAR_github_repository=todo-app
export TF_VAR_github_token=$GITHUB_PAT_FROM_STEP_0

make tf-bootstrap GCP_PROJECT=<your-project-id>   # creates GCS bucket for tfstate
make tf-init      GCP_PROJECT=<your-project-id>
make tf-plan      GCP_PROJECT=<your-project-id>
make tf-apply     GCP_PROJECT=<your-project-id>
make tf-kubeconfig GCP_PROJECT=<your-project-id>  # configure kubectl context
```

What Terraform creates ([terraform/main.tf](terraform/main.tf), [terraform/flux.tf](terraform/flux.tf)):

| Resource | Details |
|---|---|
| VPC | `todo-vpc` with subnet `10.10.0.0/20`, secondary ranges for pods/services |
| GKE cluster | `todo-cluster`, zonal (`asia-southeast1-b` by default) |
| Node pool: `system-pool` | `e2-standard-2` Spot — Flux, ingress-nginx |
| Node pool: `db-pool` | `e2-small` Spot, tainted — PostgreSQL only |
| Node pool: `app-pool` | `e2-standard-4` Spot, autoscaling 1–2 — application pods |
| Static IP | `todo-ingress-ip` — wired into ingress-nginx via ConfigMap |
| FluxCD | bootstrapped against this repo at path `k8s/cluster`, with image-reflector + image-automation controllers |
| Secrets | `ghcr-secret` (in `flux-system` and `todo-app`), `todo-db-credentials` |

**Step 2 — Flux takes over**

After `tf-apply` finishes, Flux pulls [k8s/cluster/](k8s/cluster/) and deploys:

- `ingress-nginx` — [k8s/cluster/ingress-nginx/helmrelease.yaml](k8s/cluster/ingress-nginx/helmrelease.yaml)
- `todo-postgresql` (bitnami chart, persistent volume) — [k8s/cluster/todo-app/helmrelease.yaml](k8s/cluster/todo-app/helmrelease.yaml)
- `todo-app` (our Helm chart from [deploy/helm/todo-app](deploy/helm/todo-app/))

Verify:

```bash
kubectl get helmrelease -A
flux get all -A
flux logs -f
```

The app becomes reachable at `http://<static-ip>.nip.io/web` (the IP comes from `kubectl -n ingress-nginx get svc ingress-nginx-controller`; Flux injects it into the ingress host as `<ip>.nip.io`).

**Step 3 — image automation (continuous deploy)**

[k8s/cluster/todo-app/image-automation.yaml](k8s/cluster/todo-app/image-automation.yaml) defines two `ImageRepository` + `ImagePolicy` pairs (one for `todo-api`, one for `todo-web`) and an `ImageUpdateAutomation`. Lifecycle:

1. CI on `main` builds and pushes `ghcr.io/<owner>/todo-api:main-<ts>-<sha>` and the same for `todo-web`.
2. Flux's image-reflector picks up new tags within 5 minutes.
3. The image-automation controller patches the `# {"$imagepolicy": ...}` setter markers in `helmrelease.yaml` and pushes a `chore: update images to ...` commit to `main`.
4. Flux reconciles, the HelmRelease is updated, and the new pods roll out.

**Cleanup:** `make tf-destroy GCP_PROJECT=<your-project-id>` (tears down GKE, VPC, static IP, Flux deploy key — everything Terraform created).

---

### Option 4 — VPS deployment (Ansible)

A "classic" deployment for a single Linux VPS: PostgreSQL on the host, the Go binary as a systemd service, static frontend served by nginx, with nginx also reverse-proxying `/api` to the Go process.

**Prerequisites:**

- A Linux VPS reachable over SSH (root or sudo) with an SSH key on your machine
- Ansible: `pip install ansible`
- Go (used for cross-compiling the binary on your machine)

**Configuration:**

```bash
cp ansible/inventory.example ansible/inventory.ini
# Edit ansible/inventory.ini — set the server IP and your SSH key path
```

[ansible/inventory.example](ansible/inventory.example):

```ini
[servers]
<YOUR_SERVER_IP> ansible_user=root ansible_port=22 ansible_ssh_private_key_file=~/.ssh/id_rsa
```

**Deploy:**

```bash
make deploy-vps
```

This runs `make build-linux` (cross-compiles `cmd/api/main.go` to a Linux amd64 binary) and then `ansible-playbook ansible/playbook.yml`. The four roles in [ansible/roles/](ansible/roles/):

| Role | What it does |
|---|---|
| `postgresql` | Installs PostgreSQL, creates the `todo_db` database and `todo` user |
| `backend` | Copies the Go binary to `/opt/todo-api`, creates a systemd unit, enables the service |
| `web` | Copies `web/index.html`, `docs.html`, and assets to `/var/www/todo` |
| `nginx` | Installs nginx, deploys a reverse-proxy config (static + `/api` → `127.0.0.1:8080`) |

Result: `http://<YOUR_SERVER_IP>` (frontend), `http://<YOUR_SERVER_IP>/api` (API), `http://<YOUR_SERVER_IP>/api/docs` (Swagger).

---

## Project Structure

```
.
├── cmd/api/                 # main.go — Gin server entrypoint
├── internal/
│   ├── config/              # env-based config loading
│   ├── handler/             # HTTP handlers (todo, health)
│   ├── model/               # GORM models
│   ├── repository/          # DB queries
│   ├── router/              # route registration + middleware
│   └── service/             # business logic
├── migrations/              # golang-migrate up/down SQL files
├── seeds/seed.sql           # idempotent sample data
├── tests/                   # integration tests (real PostgreSQL)
├── web/                     # static frontend (Alpine.js + Tailwind)
│
├── deploy/helm/todo-app/    # Helm chart used by Kind and FluxCD
│
├── k8s/cluster/             # FluxCD manifests reconciled on GKE
│   ├── flux-system/         # Flux components (bootstrapped by Terraform)
│   ├── ingress-nginx/       # ingress-nginx HelmRelease
│   └── todo-app/            # app namespace, HelmRelease, image automation
│
├── terraform/               # GCP infrastructure (VPC + GKE + Flux bootstrap)
│   └── bootstrap/           # one-shot module that creates the GCS state bucket
│
├── ansible/                 # VPS deployment
│   ├── playbook.yml
│   ├── inventory.example
│   └── roles/{nginx,postgresql,backend,web}/
│
├── .github/workflows/ci.yml # CI: lint, OpenAPI/Helm lint, tests, image build & push
├── docker-compose.yaml      # local dev stack
├── Dockerfile, Dockerfile.web
├── kind-config.yaml         # Kind cluster definition
├── openapi.yaml             # OpenAPI 3.0 spec (validated in CI)
└── Makefile                 # all developer/ops entry points
```

---

## Configuration

Environment variables read by [cmd/api/main.go](cmd/api/main.go) via [internal/config/](internal/config/). See [.env.example](.env.example) for the full list.

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `8080` | API listen port |
| `DB_HOST` | `localhost` | |
| `DB_PORT` | `5432` | |
| `DB_USER` | `postgres` | |
| `DB_PASSWORD` | `postgres` | Override in non-dev environments |
| `DB_NAME` | `todo_db` | CI uses `todo_test` |
| `DB_SSLMODE` | `disable` | |
| `DB_MAX_IDLE_CONNS` | `10` | |
| `DB_MAX_OPEN_CONNS` | `100` | |
| `DB_CONN_MAX_LIFETIME` | `1h` | |
| `ENABLE_SWAGGER` | unset | Set to `true` to expose `/docs` |

Per-deployment example files:

- [.env.example](.env.example) — Docker Compose
- [terraform/terraform.tfvars.example](terraform/terraform.tfvars.example) — Terraform variables
- [ansible/inventory.example](ansible/inventory.example) — Ansible inventory

---

## Development

Common `make` targets (run `make help` for the full list):

| Target | Description |
|---|---|
| `make test` | `go test ./...` |
| `make lint` | `golangci-lint run` + `helm lint` |
| `make run` | `docker compose up` |
| `make deploy-kind` | Full local Kubernetes pipeline |
| `make migrate` | Apply migrations against the Kind Postgres |
| `make migrate-down` | Roll back the last migration |
| `make seed` | Load `seeds/seed.sql` into the Kind Postgres |
| `make build-linux` | Cross-compile the Go binary for Linux amd64 |
| `make deploy-vps` | Build binary + run Ansible playbook |
| `make tf-{bootstrap,init,plan,apply,destroy,kubeconfig}` | Terraform lifecycle |
| `make clean` | Delete Kind cluster and registry |

### Tests

Unit tests (no DB):

```bash
go test ./internal/handler/... ./internal/service/... -v -race
```

Integration tests (require PostgreSQL):

```bash
docker compose up db -d
PGPASSWORD=postgres psql -h localhost -U postgres -c "CREATE DATABASE todo_test;"
migrate -path migrations \
  -database "postgres://postgres:postgres@localhost:5432/todo_test?sslmode=disable" up
TEST_DB_DSN="host=localhost port=5432 user=postgres password=postgres dbname=todo_test sslmode=disable" \
  go test ./... -v -race
```

> CI uses `todo_test`; the Docker Compose app uses `todo_db` — they don't collide.

### API surface

REST API documented in [openapi.yaml](openapi.yaml) and rendered as Swagger UI at `/docs` when `ENABLE_SWAGGER=true`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/todos` | List todos (filters: `completed`, `due_before`, `due_after`, `search`) |
| `POST` | `/api/v1/todos` | Create a todo |
| `GET` | `/api/v1/todos/:id` | Get one todo |
| `PUT` | `/api/v1/todos/:id` | Update a todo |
| `DELETE` | `/api/v1/todos/:id` | Soft delete |
| `POST` | `/api/v1/todos/clear-completed` | Delete all completed todos |
| `GET` | `/healthz` | Liveness |
| `GET` | `/readyz` | Readiness + DB ping |

---

## CI/CD Pipeline

GitHub Actions workflow: [.github/workflows/ci.yml](.github/workflows/ci.yml). Triggers on push and pull requests to `main`.

1. **Lint** — `golangci-lint` (errcheck, staticcheck, gosimple, unused).
2. **OpenAPI validation** — `@redocly/cli lint openapi.yaml`.
3. **Helm lint** — `helm lint` on [deploy/helm/todo-app](deploy/helm/todo-app/) with the bitnami subchart resolved.
4. **Tests** — unit + integration against a real PostgreSQL service container, with the `-race` flag.
5. **Docker build & push** — multi-stage builds for `todo-api` and `todo-web`, pushed to `ghcr.io` on `main` with tag `main-<timestamp>-<short-sha>` (the format Flux's image policies match).

After the push, Flux's image automation kicks in (see [Option 3 / Step 3](#option-3--gcp--gke-with-terraform--fluxcd-production-path)).

---

## Cleanup

| Path | Command | What it removes |
|---|---|---|
| Docker Compose | `docker compose down` | Stop containers (keep volumes) |
| Docker Compose | `docker compose down -v` | Stop + drop the Postgres volume |
| Kind | `make delete-cluster` | Destroy the Kind cluster |
| Kind | `make clean` | Cluster + local registry + images |
| GKE | `make tf-destroy GCP_PROJECT=<id>` | Tear down VPC, GKE, static IP, Flux deploy key |
| VPS | n/a | Run `systemctl stop todo-api`, remove `/opt/todo-api`, `/var/www/todo`, the nginx site config, and (optionally) PostgreSQL — Ansible doesn't have an `uninstall` playbook |

---
