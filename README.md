# Todo App v2

Простое приложение для управления задачами — Go API + PostgreSQL.

---

## Prerequisites

| Tool | Версия | Зачем |
|------|--------|-------|
| Go | 1.24+ | сборка и запуск API |
| Docker | 24+ | контейнеризация |
| kind | 0.20+ | локальный Kubernetes-кластер |
| kubectl | 1.28+ | управление кластером |
| Helm | 3.x | деплой чартов |
| golang-migrate | 4.x | CLI для миграций БД |

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

Приложение доступно по адресу **http://localhost:8080**.

---

## Quick start (Kubernetes)

```bash
make deploy-kind
```

Команда выполняет полный пайплайн: создание kind-кластера, сборку Docker-образа, установку Ingress Controller, деплой PostgreSQL через Helm, прогон миграций и установку приложения.

Добавьте запись в `/etc/hosts`:

```
127.0.0.1  todo.local
```

Приложение доступно по адресу **http://todo.local**.

---

## Migrations

Миграции хранятся в каталоге `migrations/` в формате SQL (up/down).

- **Docker Compose** — сервис `migrate` запускается автоматически после старта БД и применяет все pending-миграции.
- **Kubernetes (kind)** — выполните `make migrate`. Под капотом используется `kubectl port-forward` к поду PostgreSQL и CLI `golang-migrate`.

---

## Seed data

Тестовые данные находятся в `seeds/seed.sql` и содержат три примера задач.

- **Docker Compose** — сервис `seed` загружает данные автоматически после миграций.
- **Вручную** — подключитесь к БД и выполните:
  ```bash
  psql -h localhost -U postgres -d todo_db -f seeds/seed.sql
  ```

---

## Swagger UI

| Окружение | URL |
|-----------|-----|
| Docker Compose | http://localhost:8080/docs/index.html |
| Kubernetes | http://todo.local/docs/index.html |

Если вы изменили аннотации или модели, обновите документацию:

```bash
swag init -g cmd/api/main.go -d ./
```

---

## Troubleshooting

### Pod в CrashLoopBackOff

Чаще всего — PostgreSQL ещё не готов. Проверьте логи:

```bash
kubectl logs deploy/todo-app
kubectl logs sts/todo-postgresql
```

API использует health-проверки (`/healthz`, `/readyz`), поэтому Kubernetes перезапустит под автоматически, когда БД станет доступна.

### ErrImagePull / ImagePullBackOff

Образ `todo-api:v1` собирается локально и загружается в kind через `kind load docker-image`. Убедитесь, что `imagePullPolicy` установлен в `IfNotPresent` (значение по умолчанию в `values.yaml`).

### Port already in use

Порт `8080` (Compose) или `80/443` (kind Ingress) уже заняты:

```bash
lsof -i :8080
# или для kind
lsof -i :80
```

Завершите процесс или измените порт в `.env` / `kind-config.yaml`.

---

## Architecture & Tradeoffs

### Почему PostgreSQL

Реляционные constraints, ACID-транзакции и типизированные массивы `text[]` (используются для тегов) делают Postgres естественным выбором. `ILIKE` покрывает потребности в полнотекстовом поиске без внешних зависимостей. MongoDB дал бы schemaless-гибкость, но для структурированных данных с фильтрами и связями SQL удобнее — не нужно дублировать логику валидации на стороне приложения.

### Почему GORM

Быстрый старт: auto-migrate для dev-окружения, soft delete из коробки, встроенный connection pool management. Минус — магия и неоптимальные запросы на сложных кейсах (implicit query generation затрудняет обнаружение N+1). `sqlc` дал бы type-safe SQL без рантайм-рефлексии, но потребовал бы значительно больше boilerplate для базового CRUD. Для MVP компромисс оправдан — критичные пути можно заменить на raw SQL без переписывания остального кода.

### Почему Gin

Production-ready роутер со встроенным recovery middleware, structured logging и знакомым Express-like API. `chi` или стандартный `net/http` тоже подошли бы, но Gin быстрее для прототипирования: меньше кода на обвязку middleware и маршрутизацию.