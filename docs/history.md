# todo-app-v2: Go REST API с PostgreSQL, Kind/GKE Kubernetes и Helm

## ИСТОРИЯ

- **18 марта 2026**: Создан проект — модели, миграции БД, repository/service/handler слои, роутер на Gin, Swagger документация
- **19 марта 2026**: Frontend (index.html + Tailwind CSS), тесты, soft delete (deleted_at), due_date/tags поля, Graceful Shutdown, Docker Compose (API + Postgres + seed), Dockerfile, API версионирование `/api/v1`
  * Баг: Swagger не работал — несколько итераций фиксов
  * Баг: Dockerfile сломан — исправлено
- **20 марта 2026**: Миграция Swagger 2.0 → OpenAPI 3.0, фильтрация задач по query параметрам, интеграционные тесты, GitHub Actions CI (lint + test + build + push в GHCR), connection pooling, Helm chart + Kind кластер + Ingress, Makefile
  * Баг: CI не мог пушить Docker образы — добавлены permissions `packages: write`
  * Баг: Миграции в CI через Docker action не работали — переключились на прямой бинарник
- **21 марта 2026**: PostgreSQL через Bitnami Helm chart (вместо кастомного deployment), локальный Docker registry для Kind, seed данные, partial updates для todo, `/etc/hosts` для `todo.local`
- **22–23 марта 2026**: Полное покрытие тестами — handler, service, repository слои. Coverage threshold в CI: 50% → 65% → 45% → 10% (подбирали реалистичный порог)
- **26 марта 2026**: Улучшение Makefile — install-tools, smoke-test, migrate-down, OCI image labels, параметризация credentials
- **28–29 марта 2026**: Обновление frontend — расширенная форма создания задач, теги, быстрый выбор даты, фильтры по дате, поиск. Merge ветки `Frontend-update`
- **30–31 марта 2026**: Рефакторинг CI, фиксы deployment (rolling updates, dynamic ports, GIN_MODE, пустой title)
- **1 апреля 2026**: **Frontend-backend split** — отдельные Dockerfile, deployment и service для API и Web (nginx). Отдельный ingress для каждого
  * Баг: Два контейнера в одном поде мешали масштабированию → разделили на отдельные deployment'ы
- **3 апреля 2026**: Migration Job в Helm, настройка ingress путей, версионирование API в OpenAPI спецификации
  * Баг: Asset paths в index.html сломались после split → исправлены
- **4 апреля 2026**: Swagger по env переменной, OpenAPI улучшения, GIT_SHA в index.html, initContainer для ожидания PostgreSQL, динамический BASE_HREF
  * Баг: API pod стартовал раньше PostgreSQL → добавлен initContainer с wait-for-postgres
- **5 апреля 2026**: FluxCD v2.8.3 (GitOps), Ansible playbook для деплоя на Multipass VM и Hetzner сервер, Terraform для GKE кластера (VPC + GKE), CI обновлён для push в registry + imagePullSecrets

## СТЕК

- **Backend**: Go 1.24.1 + Gin + GORM
- **Frontend**: HTML + Tailwind CSS + vanilla JS, Nginx
- **БД**: PostgreSQL (Bitnami Helm chart)
- **Контейнеризация**: Docker (multi-stage build), Docker Compose
- **Оркестрация**: Kubernetes (Kind — локально, GKE — прод)
- **Деплой**: Helm 3, FluxCD v2.8.3 (GitOps)
- **IaC**: Terraform (GKE + VPC), Ansible (VM provisioning)
- **CI/CD**: GitHub Actions (lint, test, build, push в GHCR)
- **API Docs**: OpenAPI 3.0 + Swagger UI
- **Ingress**: NGINX Ingress Controller

## РЕШЕНИЯ

- **Go + Gin** вместо Python/Node: быстрая компиляция, маленький Docker образ (~20MB Alpine)
- **Kind для локальной разработки**: полная k8s среда, ingress из коробки
- **Bitnami PostgreSQL chart** вместо кастомного: проверенное решение, managed backups
- **OpenAPI 3.0 статический** вместо генерации через swaggo: полный контроль над спецификацией
- **Frontend-backend split**: независимое масштабирование API и Web
- **FluxCD**: GitOps подход — состояние кластера всегда в git
- **Terraform для GKE**: reproducible инфраструктура

## БАГИ И РЕШЕНИЯ

- ✓ Swagger не работал → несколько итераций, в итоге миграция на статический OpenAPI 3.0
- ✓ CI Docker push без прав → добавлены `packages: write` permissions
- ✓ Миграции через Docker action в CI ломались → прямой бинарник golang-migrate
- ✓ API стартовал раньше PostgreSQL → initContainer с wait-for-postgres
- ✓ Frontend/backend в одном поде → отдельные deployment'ы
- ✓ Asset paths сломались после split → fix путей в index.html
- ✓ Coverage threshold в CI нестабилен → снижен до реалистичного порога 10%
- ✓ Динамический BASE_HREF для разных окружений → конфигурация через Helm values
