# Деплой todo-app в Google Cloud (GKE)

> Два варианта деплоя приложения: **ручной (Helm)** и **автоматический (FluxCD)**.
> Шаги 1-3 общие для обоих вариантов. Дальше выбираешь ОДИН из вариантов.

---

## Шаг 1. Предварительные требования

```bash
# Установить CLI-инструменты
brew install google-cloud-sdk kubectl helm

# (только для варианта B)
brew install fluxcd/tap/flux

# Авторизоваться в GCP
gcloud auth login
gcloud auth application-default login

# Задать проект
export GCP_PROJECT=<твой-project-id>
gcloud config set project $GCP_PROJECT
```

---

## Шаг 2. Создание кластера

### 2.1 Создать бакет для Terraform state

```bash
make tf-bootstrap GCP_PROJECT=$GCP_PROJECT
```

### 2.2 Поднять GKE кластер

```bash
make tf-init GCP_PROJECT=$GCP_PROJECT
make tf-plan GCP_PROJECT=$GCP_PROJECT
make tf-apply GCP_PROJECT=$GCP_PROJECT
```

### 2.3 Получить kubeconfig

```bash
make tf-kubeconfig GCP_PROJECT=$GCP_PROJECT
```

### 2.4 Проверить подключение

```bash
kubectl get nodes
```

---

## Шаг 3. Установить Nginx Ingress Controller

Этот шаг общий для обоих вариантов. Nginx Ingress ставится один раз при создании кластера.

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.service.type=LoadBalancer
```

Дождаться получения External IP:

```bash
kubectl -n ingress-nginx get svc ingress-nginx-controller -w
# NAME                       TYPE           EXTERNAL-IP
# ingress-nginx-controller   LoadBalancer   34.xxx.xxx.xxx
```

Запомни этот IP — он понадобится для `ingress.host`.

---

## Шаг 4. Подготовка Docker-образов

CI уже пушит образ API в GHCR при пуше в main:

```
ghcr.io/<owner>/todo-api:<commit-sha>
```

Создать pull-secret в кластере, чтобы GKE мог тянуть образы из GHCR:

```bash
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=<github-user> \
  --docker-password=<github-pat> \
  --docker-email=<email>
```

---

## Дальше выбери ОДИН вариант: A или B

---

## Вариант A: Ручной деплой через Helm

Подходит для: обучения, одного разработчика, быстрого старта.

### A.1 Создать values для GCP

Создать файл `deploy/helm/todo-app/values-gcp.yaml`:

```yaml
image:
  repository: ghcr.io/<owner>/todo-api
  tag: "<commit-sha>"
  pullPolicy: IfNotPresent

imagePullSecrets:
  - name: ghcr-secret

web:
  image:
    repository: ghcr.io/<owner>/todo-web
    tag: "<commit-sha>"
    pullPolicy: IfNotPresent

ingress:
  enabled: true
  className: nginx
  host: todo.example.com   # или 34.xxx.xxx.xxx.nip.io

db:
  host: todo-app-postgresql
  port: "5432"
  name: todo_db
  user: postgres
  password: "<надежный-пароль>"
  sslmode: disable

postgresql:
  auth:
    username: postgres
    password: "<надежный-пароль>"
    postgresPassword: "<надежный-пароль>"
    database: todo_db
  primary:
    persistence:
      enabled: true
      size: 10Gi
```

### A.2 Задеплоить

```bash
cd deploy/helm/todo-app
helm dependency update

helm install todo-app . \
  -f values.yaml \
  -f values-gcp.yaml \
  --wait --timeout 5m
```

### A.3 Обновить приложение (при каждом релизе)

```bash
helm upgrade todo-app deploy/helm/todo-app/ \
  -f deploy/helm/todo-app/values.yaml \
  -f deploy/helm/todo-app/values-gcp.yaml \
  --set image.tag=<new-commit-sha> \
  --wait --timeout 5m
```

### A.4 Проверить

```bash
kubectl get pods
kubectl get ingress
curl http://<EXTERNAL-IP>/api/v1/healthz
```

---

## Вариант B: Автоматический деплой через FluxCD

Подходит для: автоматизации, команды, production.

### B.1 Bootstrap Flux в кластер

```bash
export GITHUB_TOKEN=<personal-access-token>
export GITHUB_USER=<github-username>

flux bootstrap github \
  --owner=$GITHUB_USER \
  --repository=todo-app \
  --branch=main \
  --path=clusters/gcp \
  --personal
```

Flux создаст папку `clusters/gcp/flux-system/` в репозитории и закоммитит туда свои манифесты.

### B.2 Создать структуру файлов

```
clusters/
  gcp/
    flux-system/          # создано автоматически при bootstrap
    sources.yaml          # откуда брать Helm chart
    apps.yaml             # todo-app
```

### B.3 Файл sources.yaml

```yaml
# clusters/gcp/sources.yaml

# Источник: наш репозиторий (для Helm chart из deploy/helm/)
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: todo-app-repo
  namespace: flux-system
spec:
  interval: 1m
  url: https://github.com/<owner>/todo-app
  ref:
    branch: main
---
# Источник: Bitnami (для postgresql subchart)
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: bitnami
  namespace: flux-system
spec:
  interval: 1h
  url: https://charts.bitnami.com/bitnami
```

### B.4 Файл apps.yaml

```yaml
# clusters/gcp/apps.yaml

apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: todo-app
  namespace: flux-system
spec:
  interval: 5m
  targetNamespace: default
  chart:
    spec:
      chart: deploy/helm/todo-app
      sourceRef:
        kind: GitRepository
        name: todo-app-repo
      reconcileStrategy: Revision
  values:
    image:
      repository: ghcr.io/<owner>/todo-api
      tag: main
      pullPolicy: IfNotPresent
    web:
      image:
        repository: ghcr.io/<owner>/todo-web
        tag: main
        pullPolicy: IfNotPresent
    ingress:
      enabled: true
      className: nginx
      host: todo.example.com
    db:
      host: todo-app-postgresql
      port: "5432"
      name: todo_db
      user: postgres
      password: "<надежный-пароль>"   # в проде: SOPS или Sealed Secrets
      sslmode: disable
    postgresql:
      auth:
        username: postgres
        password: "<надежный-пароль>"
        postgresPassword: "<надежный-пароль>"
        database: todo_db
      primary:
        persistence:
          enabled: true
          size: 10Gi
```

### B.5 Запушить и проверить

```bash
git add clusters/
git commit -m "feat: add FluxCD configuration for GCP deployment"
git push origin main
```

Flux подхватит автоматически. Проверить статус:

```bash
flux get all
flux get helmreleases -A
flux logs --follow
```

### B.6 (Опционально) Автообновление image tag

Если хочешь чтобы Flux сам подхватывал новые образы из GHCR:

```bash
# Установить image-automation контроллеры
flux bootstrap github \
  --owner=$GITHUB_USER \
  --repository=todo-app \
  --branch=main \
  --path=clusters/gcp \
  --components-extra=image-reflector-controller,image-automation-controller \
  --personal
```

Добавить `clusters/gcp/image-automation.yaml`:

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImageRepository
metadata:
  name: todo-api
  namespace: flux-system
spec:
  image: ghcr.io/<owner>/todo-api
  interval: 1m
  secretRef:
    name: ghcr-auth
---
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImagePolicy
metadata:
  name: todo-api
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: todo-api
  filterTags:
    pattern: '^[a-f0-9]{7,40}$'
  policy:
    alphabetical:
      order: asc
---
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImageUpdateAutomation
metadata:
  name: todo-app
  namespace: flux-system
spec:
  interval: 1m
  sourceRef:
    kind: GitRepository
    name: todo-app-repo
  git:
    checkout:
      ref:
        branch: main
    commit:
      author:
        name: fluxcdbot
        email: flux@todo-app.local
      messageTemplate: "chore: update image to {{range .Changed.Changes}}{{.NewValue}}{{end}}"
    push:
      branch: main
  update:
    strategy: Setters
    path: ./clusters/gcp
```

Затем в `apps.yaml` добавить маркер для image automation:

```yaml
    image:
      repository: ghcr.io/<owner>/todo-api
      tag: main # {"$imagepolicy": "flux-system:todo-api:tag"}
```

---

## Сравнение вариантов

| | Helm (ручной) | FluxCD (автоматический) |
|---|---|---|
| Деплой | `helm upgrade` из терминала | `git push` |
| Обновление образа | `--set image.tag=xxx` руками | Flux подхватывает из GHCR |
| Откат | `helm rollback` | `git revert` |
| Видимость состояния | `helm list` | `flux get all` |
| Drift detection | нет | есть (Flux вернёт к Git-состоянию) |
| Сложность настройки | низкая | средняя (один раз) |
| Подходит когда | учишься / один разработчик | прод / команда / автоматизация |

---

## Полезные команды

```bash
# --- Terraform ---
make tf-plan GCP_PROJECT=$GCP_PROJECT    # посмотреть изменения
make tf-apply GCP_PROJECT=$GCP_PROJECT   # применить
make tf-destroy GCP_PROJECT=$GCP_PROJECT # удалить всё

# --- Helm (вариант A) ---
helm list                                # что установлено
helm history todo-app                    # история релизов
helm rollback todo-app 1                 # откат

# --- Flux (вариант B) ---
flux get all                             # статус всех ресурсов
flux get helmreleases -A                 # статус Helm-релизов
flux reconcile source git todo-app-repo  # принудительный sync
flux logs --follow                       # логи в реальном времени
flux suspend helmrelease todo-app        # пауза деплоя
flux resume helmrelease todo-app         # возобновить

# --- Общие ---
kubectl get pods                         # поды
kubectl get ingress                      # ingress + external IP
kubectl logs -l app.kubernetes.io/name=todo-app  # логи приложения
```

---

## Удаление всего

```bash
# 1. Удалить приложение
helm uninstall todo-app          # вариант A
# или flux suspend/удалить манифесты   # вариант B

# 2. Удалить ingress
helm uninstall ingress-nginx -n ingress-nginx

# 3. Удалить кластер
make tf-destroy GCP_PROJECT=$GCP_PROJECT
```
