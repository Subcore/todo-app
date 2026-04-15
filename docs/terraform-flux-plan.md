# Plan: FluxCD GitOps на существующем GKE-кластере

## Context

Инфраструктура уже создана в Terraform:
- GCS bucket для state
- VPC + subnet с secondary ranges (pods/services)
- GKE-кластер с одним spot-pool (e2-medium)

Helm-чарт (`deploy/helm/todo-app/`) уже содержит всё для деплоя:
- deployment-api, deployment-web, services, ingress, configmap, secret, job-migrate
- Зависимость: Bitnami PostgreSQL

**Задача**: установить FluxCD в кластер, чтобы он развернул приложение из Helm-чарта автоматически.

```
Terraform (готово)              Flux (нужно добавить)
──────────────────              ─────────────────────
VPC + Subnet                    flux bootstrap → кластер
GKE + 1 spot pool                 ↓
GCS bucket (state)              HelmRelease (ingress-nginx)
Artifact Registry                 ↓
                                HelmRelease (todo-app) → ./deploy/helm/todo-app
                                  ↓
                                API + Web + PostgreSQL + Ingress + Migrations
```

---

## Часть 1: Artifact Registry в Terraform

**Файл: `terraform/main.tf`** — добавить в конец:

```hcl
resource "google_artifact_registry_repository" "todo" {
  location      = var.region
  repository_id = "todo"
  format        = "DOCKER"
}
```

**Файл: `terraform/outputs.tf`** — добавить:

```hcl
output "registry_url" {
  value = "${var.region}-docker.pkg.dev/${var.project_id}/todo"
}
```

---

## Часть 2: Flux Bootstrap через Terraform

### 2.1 Провайдеры — `terraform/versions.tf`

Добавить `kubernetes` и `flux` провайдеры:

```hcl
terraform {
  required_version = ">= 1.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.35"
    }
    flux = {
      source  = "fluxcd/flux"
      version = "~> 1.4"
    }
  }
}
```

### 2.2 Переменные — `terraform/variables.tf`

Добавить:

```hcl
variable "github_token" {
  type        = string
  sensitive   = true
  description = "GitHub PAT with repo permissions"
}

variable "github_org" {
  type        = string
  description = "GitHub username or org"
}

variable "github_repository" {
  type        = string
  default     = "todo-app-v2"
  description = "GitHub repository name"
}
```

### 2.3 Новый файл — `terraform/flux.tf`

```hcl
# ── Подключение к GKE ──
data "google_client_config" "default" {}

provider "kubernetes" {
  host                   = "https://${module.gke.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(module.gke.ca_certificate)
}

provider "flux" {
  kubernetes = {
    host                   = "https://${module.gke.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(module.gke.ca_certificate)
  }
  git = {
    url = "https://github.com/${var.github_org}/${var.github_repository}.git"
    http = {
      username = "git"
      password = var.github_token
    }
  }
}

# ── Bootstrap Flux ──
resource "flux_bootstrap_git" "this" {
  depends_on = [module.gke]
  path       = "clusters/gke"
}
```

Что происходит при `terraform apply`:
1. Flux controllers устанавливаются в namespace `flux-system`
2. Flux создаёт `clusters/gke/flux-system/` в git-репозитории (свои манифесты)
3. Flux начинает следить за `clusters/gke/` — любые yaml-файлы там будут применены

### 2.4 `terraform/terraform.tfvars`

```hcl
project_id = "project-49ca1a08-b981-4f16-994"

github_org        = "твой-github-username"
# github_token передавать через переменную окружения:
# export TF_VAR_github_token="ghp_xxxxx"
```

> **Важно**: `terraform.tfvars` уже в `.gitignore` — секреты не попадут в git.
> Но `github_token` лучше передавать через `TF_VAR_github_token` env var, а не хранить в файле.

---

## Часть 3: Flux-манифесты в `clusters/gke/`

После bootstrap Flux следит за этой папкой. Добавляем 3 файла.

### 3.1 `clusters/gke/kustomization.yaml`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ingress-nginx.yaml
  - todo-app.yaml
```

### 3.2 `clusters/gke/ingress-nginx.yaml`

NGINX Ingress Controller — нужен потому что Helm-чарт создаёт Ingress ресурсы
с `ingressClassName: nginx`, а контроллер в GKE не предустановлен (в отличие от kind).

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: ingress-nginx
  namespace: flux-system
spec:
  interval: 1h
  url: https://kubernetes.github.io/ingress-nginx
---
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: ingress-nginx
  namespace: flux-system
spec:
  targetNamespace: ingress-nginx
  install:
    createNamespace: true
  interval: 10m
  chart:
    spec:
      chart: ingress-nginx
      version: "4.*"
      sourceRef:
        kind: HelmRepository
        name: ingress-nginx
        namespace: flux-system
  values:
    controller:
      service:
        type: LoadBalancer
```

### 3.3 `clusters/gke/todo-app.yaml`

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: todo-app
  namespace: flux-system
spec:
  targetNamespace: default
  interval: 5m
  chart:
    spec:
      chart: ./deploy/helm/todo-app
      sourceRef:
        kind: GitRepository
        name: flux-system
        namespace: flux-system
  values:
    image:
      repository: asia-southeast1-docker.pkg.dev/project-49ca1a08-b981-4f16-994/todo/todo-api
      tag: latest
    web:
      image:
        repository: asia-southeast1-docker.pkg.dev/project-49ca1a08-b981-4f16-994/todo/todo-web
        tag: latest
    ingress:
      enabled: true
      className: nginx
      host: ""
    postgresql:
      primary:
        persistence:
          enabled: true
          size: 5Gi
```

Все остальные values (ports, resources, db credentials) берутся из дефолтного
`deploy/helm/todo-app/values.yaml` — переопределяем только то, что отличается для GKE.

---

## Часть 4: Вендоринг Helm-зависимости

**Проблема**: Helm-чарт зависит от Bitnami PostgreSQL (`Chart.yaml` → dependencies).
Flux **не запускает** `helm dependency build` — он ожидает что зависимости доступны.

**Решение**: закоммитить скачанный `.tgz` в git.

Сейчас в `.gitignore`:
```
deploy/helm/todo-app/charts/*.tgz
```

Нужно убрать эту строку и закоммитить `charts/postgresql-18.5.11.tgz`:

```bash
# Скачать зависимость
cd deploy/helm/todo-app && helm dependency build

# Убрать из gitignore
# (удалить строку "deploy/helm/todo-app/charts/*.tgz" из .gitignore)

# Закоммитить
git add charts/postgresql-18.5.11.tgz
git commit -m "vendor: add postgresql helm dependency for Flux"
```

---

## Часть 5: Makefile

Добавить в секцию `# ─── Terraform GCP ───`:

```makefile
GCR_REPO = $(shell cd terraform && terraform output -raw registry_url 2>/dev/null)

tf-flux: ## Bootstrap Flux GitOps on GKE cluster
	cd terraform && terraform apply -target=flux_bootstrap_git.this

gke-build-push: ## Build and push images to Artifact Registry
	docker build -t $(GCR_REPO)/todo-api:$(IMAGE_TAG) .
	docker build -f Dockerfile.web -t $(GCR_REPO)/todo-web:$(IMAGE_TAG) --build-arg GIT_SHA=$(GIT_SHA) .
	docker push $(GCR_REPO)/todo-api:$(IMAGE_TAG)
	docker push $(GCR_REPO)/todo-web:$(IMAGE_TAG)
```

---

## Порядок запуска

```bash
# 1. Инфраструктура (VPC + GKE + Artifact Registry)
make tf-apply

# 2. Настроить kubectl
make tf-kubeconfig

# 3. Настроить docker auth для Artifact Registry
gcloud auth configure-docker asia-southeast1-docker.pkg.dev

# 4. Собрать и запушить образы
make gke-build-push

# 5. Вендоринг Helm-зависимости (один раз)
cd deploy/helm/todo-app && helm dependency build
# убрать charts/*.tgz из .gitignore, закоммитить

# 6. Закоммитить и запушить Flux-манифесты (clusters/gke/*.yaml)
git add clusters/gke/ && git commit -m "add flux manifests" && git push

# 7. Bootstrap Flux (установит контроллеры + подключит к git)
make tf-flux

# 8. Готово — Flux подхватит манифесты и задеплоит:
#    - NGINX Ingress Controller (LoadBalancer с внешним IP)
#    - todo-app (API + Web + PostgreSQL + Ingress + Migrations)
```

После этого GitOps-цикл:
```
git push (изменения в Helm values или код)
  → make gke-build-push (новый образ в Artifact Registry)
    → обновить тег в clusters/gke/todo-app.yaml
      → git push
        → Flux видит изменение → передеплоит
```

---

## Архитектура

```
┌─────────────────────────────────────────────────────┐
│              GKE Cluster (1 spot pool)               │
│                                                      │
│  ┌── flux-system (namespace) ────────────────────┐  │
│  │  Source controller   — следит за git repo      │  │
│  │  Kustomize controller — применяет manifests    │  │
│  │  Helm controller     — ставит HelmRelease'ы    │  │
│  └───────────────────────────────────────────────┘  │
│                                                      │
│  ┌── ingress-nginx (namespace) ──────────────────┐  │
│  │  NGINX Ingress Controller                      │  │
│  │  LoadBalancer :80/:443 (внешний IP от GCP)     │  │
│  └──────────────┬────────────────────────────────┘  │
│                  │                                    │
│        ┌─────────┴──────────┐                        │
│        │                    │                        │
│  /api(/|$)(.*)        /web(/|$)(.*)                  │
│        │                    │                        │
│  ┌── default (namespace) ───┼────────────────────┐  │
│  │     ┌──────────┐   ┌────┴───────┐            │  │
│  │     │ todo-api  │   │ todo-web   │            │  │
│  │     │ :8080     │   │ :80       │            │  │
│  │     └─────┬─────┘   └───────────┘            │  │
│  │           │                                   │  │
│  │     ┌─────┴──────────┐                        │  │
│  │     │ todo-postgresql │                        │  │
│  │     │ :5432           │                        │  │
│  │     └────────────────┘                        │  │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

---

## Файлы для изменения (сводка)

| Действие | Файл | Часть |
|----------|------|-------|
| Изменить | `terraform/main.tf` (добавить Artifact Registry) | 1 |
| Изменить | `terraform/outputs.tf` (добавить registry_url) | 1 |
| Изменить | `terraform/versions.tf` (добавить kubernetes, flux провайдеры) | 2.1 |
| Изменить | `terraform/variables.tf` (добавить github_token, github_org, github_repository) | 2.2 |
| Создать  | `terraform/flux.tf` (providers + bootstrap) | 2.3 |
| Изменить | `terraform/terraform.tfvars` (добавить github_org) | 2.4 |
| Создать  | `clusters/gke/kustomization.yaml` | 3.1 |
| Создать  | `clusters/gke/ingress-nginx.yaml` | 3.2 |
| Создать  | `clusters/gke/todo-app.yaml` | 3.3 |
| Изменить | `.gitignore` (убрать charts/*.tgz) | 4 |
| Изменить | `Makefile` (добавить tf-flux, gke-build-push) | 5 |
