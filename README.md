# Todo App v2

Простое приложение для управления задачами.

## Запуск проекта

### 1. Запуск базы данных

Приложение использует PostgreSQL. Для запуска локальной базы данных выполните:

```bash
docker-compose up -d
```

Это запустит контейнер `todo-db` на порту `5432`.

### 2. Применение миграций

Для создания структуры таблиц в базе данных используйте `golang-migrate` через Docker (не требует локальной установки):

```bash
docker run --rm -v $(pwd)/migrations:/migrations --network todo-app-v2_default migrate/migrate -path=/migrations/ -database "postgres://postgres:postgres@todo-db:5432/todo_db?sslmode=disable" up
```

### 3. Запуск приложения

```bash
go run cmd/api/main.go
```

Приложение будет доступно по адресу `http://localhost:8080`.
### 4. Swagger документация

- **Просмотр**: `http://localhost:8080/docs/index.html` (после запуска приложения)
- **Обновление**: если вы изменили аннотации или модели данных, обновите документацию командой:
  ```bash
  swag init -g cmd/api/main.go -d ./
  ```
