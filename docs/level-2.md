Общий план
(ГОТОВО) Этап 1: Ansible — деплой на голую VM 
Цель: научиться доставлять приложение без Kubernetes

Поднять VM в Multipass
Ansible playbook: установка Nginx
Ansible playbook: деплой фронтенда (копируем web/ в nginx)
Ansible playbook: деплой бэкенда (собираем Go бинарник локально, копируем на VM, systemd unit)
Проверить что всё работает
(Опционально) повторить на Hetzner VM

Этап 2: Terraform — инфраструктура в GCP
Цель: подготовить K8s кластер для FluxCD

Создать проект в GCP
Terraform: GCS bucket для state
Terraform: настроить backend на этот bucket
Terraform: VPC + subnets для Kubernetes
Terraform: GKE кластер на spot VMs
Проверить kubectl доступ к кластеру


Этап 3: FluxCD — GitOps
Цель: автоматический деплой через git

Установить FluxCD в GKE кластер
Подключить FluxCD к твоему git-репо
Настроить FluxCD на Helm-чарты из deploy/helm/
Push в git → FluxCD автоматически деплоит
Проверить что todo-app работает в GKE
Зависимости

Этап 1 (Ansible) — независимый, делаем первым
         ↓
Этап 2 (Terraform) — создаёт кластер
         ↓
Этап 3 (FluxCD) — деплоит в этот кластер
Этап 1 и 2 можно делать параллельно — они не зависят друг от друга. Но лучше по порядку, так как Ansible проще и даст понимание основ.

Начинаем с Этапа 1 — Ansible плейбуки?

