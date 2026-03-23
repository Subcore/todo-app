---
description: Чиним тесты
---

Контекст: Go + Gin + GORM + PostgreSQL Todo App.
Существующие файлы: [список файлов из шага]
Задача: [описание из шага]
Требования:
- [конкретные тесты из шага]
- Использовать testify/assert + testify/require
- Все интеграционные тесты: skip если нет TEST_DB_DSN
- Cleanup: TRUNCATE todos RESTART IDENTITY CASCADE
- Проверить: go test ./... -race
- пиши план на русском языке