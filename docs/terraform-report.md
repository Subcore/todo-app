# Отчёт: Terraform инфраструктура в GCP

## Что было сделано

Развёрнута инфраструктура в Google Cloud Platform для запуска Kubernetes кластера.
Это Этап 2 из трёхэтапного плана деплоя todo-app:

1. ~~Этап 1: Ansible — деплой на VPS (Hetzner)~~ — выполнен
2. **Этап 2: Terraform — инфраструктура в GCP** — выполнен
3. Этап 3: FluxCD — GitOps автоматический деплой — следующий шаг

## Созданные ресурсы

```
GCP Project: project-49ca1a08-b981-4f16-994
├── GCS Bucket: project-49ca1a08-b981-4f16-994-tfstate
│   └── Хранит Terraform state (состояние инфраструктуры)
├── VPC: todo-vpc
│   └── Subnet: gke-subnet (10.0.0.0/20)
│       ├── Secondary range: pods (10.1.0.0/16)
│       └── Secondary range: services (10.2.0.0/20)
├── GKE Cluster: todo-cluster (зональный, asia-southeast1-b)
│   └── Node Pool: spot-pool
│       └── 1x e2-medium spot VM (2 vCPU, 4GB RAM)
├── Service Account: tf-gke-todo-cluster-trq7
│   ├── roles/monitoring.metricWriter
│   ├── roles/container.defaultNodeServiceAccount
│   └── roles/stackdriver.resourceMetadata.writer
└── IAM bindings (3 шт.)
```

## Структура Terraform файлов

```
terraform/
├── bootstrap/              # Одноразовая настройка — создание bucket для state
│   ├── main.tf             # Ресурс google_storage_bucket
│   └── variables.tf        # project_id, region
├── main.tf                 # Основная конфигурация: VPC + GKE
├── variables.tf            # Входные переменные
├── outputs.tf              # Выходные значения (команда kubeconfig)
├── backend.tf              # Настройка remote backend (GCS)
├── versions.tf             # Версии Terraform и провайдеров
├── terraform.tfvars        # Значения переменных (не в git)
└── terraform.tfvars.example # Шаблон для terraform.tfvars
```

## Как это работает

### Bootstrap (одноразово)

Terraform хранит состояние в файле `terraform.tfstate` — это список всех созданных ресурсов.
По умолчанию файл лежит локально, что ненадёжно. Поэтому мы создаём GCS bucket для хранения state в облаке.

Проблема: bucket нужно создать до того, как настроить remote backend. Решение — отдельная директория `bootstrap/` с локальным state, которая создаёт только bucket.

### Remote Backend

```hcl
backend "gcs" {
  prefix = "terraform/state"
}
```

После создания bucket основной Terraform использует его как backend. Имя bucket передаётся при инициализации через флаг `-backend-config`, потому что в блоке `backend` нельзя использовать переменные.

### VPC (Virtual Private Cloud)

Используется модуль `terraform-google-modules/network/google`.

GKE кластер живёт внутри VPC. Создана одна подсеть с тремя диапазонами IP:

| Диапазон | CIDR | Назначение | Кол-во IP |
|----------|------|------------|-----------|
| Основной | 10.0.0.0/20 | Ноды кластера | 4094 |
| Вторичный: pods | 10.1.0.0/16 | Поды Kubernetes | 65534 |
| Вторичный: services | 10.2.0.0/20 | Сервисы Kubernetes | 4094 |

Вторичные диапазоны — требование GKE для VPC-native кластера. Каждый под и сервис получает свой IP из этих диапазонов.

### GKE Cluster

Используется модуль `terraform-google-modules/kubernetes-engine/google`.

| Параметр | Значение | Почему |
|----------|----------|--------|
| Тип | Зональный | Дешевле регионального (1 control plane вместо 3) |
| Зона | asia-southeast1-b | Сингапур — ближайший регион к Таиланду |
| VM тип | e2-medium | 2 vCPU, 4GB RAM — достаточно для todo-app |
| Spot VM | Да | В 3-5 раз дешевле обычных VM |
| Кол-во нод | 1 | Минимум для работы |
| Диск | 30GB pd-standard | HDD, дешевле SSD |
| Auto-repair | Да | GKE пересоздаёт сломанные ноды |
| Auto-upgrade | Да | GKE обновляет версию Kubernetes |
| Deletion protection | Нет | Чтобы `terraform destroy` мог удалить кластер |

Модуль GKE автоматически создал:
- Service Account для нод с минимальными правами (мониторинг, метрики)
- Дефолтный node pool удалён, вместо него создан `spot-pool`
- Workload Identity включён для безопасного доступа подов к GCP API

## Используемые версии

| Компонент | Версия |
|-----------|--------|
| Terraform CLI | 1.14.8 |
| Provider google | ~> 6.0 (установлен 6.50.0) |
| Provider google-beta | 6.50.0 |
| Provider kubernetes | 2.38.0 |
| Модуль VPC | 9.3.0 |
| Модуль GKE | 35.0.1 |
| Kubernetes (на кластере) | 1.35.1-gke.1396002 |

## Команды управления

```bash
# Инициализация (после клонирования репо)
make tf-init GCP_PROJECT=project-49ca1a08-b981-4f16-994

# Посмотреть план изменений
make tf-plan

# Применить изменения
make tf-apply

# Настроить kubectl
make tf-kubeconfig GCP_PROJECT=project-49ca1a08-b981-4f16-994

# Проверить ноды
kubectl get nodes

# Удалить всю инфраструктуру
make tf-destroy
```

## Стоимость

| Ресурс | Примерная стоимость |
|--------|-------------------|
| GKE control plane (зональный) | Бесплатно |
| e2-medium spot VM (1 шт.) | ~$7/мес |
| GCS bucket (state) | ~$0.01/мес |
| Сеть (VPC, subnet) | Бесплатно |
| **Итого** | **~$7/мес** |

Используются бесплатные $300 trial credit. При текущем потреблении хватит на ~40 месяцев.

**Важно:** когда кластер не нужен — удалить через `make tf-destroy`, чтобы не тратить кредиты.

## Проблемы и решения при настройке

| Проблема | Причина | Решение |
|----------|---------|---------|
| `storage.buckets.create` 403 | Авторизован под аккаунтом без прав | Создан новый проект под аккаунт с trial credits |
| `Unsupported Terraform version 1.5.7` | Старая версия из brew | Установлен через `hashicorp/tap`, обновлён до 1.14.8 |
| `no available releases match constraints ~> 5.0, >= 6.11.0` | Модуль GKE v35 требует провайдер >= 6.11 | Обновлён constraint до `~> 6.0` |
| `Module not installed` | Добавлен модуль GKE, но не скачан | `terraform init` скачивает новые модули |
| `Kubernetes Engine API disabled` | API не включён в проекте | `gcloud services enable container.googleapis.com` |
