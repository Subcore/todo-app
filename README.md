# Todo App v2

Простое приложение для управления задачами.

## Запуск проекта

### 1. Запуск всей инфраструктуры (БД и API)

Приложение и база данных настроены для работы в Docker. Для запуска выполните:

```bash
docker-compose up --build -d
```

Это запустит:
- Контейнер `todo-db` на порту `5432`.
- Контейнер `todo-api` на порту `8080`.

Приложение будет доступно по адресу `http://localhost:8080`.
### 4. Swagger документация

- **Просмотр**: `http://localhost:8080/docs/index.html` (после запуска приложения)
- **Обновление**: если вы изменили аннотации или модели данных, обновите документацию командой:
  ```bash
  swag init -g cmd/api/main.go -d ./
  ```

---

## Architecture & Tradeoffs

### Language: Go

Compiled binary with a minimal Docker image (~15 MB on Alpine), goroutines for concurrency, and strict static typing. The main tradeoff is verbose error handling — every call returns an explicit `error` — but this keeps control flow transparent. Heavy ORM abstractions powered by generics were deliberately avoided to keep the codebase readable.

### Database: PostgreSQL

Chosen for relational constraints, ACID transactions, native `text[]` arrays (used for tags), and `ILIKE` for case-insensitive full-text search. The tradeoff is that a relational database adds operational overhead for a simple todo app, but it gives a solid foundation for future features (multi-user, relations, analytics) without a schema rewrite.

### ORM: GORM

GORM enables fast bootstrap: auto-migration, built-in soft delete, and struct-based query building reduce boilerplate significantly. The tradeoffs are implicit query generation (harder to spot N+1 issues) and reduced control over raw SQL. For an MVP this is acceptable — performance-critical paths can be replaced with raw queries later without touching the rest of the codebase.

### Migrations: golang-migrate

SQL-first approach that is not coupled to the ORM. Up/down migration files make rollbacks explicit and integrate cleanly into CI pipelines. GORM `AutoMigrate` is used only in local development for convenience; `golang-migrate` is the production-safe path.

### Frontend: Alpine.js + Tailwind CSS

A lightweight JS framework served as static files embedded directly in the Go binary — zero separate build step, zero Node.js dependency in production. The tradeoff is that Alpine.js does not scale well to complex client-side state, but for a demo todo app with straightforward CRUD interactions it is the right tool: minimal bundle, no framework overhead, instant load.
