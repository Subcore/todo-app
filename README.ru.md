# Todo App v2

Небольшой REST-сервис для задач на Go, оформленный как портфолио-проект, который проходит весь путь от локальной разработки до продакшена. Один и тот же код разворачивается четырьмя способами: Docker Compose для быстрой итерации, локальный Kind-кластер для практики с Kubernetes, кластер GKE, который провижится через Terraform и обслуживается FluxCD по GitOps-модели, и одно-нодовый VPS-деплой через Ansible.

[![CI](https://github.com/Subcore/todo-app-v2/actions/workflows/ci.yml/badge.svg)](.github/workflows/ci.yml)
![Go](https://img.shields.io/badge/go-1.24.1-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-blue)

[English version](README.md)

---

## Стек технологий

| Слой | Инструмент |
|---|---|
| Язык / рантайм | Go 1.24.1 |
| HTTP-фреймворк | Gin |
| ORM | GORM |
| База данных | PostgreSQL 16.2 |
| Миграции | [golang-migrate](https://github.com/golang-migrate/migrate) v4.19.1 |
| Фронтенд | Статический HTML + Alpine.js + Tailwind CSS |
| Спецификация API | OpenAPI 3.0 (Swagger UI на `/docs`) |
| Контейнеры | Docker, Docker Compose |
| Локальный Kubernetes | Kind v0.31.0 |
| Kubernetes-инструменты | kubectl v1.35.3, Helm v4.1.3 |
| Ingress | NGINX Ingress v1.12.1 |
| Облачная инфраструктура (IaC) | Terraform (GCP / GKE) |
| GitOps | FluxCD (Helm + image automation) |
| Деплой на VPS | Ansible |
| CI | GitHub Actions, GHCR |

---

## Архитектура

End-to-end GitOps-процесс для деплоя в GKE:

```
 ┌────────────┐    push    ┌──────────────────┐   build & push   ┌─────────┐
 │ Разработчик│ ─────────▶ │  GitHub Actions  │ ───────────────▶ │  GHCR   │
 └────────────┘            │   (CI workflow)  │                  └────┬────┘
                           └──────────────────┘                       │
                                                                      │ образ
                                                                      ▼
 ┌──────────────┐ синхрон.  ┌─────────────┐  HelmRelease   ┌─────────────────┐
 │ k8s/cluster/ │ ◀───────  │   FluxCD    │ ─────────────▶ │  GKE (todo-app) │
 │ (манифесты)  │ git pull  │  в кластере │                │  Postgres + API │
 └──────┬───────┘           └─────┬───────┘                │   + web + nginx │
        ▲                         │                        └─────────────────┘
        │ patch HelmRelease       │ image automation
        └─────────────────────────┘  находит новый тег в GHCR
```

- Код мержится в `main` → CI собирает два образа (`todo-api`, `todo-web`) и пушит их в `ghcr.io`.
- Контроллеры image-reflector + image-automation у Flux следят за GHCR, находят новый тег по шаблону `^main-<timestamp>-<sha>$` и переписывают [k8s/cluster/todo-app/helmrelease.yaml](k8s/cluster/todo-app/helmrelease.yaml) в `main`.
- Flux синхронизирует новый коммит и раскатывает обновлённый HelmRelease в кластере.

---

## Quick Start

Выберите один из четырёх способов ниже.

### Вариант 1 — Docker Compose (локальная разработка)

Самый быстрый способ запустить приложение на ноутбуке. Поднимает PostgreSQL, прогоняет миграции, затем стартует API и web-контейнер.

```bash
cp .env.example .env
docker compose up --build -d
```

| URL | Что |
|---|---|
| http://localhost:8080 | API (`/api/v1/todos`, `/healthz`, `/readyz`) |
| http://localhost:8080/docs | Swagger UI |
| http://localhost | Статический фронтенд (отдельный контейнер `todo-web`) |

Полезные команды:

```bash
docker compose logs -f api          # читать логи API
docker compose --profile seed up seed   # загрузить seeds/seed.sql (3 примера задач)
docker compose down                 # остановить контейнеры
docker compose down -v              # остановить + удалить том postgres
```

---

### Вариант 2 — Локальный Kubernetes (Kind)

Самодостаточный локальный кластер: Kind + локальный Docker registry, NGINX Ingress, PostgreSQL через bitnami Helm-чарт и наш собственный Helm-чарт для приложения.

```bash
make install-tools    # установит kind, kubectl, helm, golang-migrate
make deploy-kind      # одна команда — 11 шагов
```

Что делает `make deploy-kind`:

1. Поднимает локальный Docker registry на `localhost:5000`
2. Создаёт Kind-кластер (control-plane + worker)
3. Подключает registry к сети Kind
4. Настраивает containerd на каждой ноде, чтобы он тянул из этого registry
5. Устанавливает NGINX Ingress Controller
6. Собирает и пушит образы API + web
7. Деплоит PostgreSQL (bitnami) и приложение через Helm
8. Прогоняет миграции
9. Добавляет `todo.local` в `/etc/hosts`
10. Запускает smoke-тест по `/api/v1/healthz`

После завершения:

| URL | Что |
|---|---|
| http://todo.local/web | Фронтенд |
| http://todo.local/api | API |
| http://todo.local/api/docs | Swagger UI |

Траблшутинг:

- **`CrashLoopBackOff`** — обычно Postgres ещё не готов. Смотрите `kubectl logs deploy/todo` и `kubectl logs sts/todo-postgresql`. Если видите `password authentication failed` — пересоздайте: `make delete-cluster && make deploy-kind`.
- **`ErrImagePull` / `ImagePullBackOff`** — локальный registry не запущен: `make create-registry`.
- **`port already in use`** — `lsof -i :8080` (Compose) или `lsof -i :80` (Kind). Либо завершите процесс, либо поменяйте порт в `.env` / [kind-config.yaml](kind-config.yaml).
- **Compose и Kind работают одновременно** — оба отдают Postgres на 5432. Сначала остановите Compose: `docker compose down`.

---

### Вариант 3 — GCP / GKE через Terraform + FluxCD (production)

Это основной production-вариант. Terraform создаёт GKE-кластер и бутстрапит FluxCD; дальше кластер сам синхронизируется с этим репозиторием.

**Требования:**

- CLI `gcloud` (авторизованный: `gcloud auth login && gcloud auth application-default login`)
- `kubectl`, `helm`, `terraform` ≥ 1.5
- GCP-проект с включённым биллингом
- GitHub Personal Access Token — см. Шаг 0 ниже

**Шаг 0 — создать GitHub Personal Access Token**

Terraform использует один токен для двух вещей ([terraform/flux.tf](terraform/flux.tf)):

1. Провайдер `github` создаёт deploy key в репозитории — через него Flux клонирует репо и пушит коммиты от image automation.
2. Этот же токен прописывается в Kubernetes `ghcr-secret`, чтобы кластер мог пуллить образы `ghcr.io/<owner>/todo-{api,web}`.

Создайте **classic** PAT по адресу https://github.com/settings/tokens/new со скоупами:

| Скоуп | Зачем |
|---|---|
| `repo` | Управлять deploy key репозитория |
| `read:packages` | Пуллить образы из GHCR (используется в `ghcr-secret`) |
| `write:packages` | Нужен только если этим же токеном пушите образы; CI использует `GITHUB_TOKEN`, так что обычно не требуется |

> Fine-grained PAT сейчас стабильно не работает с GHCR — используйте classic PAT.

Передавайте токен через переменную окружения, **не** через `terraform.tfvars` (файл в gitignore, но его легко слить через шаринг экрана, бэкап или случайную правку example-файла):

```bash
export TF_VAR_github_token=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

После `tf-apply` токен оказывается в двух местах:

- Terraform state в вашем GCS-бакете — держите бакет приватным.
- Kubernetes Secret `ghcr-secret` (в неймспейсах `flux-system` и `todo-app`).

Если токен утёк — отзовите его на GitHub, заново выставьте `TF_VAR_github_token` и сделайте `terraform apply`: `ghcr-secret` и deploy key обновятся на месте.

**Шаг 1 — провижим инфраструктуру через Terraform**

```bash
cp terraform/terraform.tfvars.example terraform/terraform.tfvars
# Отредактируйте terraform.tfvars: укажите project_id (region/zone — опционально).
# Секреты передавайте через env vars, в terraform.tfvars их не пишите:
export TF_VAR_db_password=$(openssl rand -hex 16)
export TF_VAR_github_owner=<ваш-gh-user-или-org>
export TF_VAR_github_repository=todo-app-v2
export TF_VAR_github_token=$GITHUB_PAT_FROM_STEP_0

make tf-bootstrap GCP_PROJECT=<ваш-project-id>   # создаёт GCS-бакет под tfstate
make tf-init      GCP_PROJECT=<ваш-project-id>
make tf-plan      GCP_PROJECT=<ваш-project-id>
make tf-apply     GCP_PROJECT=<ваш-project-id>
make tf-kubeconfig GCP_PROJECT=<ваш-project-id>  # настроить kubectl-контекст
```

Что создаёт Terraform ([terraform/main.tf](terraform/main.tf), [terraform/flux.tf](terraform/flux.tf)):

| Ресурс | Детали |
|---|---|
| VPC | `todo-vpc` с подсетью `10.10.0.0/20`, secondary ranges под pods/services |
| GKE-кластер | `todo-cluster`, зональный (`asia-southeast1-b` по умолчанию) |
| Node pool: `system-pool` | `e2-standard-2` Spot — Flux, ingress-nginx |
| Node pool: `db-pool` | `e2-small` Spot, с taint — только PostgreSQL |
| Node pool: `app-pool` | `e2-standard-4` Spot, автоскейл 1–2 — поды приложения |
| Статический IP | `todo-ingress-ip` — прокидывается в ingress-nginx через ConfigMap |
| FluxCD | бутстрап на этот репо, путь `k8s/cluster`, плюс контроллеры image-reflector и image-automation |
| Секреты | `ghcr-secret` (в `flux-system` и `todo-app`), `todo-db-credentials` |

**Шаг 2 — Flux подхватывает управление**

После `tf-apply` Flux подтягивает [k8s/cluster/](k8s/cluster/) и деплоит:

- `ingress-nginx` — [k8s/cluster/ingress-nginx/helmrelease.yaml](k8s/cluster/ingress-nginx/helmrelease.yaml)
- `todo-postgresql` (bitnami-чарт, persistent volume) — [k8s/cluster/todo-app/helmrelease.yaml](k8s/cluster/todo-app/helmrelease.yaml)
- `todo-app` (наш Helm-чарт из [deploy/helm/todo-app](deploy/helm/todo-app/))

Проверка:

```bash
kubectl get helmrelease -A
flux get all -A
flux logs -f
```

Приложение становится доступно по `http://<static-ip>.nip.io/web` (IP берётся из `kubectl -n ingress-nginx get svc ingress-nginx-controller`; Flux подставляет его в host ingress'а как `<ip>.nip.io`).

**Шаг 3 — image automation (continuous deploy)**

В [k8s/cluster/todo-app/image-automation.yaml](k8s/cluster/todo-app/image-automation.yaml) описаны две пары `ImageRepository` + `ImagePolicy` (одна для `todo-api`, другая для `todo-web`) и один `ImageUpdateAutomation`. Жизненный цикл:

1. CI на `main` собирает и пушит `ghcr.io/<owner>/todo-api:main-<ts>-<sha>` и аналогично `todo-web`.
2. image-reflector у Flux замечает новые теги в течение 5 минут.
3. Контроллер image-automation патчит маркеры `# {"$imagepolicy": ...}` в `helmrelease.yaml` и пушит коммит `chore: update images to ...` в `main`.
4. Flux синхронизирует репо, HelmRelease обновляется, новые поды раскатываются.

**Cleanup:** `make tf-destroy GCP_PROJECT=<ваш-project-id>` (сносит GKE, VPC, статический IP, deploy-key Flux — всё, что создал Terraform).

---

### Вариант 4 — Деплой на VPS (Ansible)

«Классический» деплой на одну Linux-VPS: PostgreSQL прямо на хосте, Go-бинарник как systemd-сервис, статика отдаётся через nginx, и nginx же reverse-проксирует `/api` на Go-процесс.

**Требования:**

- Linux-VPS, доступ по SSH (root или sudo), и SSH-ключ на вашей машине
- Ansible: `pip install ansible`
- Go (нужен для кросс-компиляции бинарника на вашей машине)

**Конфигурация:**

```bash
cp ansible/inventory.example ansible/inventory.ini
# Отредактируйте ansible/inventory.ini — укажите IP сервера и путь к SSH-ключу
```

[ansible/inventory.example](ansible/inventory.example):

```ini
[servers]
<YOUR_SERVER_IP> ansible_user=root ansible_port=22 ansible_ssh_private_key_file=~/.ssh/id_rsa
```

**Деплой:**

```bash
make deploy-vps
```

Эта команда запускает `make build-linux` (кросс-компилирует `cmd/api/main.go` в Linux amd64 бинарник) и затем `ansible-playbook ansible/playbook.yml`. Четыре роли в [ansible/roles/](ansible/roles/):

| Роль | Что делает |
|---|---|
| `postgresql` | Ставит PostgreSQL, создаёт БД `todo_db` и пользователя `todo` |
| `backend` | Копирует Go-бинарник в `/opt/todo-api`, создаёт systemd-юнит, включает сервис |
| `web` | Копирует `web/index.html`, `docs.html` и ассеты в `/var/www/todo` |
| `nginx` | Ставит nginx, кладёт reverse-proxy конфиг (статика + `/api` → `127.0.0.1:8080`) |

Результат: `http://<YOUR_SERVER_IP>` (фронтенд), `http://<YOUR_SERVER_IP>/api` (API), `http://<YOUR_SERVER_IP>/api/docs` (Swagger).

---

## Структура проекта

```
.
├── cmd/api/                 # main.go — точка входа Gin-сервера
├── internal/
│   ├── config/              # загрузка конфига из env
│   ├── handler/             # HTTP-хендлеры (todo, health)
│   ├── model/               # GORM-модели
│   ├── repository/          # запросы к БД
│   ├── router/              # регистрация роутов + middleware
│   └── service/             # бизнес-логика
├── migrations/              # up/down SQL-файлы golang-migrate
├── seeds/seed.sql           # идемпотентные тестовые данные
├── tests/                   # интеграционные тесты (реальный PostgreSQL)
├── web/                     # статический фронтенд (Alpine.js + Tailwind)
│
├── deploy/helm/todo-app/    # Helm-чарт, используемый Kind и FluxCD
│
├── k8s/cluster/             # манифесты, которые синхронизирует Flux в GKE
│   ├── flux-system/         # компоненты Flux (бутстрап через Terraform)
│   ├── ingress-nginx/       # HelmRelease для ingress-nginx
│   └── todo-app/            # неймспейс приложения, HelmRelease, image automation
│
├── terraform/               # инфраструктура GCP (VPC + GKE + бутстрап Flux)
│   └── bootstrap/           # one-shot модуль, который создаёт GCS-бакет под state
│
├── ansible/                 # деплой на VPS
│   ├── playbook.yml
│   ├── inventory.example
│   └── roles/{nginx,postgresql,backend,web}/
│
├── .github/workflows/ci.yml # CI: lint, OpenAPI/Helm lint, тесты, сборка и пуш образов
├── docker-compose.yaml      # стек для локальной разработки
├── Dockerfile, Dockerfile.web
├── kind-config.yaml         # описание Kind-кластера
├── openapi.yaml             # спецификация OpenAPI 3.0 (валидируется в CI)
└── Makefile                 # все entry-points для разработчика и DevOps
```

---

## Конфигурация

Переменные окружения, которые читает [cmd/api/main.go](cmd/api/main.go) через [internal/config/](internal/config/). Полный список — в [.env.example](.env.example).

| Переменная | По умолчанию | Примечания |
|---|---|---|
| `PORT` | `8080` | Порт API |
| `DB_HOST` | `localhost` | |
| `DB_PORT` | `5432` | |
| `DB_USER` | `postgres` | |
| `DB_PASSWORD` | `postgres` | Замените в не-dev окружениях |
| `DB_NAME` | `todo_db` | CI использует `todo_test` |
| `DB_SSLMODE` | `disable` | |
| `DB_MAX_IDLE_CONNS` | `10` | |
| `DB_MAX_OPEN_CONNS` | `100` | |
| `DB_CONN_MAX_LIFETIME` | `1h` | |
| `ENABLE_SWAGGER` | не задана | Установите `true`, чтобы открыть `/docs` |

Файлы-примеры по способам деплоя:

- [.env.example](.env.example) — Docker Compose
- [terraform/terraform.tfvars.example](terraform/terraform.tfvars.example) — переменные Terraform
- [ansible/inventory.example](ansible/inventory.example) — инвентарь Ansible

---

## Разработка

Часто используемые `make`-таргеты (полный список — `make help`):

| Таргет | Описание |
|---|---|
| `make test` | `go test ./...` |
| `make lint` | `golangci-lint run` + `helm lint` |
| `make run` | `docker compose up` |
| `make deploy-kind` | Полный пайплайн локального Kubernetes |
| `make migrate` | Применить миграции к Postgres в Kind |
| `make migrate-down` | Откатить последнюю миграцию |
| `make seed` | Загрузить `seeds/seed.sql` в Postgres в Kind |
| `make build-linux` | Кросс-компиляция Go-бинарника под Linux amd64 |
| `make deploy-vps` | Сборка бинарника + запуск Ansible-плейбука |
| `make tf-{bootstrap,init,plan,apply,destroy,kubeconfig}` | Жизненный цикл Terraform |
| `make clean` | Удалить Kind-кластер и registry |

### Тесты

Unit-тесты (без БД):

```bash
go test ./internal/handler/... ./internal/service/... -v -race
```

Интеграционные тесты (нужен PostgreSQL):

```bash
docker compose up db -d
PGPASSWORD=postgres psql -h localhost -U postgres -c "CREATE DATABASE todo_test;"
migrate -path migrations \
  -database "postgres://postgres:postgres@localhost:5432/todo_test?sslmode=disable" up
TEST_DB_DSN="host=localhost port=5432 user=postgres password=postgres dbname=todo_test sslmode=disable" \
  go test ./... -v -race
```

> CI работает с `todo_test`, а Docker Compose-приложение — с `todo_db`. Они не конфликтуют.

### API

REST API задокументирован в [openapi.yaml](openapi.yaml) и рендерится как Swagger UI на `/docs`, если `ENABLE_SWAGGER=true`.

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/api/v1/todos` | Список задач (фильтры: `completed`, `due_before`, `due_after`, `search`) |
| `POST` | `/api/v1/todos` | Создать задачу |
| `GET` | `/api/v1/todos/:id` | Получить одну задачу |
| `PUT` | `/api/v1/todos/:id` | Обновить задачу |
| `DELETE` | `/api/v1/todos/:id` | Мягкое удаление |
| `POST` | `/api/v1/todos/clear-completed` | Удалить все выполненные задачи |
| `GET` | `/healthz` | Liveness |
| `GET` | `/readyz` | Readiness + ping БД |

---

## CI/CD Pipeline

GitHub Actions workflow: [.github/workflows/ci.yml](.github/workflows/ci.yml). Срабатывает на push и pull request в `main`.

1. **Lint** — `golangci-lint` (errcheck, staticcheck, gosimple, unused).
2. **Валидация OpenAPI** — `@redocly/cli lint openapi.yaml`.
3. **Helm lint** — `helm lint` для [deploy/helm/todo-app](deploy/helm/todo-app/) с подгруженным субчартом bitnami.
4. **Тесты** — unit + интеграционные с реальным PostgreSQL service-контейнером и флагом `-race`.
5. **Сборка и пуш Docker-образов** — multi-stage сборки `todo-api` и `todo-web`, пуш в `ghcr.io` на `main` с тегом `main-<timestamp>-<short-sha>` (формат, который ловят image-policies Flux).

После пуша подключается image automation у Flux (см. [Вариант 3 / Шаг 3](#вариант-3--gcp--gke-через-terraform--fluxcd-production)).

---

## Cleanup

| Окружение | Команда | Что удаляет |
|---|---|---|
| Docker Compose | `docker compose down` | Останавливает контейнеры (тома сохраняются) |
| Docker Compose | `docker compose down -v` | Останавливает + удаляет том Postgres |
| Kind | `make delete-cluster` | Удаляет Kind-кластер |
| Kind | `make clean` | Кластер + локальный registry + образы |
| GKE | `make tf-destroy GCP_PROJECT=<id>` | Сносит VPC, GKE, статический IP, deploy-key Flux |
| VPS | — | Запустите `systemctl stop todo-api`, удалите `/opt/todo-api`, `/var/www/todo`, конфиг сайта nginx и (опционально) PostgreSQL — отдельного uninstall-плейбука у Ansible нет |

---

## License

[MIT](LICENSE) — пользуйтесь как референсом для своих проектов.
