# Todo App v2

Simple todo app — Go + PostgreSQL + Docker + Kubernetes.

**Tech stack:** Go 1.24, Gin, GORM, PostgreSQL 16, golang-migrate, Docker Compose, Kind, Helm 3, NGINX Ingress, GitHub Actions

---

## Contents / Содержание

- [Prerequisites](#prerequisites)
- [How to run docker compose up](#how-to-run-docker-compose-up)
- [How to run migrations](#how-to-run-migrations)
- [How to seed data](#how-to-seed-data)
- [How to open the app locally](#how-to-open-the-app-locally)
- [How to open Swagger UI](#how-to-open-swagger-ui)
- [Local Kubernetes cluster](#local-kubernetes-cluster)
- [Build pipeline](#build-pipeline)
- [Helm chart](#helm-chart)
- [Architecture & tradeoffs](#architecture--tradeoffs)
- [Common troubleshooting](#common-troubleshooting)
- [Cleanup](#cleanup)

---

## Prerequisites

### For Docker Compose

Just Docker. / Достаточно только Docker.

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker
```

After `newgrp` opens a new shell, verify Docker works without `sudo`:

После того как `newgrp` откроет новый shell, проверьте что Docker работает без `sudo`:

```bash
docker version
```

macOS: https://docs.docker.com/desktop/mac/install/

### For Kubernetes (Kind)

Install Docker first, then: / Сначала Docker, затем:

```bash
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') 

# kind
curl -Lo ./kind "https://kind.sigs.k8s.io/dl/latest/kind-linux-$ARCH" && chmod +x ./kind && sudo mv ./kind /usr/local/bin/kind
kind version

# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -Ls https://dl.k8s.io/release/stable.txt)/bin/linux/$ARCH/kubectl" && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
kubectl version

# helm (auto-detects arch)
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
helm version

# golang-migrate
curl -L "https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-$ARCH.tar.gz" | tar xvz && sudo mv migrate /usr/local/bin/
migrate -version
```


---

## How to run docker compose up

```bash
docker compose up --build -d
```

That's it. This starts three services:

Это всё. Поднимутся три сервиса:

- **db** — PostgreSQL 16, port 5432
- **migrate** — runs migrations and exits / применяет миграции и завершается
- **api** — Go server, port 8080 / Go-сервер, порт 8080

The API won't start until migrations finish. You can check that everything is up with:

API не стартует, пока миграции не завершатся. Проверить что всё поднялось:

```bash
docker compose ps
```

App is available at / Приложение доступно по адресу: http://localhost:8080

---

## How to run migrations

**EN:** Migrations live in `migrations/` — plain SQL files with up/down versions, managed by golang-migrate.

**RU:** Миграции лежат в `migrations/` — обычные SQL-файлы с up/down версиями, управляются golang-migrate.

### Docker Compose

Automatic. The `migrate` service runs before the API and applies all pending migrations. You don't need to do anything.

Автоматически. Сервис `migrate` запускается перед API и применяет все миграции. Делать ничего не нужно.

### Kubernetes

**Note:** If Docker Compose is still running, you'll have two PostgreSQL instances on port 5432 at the same time — that's a recipe for confusion. Stop Docker Compose first:

**Внимание:** Если Docker Compose ещё запущен, два PostgreSQL на порту 5432 одновременно — источник путаницы. Сначала остановите:

```bash
docker compose down
```

тепеь можно выполнять

```bash
kubectl port-forward svc/todo-postgresql 5433:5432 &
migrate -path=./migrations -database="postgres://postgres:postgres@localhost:5433/todo_db?sslmode=disable" up
```

Or use the Makefile shortcut: / Или через Makefile:

```bash
make migrate
```

---

## How to seed data

Test data is in `seeds/seed.sql` — 3 sample todos. It's idempotent, inserts only if the table is empty.

Тестовые данные в `seeds/seed.sql` — 3 примера задач. Идемпотентно, вставляет только если таблица пустая.

Seed is not automatic — I separated it from infra on purpose so you always start with a clean DB unless you explicitly want test data.

Seed не запускается автоматически — я сознательно отделил его от инфраструктуры, чтобы БД всегда стартовала чистой, если вы явно не хотите тестовые данные.

### Docker Compose

```bash
docker compose --profile seed up seed
```

### Kubernetes

```bash
make seed
```

---

## How to open the app locally

### Docker Compose

Open http://localhost:8080 in a browser.

Откройте http://localhost:8080 в браузере.

### Kubernetes

Open http://todo.local in a browser.

Откройте http://todo.local в браузере.

`make deploy-kind` automatically adds `todo.local` to `/etc/hosts`, so it should just work. If it doesn't, check that this line exists:

`make deploy-kind` автоматически добавляет `todo.local` в `/etc/hosts`. Если не работает, проверьте что есть строка:

```
127.0.0.1  todo.local
```

---

## How to open Swagger UI

The API is documented with OpenAPI 3.0. Swagger UI is at `/docs`.

API документирован через OpenAPI 3.0. Swagger UI по адресу `/docs`.

| Environment / Окружение | URL |
|---|---|
| Docker Compose | http://localhost:8080/docs |
| Kubernetes | http://todo.local/docs |

### Endpoints

| Method | Path | What it does / Что делает |
|---|---|---|
| `GET` | `/api/v1/todos` | List todos with filters / Список задач с фильтрами |
| `POST` | `/api/v1/todos` | Create a todo / Создать задачу |
| `GET` | `/api/v1/todos/:id` | Get one todo / Получить задачу |
| `PUT` | `/api/v1/todos/:id` | Update a todo / Обновить задачу |
| `DELETE` | `/api/v1/todos/:id` | Soft delete / Мягкое удаление |
| `POST` | `/api/v1/todos/clear-completed` | Delete all completed / Удалить завершённые |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe + DB ping |

**Filters:** `?completed=true`, `?due_before=2025-01-01T00:00:00Z`, `?due_after=...`, `?search=grocery`

---

## Local Kubernetes cluster

Kind cluster: 1 control-plane + 1 worker node.

Kind-кластер: 1 control-plane + 1 worker нода.

```bash
make deploy-kind
```

One command does everything:

Одна команда делает всё:

1. Starts local Docker Registry on `localhost:5000`
2. Creates Kind cluster
3. Connects registry to Kind network
4. Installs NGINX Ingress Controller
5. Builds and pushes Docker image
6. Deploys PostgreSQL (bitnami/postgresql) and the app via Helm
7. Runs migrations
8. Adds `todo.local` to `/etc/hosts`

After that: http://todo.local

If you need to do the build step manually:

Если нужно собрать вручную:

```bash
docker build -t localhost:5000/todo-api:$(git rev-parse --short HEAD) .
docker push localhost:5000/todo-api:$(git rev-parse --short HEAD)

helm upgrade --install todo ./deploy/helm/todo-app \
  --set image.repository=localhost:5000/todo-api \
  --set image.tag=$(git rev-parse --short HEAD)
```

---

## Build pipeline

CI runs on GitHub Actions (`.github/workflows/ci.yml`), triggers on push/PR to `main`.

CI работает на GitHub Actions, срабатывает на push/PR в `main`.

What runs:

Что запускается:

1. **Lint** — `golangci-lint` (errcheck, staticcheck, gosimple, unused)
2. **OpenAPI validation** — `@redocly/cli lint openapi.yaml`
3. **Helm lint** — `helm lint` with bitnami dependency
4. **Tests** — unit + integration tests against real PostgreSQL, with `-race` flag
5. **Docker build & push** — multi-stage build, pushes to GHCR on `main`

---

## Helm chart

Chart lives in `deploy/helm/todo-app/`.

Чарт находится в `deploy/helm/todo-app/`.

**Templates:**

- `deployment.yaml` — API Deployment with liveness/readiness probes
- `service.yaml` — ClusterIP (port 80 -> 8080)
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

---

## Architecture & tradeoffs

### Why PostgreSQL and not MongoDB?

**EN:** The app has structured data with filters (completed, dueDate range, text search in title) and typed fields. PostgreSQL gives me relational constraints, ACID, and `text[]` arrays for tags out of the box. `ILIKE` covers the search without needing extra services. MongoDB would work fine too, but for a todo app with predictable schema and filters, SQL just fits better — I don't need to duplicate validation in app code.

**RU:** У приложения структурированные данные с фильтрами (completed, диапазон dueDate, поиск по title) и типизированные поля. PostgreSQL даёт реляционные constraints, ACID и массивы `text[]` для тегов из коробки. `ILIKE` покрывает поиск без дополнительных сервисов. MongoDB тоже подошёл бы, но для todo-приложения с предсказуемой схемой и фильтрами SQL удобнее — не нужно дублировать валидацию в коде приложения.

### Why GORM?

**EN:** Quick to get started: soft delete built in, connection pool management, less boilerplate for CRUD. The downside is magic — implicit queries make it harder to spot N+1 problems. I considered sqlc (type-safe SQL, no reflection), but it needs a lot more code for basic CRUD. For a small app this tradeoff is fine. If something gets slow, I can always drop down to raw SQL for that specific query.

**RU:** Быстрый старт: soft delete из коробки, управление пулом соединений, меньше бойлерплейта для CRUD. Минус — магия: неявные запросы усложняют поиск N+1. Рассматривал sqlc (type-safe SQL, без рефлексии), но он требует заметно больше кода для базового CRUD. Для небольшого приложения компромисс нормальный. Если что-то тормозит, всегда можно спуститься до raw SQL для конкретного запроса.

### Why golang-migrate instead of GORM AutoMigrate?

**EN:** AutoMigrate is handy for prototyping but dangerous for anything serious: no down-migrations, no version control on schema, can silently drop data when you rename columns. golang-migrate gives me versioned SQL files (up/down) that I can review in PRs, roll back, and replay on any environment.

**RU:** AutoMigrate удобен для прототипирования, но опасен для чего-то серьёзного: нет down-миграций, нет версионирования схемы, может молча потерять данные при переименовании колонок. golang-migrate даёт версионированные SQL-файлы (up/down), которые можно ревьюить в PR, откатывать и воспроизводить на любом окружении.

### Why Gin?

**EN:** Solid router, recovery middleware out of the box, familiar API if you've used Express. I could've used chi or plain net/http, but Gin needs less boilerplate for routing and middleware — gets things done faster.

**RU:** Надёжный роутер, recovery middleware из коробки, привычный API если работали с Express. Можно было взять chi или чистый net/http, но Gin требует меньше кода для маршрутизации и middleware — быстрее получается.

### Why NGINX Ingress and not MetalLB?

**EN:** For a local Kind cluster, NGINX Ingress with `extraPortMappings` is enough — ports 80/443 on the host go straight to the control-plane node where the Ingress Controller runs. MetalLB makes sense if you need a real LoadBalancer IP on your LAN (access from other machines), but for solo dev that's overkill and more config to maintain.

**RU:** Для локального Kind-кластера хватает NGINX Ingress с `extraPortMappings` — порты 80/443 на хосте идут напрямую в control-plane ноду с Ingress Controller. MetalLB нужен если хочется реальный LoadBalancer IP в локальной сети (доступ с других машин), но для одного разработчика это лишнее и больше конфигурации.

### Why local Docker Registry?

**EN:** I wanted a `build -> push -> pull` flow like in production. `kind load docker-image` works but it's a dev shortcut that loads images straight onto nodes — doesn't match how real clusters work. The local registry on `localhost:5000` reproduces the actual pattern: CI builds, pushes to registry, Kubernetes pulls by tag.

**RU:** Хотел flow `build -> push -> pull` как в production. `kind load docker-image` работает, но это dev-хак, который грузит образы прямо на ноды — не соответствует реальным кластерам. Локальный registry на `localhost:5000` воспроизводит настоящий паттерн: CI собирает, пушит в registry, Kubernetes пуллит по тегу.

### Why bitnami/postgresql Helm chart?

**EN:** Instead of writing my own PostgreSQL manifests, I used the Bitnami chart — it comes with proper health checks, security context, and is battle-tested. For a dev cluster it's a bit much, but it shows how to work with Helm dependencies and configure subcharts through parent values.

**RU:** Вместо написания своих манифестов для PostgreSQL взял чарт Bitnami — он идёт с нормальными health checks, security context и проверен в бою. Для dev-кластера это с запасом, но показывает работу с Helm dependencies и конфигурацию субчартов через parent values.

### Why is PostgreSQL persistence disabled in Kubernetes?

**EN:** It's a dev cluster, data lives in emptyDir and dies with the pod. That's fine for local development. In production you'd use a PersistentVolumeClaim or a managed database (RDS, Cloud SQL).

**RU:** Это dev-кластер, данные живут в emptyDir и пропадают с подом. Для локальной разработки нормально. В production нужен PersistentVolumeClaim или managed БД (RDS, Cloud SQL).

---

## Common troubleshooting

### Pod stuck in CrashLoopBackOff

Usually PostgreSQL isn't ready yet. Check the logs:

Обычно PostgreSQL ещё не готов. Смотрите логи:

```bash
kubectl logs deploy/todo
kubectl logs sts/todo-postgresql
```

The app has health probes (`/healthz`, `/readyz`), Kubernetes will keep restarting until the DB is up. Give it a minute.

У приложения есть health probes, Kubernetes будет перезапускать пока БД не поднимется. Подождите минуту.

If you see `password authentication failed` — passwords in `values.yaml` might be out of sync. Easiest fix:

Если видите `password authentication failed` — пароли в `values.yaml` могут разъехаться. Самый простой фикс:

```bash
make delete-cluster && make deploy-kind
```

### ErrImagePull / ImagePullBackOff

The image is in the local registry. Check it's running:

Образ в локальном registry. Проверьте что он работает:

```bash
docker ps | grep kind-registry
# If not running / Если не запущен:
make create-registry
```

### docker push says "connection refused"

Same thing — registry isn't running:

То же самое — registry не запущен:

```bash
make create-registry
```

### Port already in use

```bash
lsof -i :8080   # Docker Compose
lsof -i :80     # Kind
```

Kill the process or change the port in `.env` / `kind-config.yaml`.

Убейте процесс или поменяйте порт в `.env` / `kind-config.yaml`.

### Running both Docker Compose and Kind at the same time

Technically fine — ports don't overlap (8080 vs 80/443). But having two PostgreSQL instances can be confusing. I'd recommend stopping Compose first:

Технически можно — порты не пересекаются. Но два PostgreSQL одновременно — путаница. Лучше остановить Compose:

```bash
docker compose down
```

---

## Cleanup

### Docker Compose

```bash
docker compose down        # stop containers / остановить контейнеры
docker compose down -v     # + delete DB data / + удалить данные БД
```

### Kubernetes

```bash
make delete-cluster   # delete Kind cluster / удалить кластер
make clean            # cluster + registry / кластер + registry
```
