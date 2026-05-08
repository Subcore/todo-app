# Развёртывание todo-app в GKE

Пошаговая инструкция для нового GCP-проекта `<your-project-id>`.

## Что будет создано

| Ресурс | Описание |
|---|---|
| GCS bucket | `<your-project-id>-tfstate` — хранилище Terraform state |
| VPC + Subnet | `todo-vpc`, subnet `gke-subnet` (10.0.0.0/20) + secondary ranges для pods/services |
| GKE кластер | `todo-cluster`, зональный (asia-southeast1-b), spot-ноды e2-medium |
| Static IP | `todo-ingress-ip` — внешний адрес для Ingress |
| FluxCD | GitOps — автоматический деплой из ветки `main` |
| PostgreSQL | Bitnami Helm chart внутри кластера |
| Todo App | API (Go) + Web (Nginx) через Helm chart |

---

## Предварительные требования

```bash
# Убедись что установлены:
terraform --version   # >= 1.6
gcloud --version      # Google Cloud SDK
kubectl version       # Kubernetes CLI
flux --version        # FluxCD CLI (опционально, для дебага)
```

---

## Шаг 0. Аутентификация в GCP

```bash
gcloud auth login
gcloud auth application-default login
gcloud config set project <your-project-id>
```

---

## Шаг 1. Включить необходимые GCP API

```bash
gcloud services enable \
  storage.googleapis.com \
  compute.googleapis.com \
  container.googleapis.com \
  --project=<your-project-id>
```

> Без этого шага Terraform упадёт с ошибкой `googleapi: Error 403: ... has not been used in project`.
> API включаются 1-2 минуты — подожди перед следующим шагом.

---

## Шаг 2. Создать GCS bucket для Terraform state (bootstrap)

```bash
cd terraform/bootstrap

# Удалить старый state если есть (от предыдущего проекта)
rm -f terraform.tfstate terraform.tfstate.backup

terraform init -upgrade
terraform apply
```

Terraform создаст bucket `<your-project-id>-tfstate` в `asia-southeast1`.

**Проверка:**
```bash
gcloud storage ls --project=<your-project-id>
# Должен вывести: gs://<your-project-id>-tfstate/
```

---

## Шаг 3. Подготовить переменные

Тебе нужен GitHub Personal Access Token (PAT) с правами:
- `repo` (полный доступ к репозиторию)
- `admin:public_key` (для deploy key FluxCD)

Создать: https://github.com/settings/tokens

```bash
cd ../   # вернуться в terraform/

# Создать файл с секретными переменными (уже в .gitignore)
cat > terraform.tfvars <<'EOF'
project_id         = "<your-project-id>"
github_owner       = "Subcore"
github_repository  = "todo-app"
github_token       = "ghp_XXXXXXXXXXXXXXXXXXXX"
db_password        = "ПРИДУМАЙ_НАДЁЖНЫЙ_ПАРОЛЬ"
EOF
```

> **Важно:** `terraform.tfvars` в `.gitignore` — он не попадёт в git. Не коммить его.

---

## Шаг 4. Развернуть инфраструктуру

```bash
cd terraform/

# Удалить старый lock если меняли версию провайдера
rm -f .terraform.lock.hcl

terraform init -backend-config="bucket=<your-project-id>-tfstate"
```

> `-backend-config="bucket=..."` — подставляет имя bucket в `backend "gcs"`.

```bash
terraform plan
```

Убедись, что plan показывает создание:
- `module.vpc` — VPC + subnet
- `module.gke` — GKE кластер
- `google_compute_address.ingress` — static IP
- `flux_bootstrap_git.this` — FluxCD
- `kubernetes_namespace.*` — namespaces
- `kubernetes_secret.*` — secrets
- `kubernetes_config_map.*` — configmaps

```bash
terraform apply
```

> **Время выполнения:** 10-15 минут. Основное время — создание GKE кластера.
>
> **Если упало:** чаще всего причина — timeout при создании GKE или GitHub token без нужных прав. Просто повтори `terraform apply` — Terraform продолжит с того места, где остановился.

---

## Шаг 5. Подключиться к кластеру

```bash
# Terraform выводит готовую команду:
terraform output kubeconfig_command
# Скопировать и выполнить, или:

gcloud container clusters get-credentials todo-cluster \
  --zone asia-southeast1-b \
  --project <your-project-id>
```

---

## Шаг 6. Проверить что всё поднялось

```bash
# Flux контроллеры
kubectl get pods -n flux-system
# Должно быть 5-6 подов в Running (source, kustomize, helm, image-reflector, image-automation, notification)

# Helm releases
kubectl get helmrelease -A
# todo-postgresql   — должен быть Ready
# todo-app          — должен быть Ready (после postgresql)
# ingress-nginx     — должен быть Ready

# Поды приложения
kubectl get pods -n todo-app
# todo-postgresql-0         — Running
# todo-app-api-*            — Running
# todo-app-web-*            — Running

# Ingress и внешний IP
kubectl get svc -n ingress-nginx
# ingress-nginx-controller   LoadBalancer   <EXTERNAL-IP>

# Или через terraform:
terraform output ingress_static_ip
```

---

## Шаг 7. Открыть приложение

```bash
# Узнать IP
IP=$(terraform output -raw ingress_static_ip)
echo "http://${IP}.nip.io/web/"
echo "http://${IP}.nip.io/api/todos"
```

Открой в браузере:
- `http://<IP>.nip.io/web/` — фронтенд
- `http://<IP>.nip.io/api/todos` — API

---

## Типичные проблемы и решения

### `Error 403: ... API has not been used in project`
```bash
gcloud services enable compute.googleapis.com container.googleapis.com storage.googleapis.com
# Подожди 2 минуты, повтори terraform apply
```

### Flux не может склонировать репозиторий
```bash
flux logs -n flux-system
# Если "authentication required" — проверь что github_token валидный и имеет scope `repo`
```

### HelmRelease todo-app в статусе `not ready`
```bash
kubectl describe helmrelease todo-app -n todo-app
kubectl get events -n todo-app --sort-by='.lastTimestamp'
# Чаще всего — PostgreSQL ещё не готов. Подожди 2-3 минуты.
```

### Pods в ImagePullBackOff
```bash
kubectl describe pod <pod-name> -n todo-app
# Если "unauthorized" — проверь что ghcr-secret создан:
kubectl get secret ghcr-secret -n todo-app
# И что образы существуют в GitHub Packages (ghcr.io)
```

### ConfigMap ingress-nginx-values не создаётся
```bash
kubectl get ns ingress-nginx
# Если namespace нет — подожди, Terraform создаст его. Повтори terraform apply.
```

### Миграция БД упала
```bash
kubectl get jobs -n todo-app
kubectl logs job/todo-app-migrate -n todo-app -c migrate
# Job сохраняется при ошибке (hook-failed policy) — можно прочитать логи
```

---

## Удаление всего

```bash
# 1. Удалить инфраструктуру
cd terraform/
terraform destroy

# 2. Удалить bucket (bootstrap)
cd bootstrap/
terraform destroy

# Или одной командой через gcloud:
# gcloud projects delete <your-project-id>
```

---

## Схема зависимостей

```
terraform/bootstrap (bucket)
        │
        ▼
terraform/ (init -backend-config=bucket)
        │
        ├── VPC + Subnet
        │       │
        │       ▼
        ├── GKE Cluster ──────────────────┐
        │       │                         │
        │       ▼                         ▼
        ├── Namespaces              FluxCD Bootstrap
        │   (todo-app,                    │
        │    ingress-nginx)               ▼
        │       │                   Kustomization
        │       ▼                   (k8s/cluster/)
        ├── Secrets                       │
        │   (ghcr, db-creds)              ├── ingress-nginx HelmRelease
        │                                 ├── PostgreSQL HelmRelease
        ├── ConfigMaps                    └── todo-app HelmRelease
        │   (ingress IP,                        │
        │    app ingress host)                  ├── API Deployment
        │                                       ├── Web Deployment
        └── Static IP                           └── Migration Job
```
