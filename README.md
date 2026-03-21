# Todo App v2

Простое приложение для управления задачами — Go API + PostgreSQL.

---

## Prerequisites

| Tool | Версия | Зачем | Установка |
|------|--------|-------|-----------|
| Go | 1.24+ | сборка и запуск API | https://go.dev/doc/install |
| Docker | 24+ | контейнеризация | https://docs.docker.com/get-docker/ |
| kind | 0.20+ | локальный Kubernetes-кластер | https://kind.sigs.k8s.io/docs/user/quick-start/#installation |
| kubectl | 1.28+ | управление кластером | https://kubernetes.io/docs/tasks/tools/ |
| Helm | 3.x | деплой чартов | https://helm.sh/docs/intro/install/ |
| golang-migrate | 4.x | CLI для миграций БД | https://github.com/golang-migrate/migrate/releases |

---

## Quick start (Docker Compose)

```bash
docker compose up --build -d
```

Поднимутся четыре сервиса:

- **db** — PostgreSQL 16 на порту `5432`
- **migrate** — прогон миграций (завершается после применения)
- **seed** — загрузка тестовых данных (завершается после вставки)
- **api** — Go-сервер на порту `8080`

Приложение доступно по адресу **http://localhost:8080**
или **http://todo.local:8080**.

---

## Quick start (Kubernetes)

```bash
make deploy-kind
```

Команда выполняет полный пайплайн: создание kind-кластера, сборку Docker-образа, установку Ingress Controller, деплой PostgreSQL и приложения через Helm, прогон миграций.

Добавьте запись в `/etc/hosts`:

```
127.0.0.1  todo.local
```

Приложение доступно по адресу **http://todo.local**.

---

## Migrations

Миграции хранятся в каталоге `migrations/` в формате SQL (up/down).

- **Docker Compose** — сервис `migrate` запускается автоматически после старта БД и применяет все pending-миграции. API стартует только после завершения миграций.
- **Kubernetes (kind)** — выполните `make migrate`. Под капотом используется `kubectl port-forward` к поду PostgreSQL и CLI `golang-migrate`.

---

## Seed data

Тестовые данные находятся в `seeds/seed.sql` и содержат три примера задач. Seed идемпотентен — данные вставляются только если таблица пуста.

- **Docker Compose** — сервис `seed` загружает данные автоматически после миграций.
- **Вручную** — подключитесь к БД и выполните:
  ```bash
  psql -h localhost -U postgres -d todo_db -f seeds/seed.sql
  ```

---

## Swagger UI

| Окружение | URL |
|-----------|-----|
| Docker Compose | http://localhost:8080/docs |
| Kubernetes | http://todo.local/docs |

---

## Troubleshooting

### Pod в CrashLoopBackOff

Чаще всего — PostgreSQL ещё не готов. Проверьте логи:

```bash
kubectl logs deploy/todo
kubectl logs sts/todo-postgresql
```

API использует health-проверки (`/healthz`, `/readyz`), поэтому Kubernetes перезапустит под автоматически, когда БД станет доступна.

### ErrImagePull / ImagePullBackOff

Образ `localhost:5000/todo-api:<git-sha>` собирается локально и загружается в kind через `kind load docker-image`. Убедитесь, что `imagePullPolicy` установлен в `IfNotPresent` (значение по умолчанию в `values.yaml`).

### Port already in use

Порт `8080` (Compose) или `80/443` (kind Ingress) уже заняты:

```bash
lsof -i :8080
# или для kind
lsof -i :80
```

Завершите процесс или измените порт в `.env` / `kind-config.yaml`.

### Перед запуском kind — остановите Docker Compose

```bash
docker compose down
```

Хотя порты Docker Compose (8080) и kind Ingress (80/443) не пересекаются, одновременная работа двух PostgreSQL может вызвать путаницу.

---

## Architecture & Tradeoffs

### Почему PostgreSQL

Реляционные constraints, ACID-транзакции и типизированные массивы `text[]` (используются для тегов) делают Postgres естественным выбором. `ILIKE` покрывает потребности в полнотекстовом поиске без внешних зависимостей. MongoDB дал бы schemaless-гибкость, но для структурированных данных с фильтрами и связями SQL удобнее — не нужно дублировать логику валидации на стороне приложения.

### Почему GORM

Быстрый старт: auto-migrate для dev-окружения, soft delete из коробки, встроенный connection pool management. Минус — магия и неоптимальные запросы на сложных кейсах (implicit query generation затрудняет обнаружение N+1). `sqlc` дал бы type-safe SQL без рантайм-рефлексии, но потребовал бы значительно больше boilerplate для базового CRUD. Для MVP компромисс оправдан — критичные пути можно заменить на raw SQL без переписывания остального кода.

### Почему golang-migrate (а не GORM AutoMigrate)

GORM AutoMigrate удобен для прототипирования, но в production ненадёжен: он не поддерживает down-миграции, не гарантирует детерминированную схему, может молча потерять данные при переименовании колонок. golang-migrate даёт версионированные SQL-файлы (up/down), которые легко ревьюить в PR, откатывать и воспроизводить на любом окружении.

### Почему Gin

Production-ready роутер со встроенным recovery middleware, structured logging и знакомым Express-like API. `chi` или стандартный `net/http` тоже подошли бы, но Gin быстрее для прототипирования: меньше кода на обвязку middleware и маршрутизацию.

### Почему NGINX Ingress без MetalLB

Для локального kind-кластера достаточно NGINX Ingress Controller с `extraPortMappings` — трафик на порты 80/443 хоста пробрасывается напрямую в контейнер control-plane ноды, где работает Ingress Controller. MetalLB нужен, если требуется реальный LoadBalancer IP в локальной сети (например, для доступа с других машин в LAN). Для single-developer сценария это избыточно и добавляет сложность конфигурации.

### Почему persistence отключён для PostgreSQL в Kubernetes

В задании указано «persistence disabled for simplicity». Для локального dev-кластера это оправдано: данные живут в `emptyDir` и теряются при рестарте пода. В production необходимо использовать `PersistentVolumeClaim` или managed PostgreSQL (RDS, Cloud SQL).