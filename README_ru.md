# Todo App v2

**Tech stack:** Go 1.24.1, Gin, GORM, PostgreSQL 16.2, golang-migrate v4.19.1, Docker Compose, Kind v0.31.0, kubectl v1.35.3, Helm v4.1.3, NGINX Ingress v1.12.1, GitHub Actions

━━━━━━━━━━━━━━━━━━━━
## Requirements
━━━━━━━━━━━━━━━━━━━━

**REST API** задокументирован через OpenAPI 3.0 — Swagger UI доступен по `/docs`.

| Окружение | Swagger URL |
|---|---|
| Docker Compose | http://localhost:8080/docs |
| Kubernetes | http://todo.local/docs |

━━━━━━━━━━━━━━━━━━━━
## Data Layer
━━━━━━━━━━━━━━━━━━━━

**База данных:** PostgreSQL 16.2

**Почему PostgreSQL?** Структурированные данные с фильтрами (completed, диапазон dueDate, поиск по title) и типизированные поля. Реляционные constraints, ACID, массивы `text[]` для тегов из коробки. `ILIKE` покрывает поиск без дополнительных сервисов. Для todo-приложения с предсказуемой схемой и фильтрами SQL подходит лучше MongoDB — не нужно дублировать валидацию в коде приложения.

**ORM:** GORM

**Почему GORM?** Меньше кода для базового CRUD. Альтернатива sqlc требует заметно больше бойлерплейта.

━━━━━━━━━━━━━━━━━━━━
## Migrations
━━━━━━━━━━━━━━━━━━━━

**Инструмент:** [golang-migrate v4.19.1](https://github.com/golang-migrate/migrate) — версионированные SQL-файлы (up/down) в `migrations/`.

**Почему не GORM AutoMigrate?** Нет down-миграций, нет версионирования схемы, может молча потерять данные при переименовании колонок. Файлы golang-migrate можно ревьюить в PR, откатывать и воспроизводить на любом окружении.

━━━━━━━━━━━━━━━━━━━━
## Minimal Feature Set
━━━━━━━━━━━━━━━━━━━━

**CRUD Todos** с полями: title, completed, dueDate, tags (опционально).

**Эндпоинты:**

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/api/v1/todos` | Список задач с фильтрами |
| `POST` | `/api/v1/todos` | Создать задачу |
| `GET` | `/api/v1/todos/:id` | Получить задачу |
| `PUT` | `/api/v1/todos/:id` | Обновить задачу |
| `DELETE` | `/api/v1/todos/:id` | Мягкое удаление |
| `POST` | `/api/v1/todos/clear-completed` | Удалить все завершённые |

**Фильтры:** `?completed=true`, `?due_before=2025-01-01T00:00:00Z`, `?due_after=...`, `?search=grocery`

**Health-эндпоинты:**

| Путь | Тип |
|---|---|
| `/healthz` | Liveness probe |
| `/readyz` | Readiness probe + DB ping |

**Тесты:** unit-тесты сервисного слоя + интеграционные API-тесты с реальным PostgreSQL (флаг `-race`).

**Фронтенд:** статическая HTML-страница (Alpine.js + Tailwind CSS), раздаётся из API — список / добавление / завершение задач.

━━━━━━━━━━━━━━━━━━━━
## Deliverables
━━━━━━━━━━━━━━━━━━━━

- `openapi.yaml` — спецификация OpenAPI 3.0, валидируется в CI
- Исходный код — этот репозиторий
- Architecture & tradeoffs — описаны в разделах [Data Layer](#data-layer), [Migrations](#migrations) и [Ingress](#ingress--service-exposure) этого README

━━━━━━━━━━━━━━━━━━━━
## Local Dev Environment
━━━━━━━━━━━━━━━━━━━━

Цель: запуск одной командой через Docker Compose.

### Prerequisites

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker
```

Проверка: `docker version`

macOS: https://docs.docker.com/desktop/mac/install/

Для Kubernetes — сначала Docker, затем:

```bash
sudo apt install make   # если не установлен
make install-tools      # установит kind, kubectl, helm, golang-migrate
```

### Как запустить docker compose up

```bash
docker compose up --build -d
```

| Сервис | Описание |
|---|---|
| **db** | PostgreSQL 16.2, порт 5432 |
| **migrate** | Применяет миграции и завершается |
| **api** | Go-сервер, порт 8080 |

API не стартует, пока миграции не завершатся. Проверить статус: `docker compose ps`

### Как запустить миграции

**Docker Compose** — автоматически. Сервис `migrate` применяет все миграции перед запуском API.

**Kubernetes:**

```bash
kubectl port-forward svc/todo-postgresql 5433:5432 &
migrate -path=./migrations -database="postgres://postgres:postgres@localhost:5433/todo_db?sslmode=disable" up
```

Или: `make migrate`

### Как загрузить тестовые данные

Тестовые данные в `seeds/seed.sql` — 3 примера задач. Идемпотентно, вставляет только если таблица пустая. Не запускается автоматически — БД стартует чистой, если вы явно не загрузите данные.

| Окружение | Команда |
|---|---|
| Docker Compose | `docker compose --profile seed up seed` |
| Kubernetes | `make seed` |

### Как открыть приложение локально

| Окружение | URL |
|---|---|
| Docker Compose | http://localhost:8080 |
| Kubernetes | http://todo.local |

`make deploy-kind` автоматически добавляет `todo.local` в `/etc/hosts`.

### Как открыть Swagger UI

| Окружение | URL |
|---|---|
| Docker Compose | http://localhost:8080/docs |
| Kubernetes | http://todo.local/docs |

### Частые проблемы

**CrashLoopBackOff** — обычно PostgreSQL ещё не готов. Проверьте `kubectl logs deploy/todo` и `kubectl logs sts/todo-postgresql`. Подождите минуту. Если `password authentication failed`: `make delete-cluster && make deploy-kind`

**ErrImagePull / ImagePullBackOff** — локальный registry не запущен: `make create-registry`

**"connection refused" при docker push** — тот же фикс: `make create-registry`

**Порт уже занят** — `lsof -i :8080` (Compose) или `lsof -i :80` (Kind). Убейте процесс или поменяйте порт в `.env` / `kind-config.yaml`.

**Запущены и Compose, и Kind одновременно** — два PostgreSQL (один из Docker Compose, второй из Kind) могут конфликтовать по порту 5432. Перед работой с Kind остановите Compose: `docker compose down`

━━━━━━━━━━━━━━━━━━━━
## Local Kubernetes Cluster
━━━━━━━━━━━━━━━━━━━━

Kind-кластер: 1 control-plane + 1 worker нода.

```bash
make deploy-kind
```

Одна команда делает всё:

1. Запускает локальный Docker Registry на `localhost:5000`
2. Создаёт Kind-кластер
3. Подключает registry к сети Kind
4. Устанавливает NGINX Ingress Controller
5. Собирает и пушит Docker-образ
6. Деплоит PostgreSQL (bitnami/postgresql) и приложение через Helm
7. Запускает миграции
8. Добавляет `todo.local` в `/etc/hosts`

После этого: http://todo.local

━━━━━━━━━━━━━━━━━━━━
## Build Pipeline
━━━━━━━━━━━━━━━━━━━━

CI работает на GitHub Actions (`.github/workflows/ci.yml`), срабатывает на push/PR в `main`.

1. **Lint** — `golangci-lint` (errcheck, staticcheck, gosimple, unused)
2. **Валидация OpenAPI** — `@redocly/cli lint openapi.yaml`
3. **Helm lint** — `helm lint` с зависимостью bitnami
4. **Тесты** — unit + интеграционные тесты с реальным PostgreSQL, флаг `-race`
5. **Docker build & push** — multi-stage сборка, пуш в GHCR на `main`

━━━━━━━━━━━━━━━━━━━━
## Helm Packaging
━━━━━━━━━━━━━━━━━━━━

Чарт находится в `deploy/helm/todo-app/`.

**Шаблоны:**

- `deployment.yaml` — API Deployment с liveness/readiness probes
- `service.yaml` — ClusterIP (порт 80 → 8080)
- `ingress.yaml` — NGINX Ingress, хост: `todo.local`
- `configmap.yaml` — настройки подключения к БД
- `secret.yaml` — креденшалы БД

**Зависимость:** bitnami/postgresql 18.5.11 (persistence отключён для dev)

**Основные values:**

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

Kubeconfig указывает на Kind-кластер (контекст: `kind-todo`).

**База данных:** PostgreSQL деплоится через субчарт bitnami/postgresql. Persistence отключён — данные живут в emptyDir и пропадают с подом. Для локальной разработки нормально; в production нужен PVC или managed БД (RDS, Cloud SQL).

━━━━━━━━━━━━━━━━━━━━
## Deploy Application
━━━━━━━━━━━━━━━━━━━━

Деплой приложения в Kind-кластер через Helm. Образ берётся из локального registry, тег — короткий хеш коммита:

```bash
helm upgrade --install todo ./deploy/helm/todo-app \
  --set image.repository=localhost:5000/todo-api \
  --set image.tag=$GIT_SHA
```

━━━━━━━━━━━━━━━━━━━━
## Release Workflow
━━━━━━━━━━━━━━━━━━━━

```bash
make deploy-kind
```

Выполняемые шаги:
1. Создать Kind-кластер (если не существует)
2. Собрать и запушить образ в локальный registry (`localhost:5000`)
3. Запустить Helm deploy
4. Запустить миграции
5. Настроить `/etc/hosts`

━━━━━━━━━━━━━━━━━━━━
## Ingress / Service Exposure
━━━━━━━━━━━━━━━━━━━━

NGINX Ingress Controller установлен через Helm. Kind `extraPortMappings` маршрутизирует порты 80/443 хоста на control-plane ноду.

Приложение доступно по http://todo.local (Ingress rule с хостом `todo.local`).

━━━━━━━━━━━━━━━━━━━━
## Explain Choice: NGINX vs MetalLB
━━━━━━━━━━━━━━━━━━━━

**NGINX Ingress** — используется в этом проекте. Kind `extraPortMappings` в `kind-config.yaml` пробрасывает порты 80/443 с хоста (или виртуальной машины) в control-plane ноду. Приложение доступно по `http://todo.local` — работает и локально, и на VM.

**MetalLB** — нужен, если Kind запущен в среде, где `extraPortMappings` недоступен, или нужен выделенный LoadBalancer IP в локальной сети (доступ с других машин). В нашем случае не требуется.

━━━━━━━━━━━━━━━━━━━━
## Cleanup
━━━━━━━━━━━━━━━━━━━━

| Окружение | Команда | Что делает |
|---|---|---|
| Docker Compose | `docker compose down` | Остановить контейнеры |
| Docker Compose | `docker compose down -v` | Остановить + удалить данные БД |
| Kubernetes | `make delete-cluster` | Удалить Kind-кластер |
| Kubernetes | `make clean` | Кластер + registry |
