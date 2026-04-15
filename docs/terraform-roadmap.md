# Terraform GCP — пошаговый roadmap

Цель: руками написать Terraform, который создаст GKE кластер в GCP.
Результат: `kubectl get nodes` показывает ноды твоего кластера.

---

## Шаг 0. Подготовка

Перед тем как писать Terraform, нужно:

1. Создать проект в GCP Console (или через `gcloud projects create`)
2. Включить billing для проекта
3. Включить нужные API:
```bash
gcloud services enable compute.googleapis.com
gcloud services enable container.googleapis.com
gcloud services enable storage.googleapis.com
```
4. Авторизоваться: `gcloud auth application-default login`

---

## Шаг 1. Bootstrap — создаём bucket для state

**Зачем:** Terraform хранит состояние (какие ресурсы создал) в файле `terraform.tfstate`. По умолчанию он лежит локально, но это плохо — потеряешь файл и Terraform забудет про все ресурсы. GCS bucket решает эту проблему. Но сам bucket нужно создать до того, как настроить remote backend — это "проблема курицы и яйца". Поэтому bootstrap — отдельная папка с локальным state.

Создай структуру:
```
terraform/
└── bootstrap/
    ├── main.tf
    └── variables.tf
```

### `terraform/bootstrap/variables.tf`

```hcl
variable "project_id" {
  type = string
}

variable "region" {
  type    = string
  default = "asia-southeast1"
}
```

**Построчно:**
- `variable "project_id"` — объявляет входную переменную. `type = string` — тип строка. Без `default` — значит обязательная, Terraform спросит при apply
- `variable "region"` — регион GCP. `default = "asia-southeast1"` — если не передашь, возьмёт это значение. Сингапур — ближайший к Таиланду регион GCP (в самом Таиланде региона нет)

### `terraform/bootstrap/main.tf`

```hcl
provider "google" {
  project = var.project_id
  region  = var.region
}

resource "google_storage_bucket" "tfstate" {
  name     = "${var.project_id}-tfstate"
  location = var.region

  versioning {
    enabled = true
  }

  uniform_bucket_level_access = true
}
```

**Построчно:**
- `provider "google"` — подключает Google Cloud провайдер. `project` и `region` — куда будем создавать ресурсы
- `var.project_id` — обращение к переменной из variables.tf
- `resource "google_storage_bucket" "tfstate"` — создаёт GCS bucket. Первый аргумент — тип ресурса, второй — локальное имя (для ссылок внутри Terraform)
- `name = "${var.project_id}-tfstate"` — имя bucket глобально уникально в GCP, поэтому добавляем project_id
- `location` — где физически хранятся данные
- `versioning { enabled = true }` — хранит предыдущие версии state файла, можно откатиться если что-то сломалось
- `uniform_bucket_level_access = true` — упрощает управление доступом (одна политика на весь bucket вместо ACL на каждый файл)

### Применяем:

```bash
cd terraform/bootstrap
terraform init      # скачивает провайдер google
terraform plan -var="project_id=MY_PROJECT"   # показывает что создаст
terraform apply -var="project_id=MY_PROJECT"  # создаёт bucket
```

---

## Шаг 2. Настраиваем remote backend

Создай файлы в `terraform/`:
```
terraform/
├── bootstrap/          # уже есть
├── versions.tf
└── backend.tf
```

### `terraform/versions.tf`

```hcl
terraform {
  required_version = ">= 1.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}
```

**Построчно:**
- `required_version = ">= 1.6"` — минимальная версия Terraform CLI
- `required_providers` — какие провайдеры нужны
- `source = "hashicorp/google"` — откуда скачивать (registry.terraform.io)
- `version = "~> 5.0"` — оператор `~>` означает "5.x, но не 6.0". Защищает от ломающих обновлений

### `terraform/backend.tf`

```hcl
terraform {
  backend "gcs" {
    prefix = "terraform/state"
  }
}
```

**Построчно:**
- `backend "gcs"` — хранить state в Google Cloud Storage
- `prefix` — "папка" внутри bucket где лежит файл state
- Имя bucket НЕ указываем здесь — передадим при `terraform init` (потому что в backend блоке нельзя использовать переменные)

### Инициализируем:

```bash
cd terraform
terraform init -backend-config="bucket=MY_PROJECT-tfstate"
```

---

## Шаг 3. VPC — сеть для кластера

**Зачем:** GKE кластер живёт внутри VPC (Virtual Private Cloud). Нужна подсеть для нод + два вторичных диапазона IP для подов и сервисов Kubernetes.

Добавь файлы:
```
terraform/
├── variables.tf
└── main.tf
```

### `terraform/variables.tf`

```hcl
variable "project_id" {
  type = string
}

variable "region" {
  type    = string
  default = "asia-southeast1"
}

variable "zone" {
  type    = string
  default = "asia-southeast1-b"
}
```

**Построчно:**
- `project_id` — ID проекта GCP, обязательная
- `region` — регион для VPC и подсетей
- `zone` — конкретная зона для GKE (зональный кластер дешевле регионального — один control plane вместо трёх)

### `terraform/main.tf` (начало — провайдер + VPC)

```hcl
provider "google" {
  project = var.project_id
  region  = var.region
}

module "vpc" {
  source  = "terraform-google-modules/network/google"
  version = "~> 9.0"

  project_id   = var.project_id
  network_name = "todo-vpc"

  subnets = [
    {
      subnet_name           = "gke-subnet"
      subnet_ip             = "10.0.0.0/20"
      subnet_region         = var.region
      subnet_private_access = "true"
    }
  ]

  secondary_ranges = {
    gke-subnet = [
      {
        range_name    = "pods"
        ip_cidr_range = "10.1.0.0/16"
      },
      {
        range_name    = "services"
        ip_cidr_range = "10.2.0.0/20"
      }
    ]
  }
}
```

**Построчно:**
- `module "vpc"` — используем готовый модуль от Google вместо ручного создания `google_compute_network` + `google_compute_subnetwork`. Модуль делает то же самое, но с best practices
- `source = "terraform-google-modules/network/google"` — модуль из Terraform Registry
- `network_name = "todo-vpc"` — имя VPC сети
- `subnets` — список подсетей. У нас одна:
  - `subnet_ip = "10.0.0.0/20"` — 4094 IP адреса для нод кластера. /20 — это с запасом
  - `subnet_private_access = "true"` — ноды могут обращаться к Google API без внешнего IP
- `secondary_ranges` — вторичные диапазоны IP, привязанные к подсети. GKE требует их для VPC-native кластера:
  - `pods: 10.1.0.0/16` — 65534 IP для подов (GKE выделяет /24 на ноду, т.е. хватит на ~256 нод)
  - `services: 10.2.0.0/20` — 4094 IP для Kubernetes сервисов

### Проверяем:

```bash
terraform plan -var="project_id=MY_PROJECT"
```

Должен показать создание VPC + подсети. Пока НЕ применяй — добавим GKE.

---

## Шаг 4. GKE кластер на spot VM

Дописываем в `terraform/main.tf`:

```hcl
module "gke" {
  source  = "terraform-google-modules/kubernetes-engine/google"
  version = "~> 35.0"

  project_id = var.project_id
  name       = "todo-cluster"
  region     = var.region
  zones      = [var.zone]

  network           = module.vpc.network_name
  subnetwork        = module.vpc.subnets_names[0]
  ip_range_pods     = "pods"
  ip_range_services = "services"

  regional                   = false
  remove_default_node_pool   = true
  deletion_protection        = false

  node_pools = [
    {
      name         = "spot-pool"
      machine_type = "e2-medium"
      node_count   = 1
      spot         = true
      disk_size_gb = 30
      disk_type    = "pd-standard"
      auto_repair  = true
      auto_upgrade = true
    }
  ]

  node_pools_oauth_scopes = {
    all = ["https://www.googleapis.com/auth/cloud-platform"]
  }
}
```

**Построчно:**
- `module "gke"` — модуль от Google для создания GKE кластера
- `name = "todo-cluster"` — имя кластера
- `zones = [var.zone]` — в какой зоне создать ноды
- `network = module.vpc.network_name` — ссылка на VPC из шага 3. `module.vpc` — обращение к выходам модуля vpc
- `subnetwork = module.vpc.subnets_names[0]` — первая (единственная) подсеть
- `ip_range_pods = "pods"` — имя вторичного диапазона для подов (создали в VPC)
- `ip_range_services = "services"` — имя вторичного диапазона для сервисов
- `regional = false` — зональный кластер: один control plane, дешевле. Для прода ставят `true` (3 control plane в разных зонах)
- `remove_default_node_pool = true` — GKE создаёт дефолтный пул нод, мы его удаляем и создаём свой с нужными настройками
- `deletion_protection = false` — без этого `terraform destroy` не сможет удалить кластер. Для учебного проекта — ок
- `node_pools` — список пулов нод:
  - `spot = true` — **spot VM** — в 3-5 раз дешевле обычных, но GCP может забрать их в любой момент. Для todo app — нормально
  - `machine_type = "e2-medium"` — 2 vCPU, 4GB RAM. Хватит для нашего приложения
  - `disk_size_gb = 30` — диск ноды. 30GB минимум для GKE
  - `disk_type = "pd-standard"` — HDD, дешевле SSD. Для dev достаточно
  - `auto_repair = true` — GKE автоматически пересоздаёт сломанные ноды
  - `auto_upgrade = true` — GKE обновляет версию Kubernetes на нодах
- `node_pools_oauth_scopes` — какие API доступны нодам. `cloud-platform` — полный доступ (ограничивается через IAM)

---

## Шаг 5. Outputs

Создай `terraform/outputs.tf`:

```hcl
output "kubeconfig_command" {
  value = "gcloud container clusters get-credentials ${module.gke.name} --zone ${var.zone} --project ${var.project_id}"
}
```

**Построчно:**
- `output` — значение, которое Terraform напечатает после `apply`
- Формирует готовую команду для настройки kubectl — скопировал и вставил в терминал

---

## Шаг 6. tfvars и .gitignore

Создай `terraform/terraform.tfvars.example`:
```hcl
project_id = "my-gcp-project-id"
region     = "asia-southeast1"
zone       = "asia-southeast1-b"
```

Скопируй в `terraform/terraform.tfvars` и впиши свой project ID.

Добавь в `.gitignore`:
```
# Terraform
terraform/.terraform/
terraform/**/*.tfstate
terraform/**/*.tfstate.backup
terraform/**/*.tfvars
!terraform/**/*.tfvars.example
```

**Зачем:** `.terraform/` — скачанные провайдеры (тяжёлые). `*.tfstate` — содержит чувствительные данные. `*.tfvars` — может содержать project ID и другие значения.

---

## Шаг 7. Makefile таргеты

Добавь в `Makefile`:

```makefile
# ─── Terraform GCP ───

GCP_PROJECT ?= my-gcp-project-id

tf-bootstrap: ## Create GCS bucket for Terraform state
	cd terraform/bootstrap && terraform init && terraform apply -var="project_id=$(GCP_PROJECT)"

tf-init: ## Init Terraform with GCS backend
	cd terraform && terraform init -backend-config="bucket=$(GCP_PROJECT)-tfstate"

tf-plan: ## Plan Terraform changes
	cd terraform && terraform plan

tf-apply: ## Apply Terraform changes
	cd terraform && terraform apply

tf-destroy: ## Destroy all GCP infrastructure
	cd terraform && terraform destroy

tf-kubeconfig: ## Configure kubectl for GKE
	gcloud container clusters get-credentials todo-cluster --zone asia-southeast1-b --project $(GCP_PROJECT)
```

---

## Шаг 8. Применяем всё

```bash
# 1. Bootstrap bucket
make tf-bootstrap GCP_PROJECT=project-49ca1a08-b981-4f16-994

# 2. Init с remote backend
make tf-init GCP_PROJECT=project-49ca1a08-b981-4f16-994

# 3. Смотрим что создастся
make tf-plan

# 4. Создаём инфраструктуру
make tf-apply

# 5. Настраиваем kubectl
make tf-kubeconfig GCP_PROJECT=project-49ca1a08-b981-4f16-994

# 6. Проверяем
kubectl get nodes
```

Если `kubectl get nodes` показывает ноду со статусом `Ready` — всё работает.

---

## Итого: что мы создали

```
GCP Project
├── GCS Bucket (terraform state)
├── VPC "todo-vpc"
│   └── Subnet "gke-subnet" (10.0.0.0/20)
│       ├── Secondary: pods (10.1.0.0/16)
│       └── Secondary: services (10.2.0.0/20)
└── GKE "todo-cluster" (зональный, asia-southeast1-b)
    └── Node Pool "spot-pool"
        └── 1x e2-medium spot VM
```
