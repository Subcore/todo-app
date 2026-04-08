IMAGE_NAME     := localhost:5000/todo-api
WEB_IMAGE_NAME := localhost:5000/todo-web
GIT_SHA    := $(shell git rev-parse --short HEAD)
IMAGE_TAG  := $(GIT_SHA)
CLUSTER_NAME := kind
HELM_RELEASE := todo
HELM_CHART   := ./deploy/helm/todo-app
INGRESS_NGINX_VERSION := v1.12.1
REGISTRY_NAME := kind-registry
REGISTRY_PORT := 5000
REGISTRY_IMAGE := registry:3
DB_USER       ?= postgres
DB_PASS       ?= postgres
DB_NAME       ?= todo_db
DB_PORT       ?= 5433

.PHONY: help test lint run deploy-kind build-image build-web-image create-registry create-cluster connect-registry configure-registry install-ingress push-image push-web-image helm-deps migrate migrate-down seed helm-deploy ensure-hosts smoke-test delete-cluster delete-registry clean install-docker install-tools

.DEFAULT_GOAL := help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: ## Run tests
	go test ./...

lint: ## Run linter
	golangci-lint run
	helm lint $(HELM_CHART)

run: ## Start app locally via docker-compose
	docker compose up

deploy-kind: create-registry create-cluster connect-registry configure-registry install-ingress build-image build-web-image push-image push-web-image helm-deps helm-deploy migrate ensure-hosts smoke-test ## Full local deployment pipeline (run 'make install-tools' first, then re-login)
	@echo ""
	@echo "=== Deployment complete ==="
	@echo "Web:     http://todo.local/web"
	@echo "API:     http://todo.local/api"
	@echo "Swagger: http://todo.local/api/docs"
	@echo ""

smoke-test: ## Verify the app is responding after deployment
	@echo "[smoke] Checking http://todo.local/api/v1/healthz..."
	@TRIES=0; \
	until curl -sf http://todo.local/api/v1/healthz >/dev/null 2>&1; do \
		TRIES=$$((TRIES+1)); \
		if [ $$TRIES -ge 15 ]; then \
			echo "  [ERROR] App not responding after 15s"; \
			curl -sv http://todo.local/api/v1/healthz 2>&1 | sed 's/^/  /'; \
			exit 1; \
		fi; \
		sleep 1; \
	done
	@echo "  App is healthy."

build-image: ## Build Docker image
	@echo "[6/11] Building Docker image $(IMAGE_NAME):$(IMAGE_TAG)..."
	docker build \
		--label "org.opencontainers.image.revision=$(GIT_SHA)" \
		--label "org.opencontainers.image.version=$(GIT_SHA)" \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		-t $(IMAGE_NAME):latest .

# NOTE: The assignment suggests using `kind load docker-image` to load images into the cluster.
# We use a local Docker registry (localhost:5000) instead — this approach is closer to a real
# CI/CD pipeline (build once, push to registry, pull anywhere) and avoids re-loading images
# on every deploy. The registry is connected to the kind network so nodes can pull from it directly.
create-registry: ## Create local Docker registry if not running
	@echo "[1/11] Setting up local Docker registry..."
	@if docker inspect $(REGISTRY_NAME) >/dev/null 2>&1; then \
		echo "  Registry '$(REGISTRY_NAME)' already running, skipping."; \
	else \
		echo "  Starting local Docker registry on port $(REGISTRY_PORT)..."; \
		docker run -d --restart=always -p "127.0.0.1:$(REGISTRY_PORT):5000" --network bridge --name $(REGISTRY_NAME) $(REGISTRY_IMAGE); \
	fi

create-cluster: ## Create kind cluster if it does not exist
	@echo "[2/11] Setting up kind cluster '$(CLUSTER_NAME)'..."
	@if kind get clusters 2>/dev/null | grep -q "^$(CLUSTER_NAME)$$"; then \
		echo "  Cluster '$(CLUSTER_NAME)' already exists, skipping creation."; \
	else \
		echo "  Creating kind cluster '$(CLUSTER_NAME)'..."; \
		kind create cluster --name $(CLUSTER_NAME) --config kind-config.yaml; \
	fi
	@echo "  Ensuring worker node roles..."
	@kubectl label node $(CLUSTER_NAME)-worker node-role.kubernetes.io/worker= --overwrite 2>/dev/null || true

connect-registry: ## Connect registry to kind network
	@echo "[3/11] Connecting registry to kind network..."
	@if docker network inspect kind | grep -q '"$(REGISTRY_NAME)"'; then \
		echo "  Registry already connected to kind network."; \
	else \
		docker network connect kind $(REGISTRY_NAME) || true; \
	fi

configure-registry: ## Configure registry access on kind nodes
	@echo "[4/11] Configuring registry on kind nodes..."
	@for node in $$(kind get nodes --name $(CLUSTER_NAME)); do \
		echo "  Configuring node: $$node"; \
		docker exec $$node mkdir -p /etc/containerd/certs.d/localhost:$(REGISTRY_PORT); \
		printf '[host."http://$(REGISTRY_NAME):5000"]\n  capabilities = ["pull", "resolve", "push"]\n' \
			| docker exec -i $$node cp /dev/stdin /etc/containerd/certs.d/localhost:$(REGISTRY_PORT)/hosts.toml; \
	done

install-ingress: ## Install NGINX Ingress Controller for kind
	@echo "[5/11] Installing ingress-nginx controller $(INGRESS_NGINX_VERSION)..."
	@if kubectl get namespace ingress-nginx >/dev/null 2>&1; then \
		echo "  ingress-nginx namespace exists, skipping installation."; \
	else \
		kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-$(INGRESS_NGINX_VERSION)/deploy/static/provider/kind/deploy.yaml; \
	fi
	@echo "  Waiting for controller pod to become Ready (timeout: 120s)..."
	@echo "  Tip: watch progress in another terminal: kubectl get pods -n ingress-nginx -w"
	@kubectl wait --namespace ingress-nginx \
		--for=condition=ready pod \
		--selector=app.kubernetes.io/component=controller \
		--timeout=120s || { \
		echo ""; \
		echo "  [ERROR] Timed out waiting for ingress-nginx. Recent events:"; \
		kubectl get events -n ingress-nginx --sort-by='.lastTimestamp' 2>/dev/null | tail -15 | sed 's/^/  /'; \
		exit 1; \
	}

build-web-image: ## Build frontend Docker image
	@echo "[6b/11] Building frontend image $(WEB_IMAGE_NAME):$(IMAGE_TAG)..."
	docker build \
		-f Dockerfile.web \
		--build-arg GIT_SHA=$(GIT_SHA) \
		-t $(WEB_IMAGE_NAME):$(IMAGE_TAG) \
		-t $(WEB_IMAGE_NAME):latest .

push-image: ## Push API image to local registry
	@echo "[7/11] Pushing API image to local registry..."
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(IMAGE_NAME):latest

push-web-image: ## Push frontend image to local registry
	@echo "[7b/11] Pushing frontend image to local registry..."
	docker push $(WEB_IMAGE_NAME):$(IMAGE_TAG)
	docker push $(WEB_IMAGE_NAME):latest

helm-deps: ## Build Helm chart dependencies
	@echo "[8/11] Building Helm chart dependencies..."
	@if ! helm repo list 2>/dev/null | grep -q bitnami; then \
		echo "  Adding Bitnami Helm repository..."; \
		helm repo add bitnami https://charts.bitnami.com/bitnami; \
		helm repo update; \
	fi
	helm dependency build $(HELM_CHART)

helm-deploy: ## Deploy via Helm (API + PostgreSQL + Ingress)
	@echo "[9/11] Deploying via Helm (timeout: 120s)..."
	@echo "  Tip: watch progress in another terminal: kubectl get pods -w"
	@helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
		--set image.repository=$(IMAGE_NAME) \
		--set image.tag=$(IMAGE_TAG) \
		--set web.image.repository=$(WEB_IMAGE_NAME) \
		--set web.image.tag=$(IMAGE_TAG) \
		--wait --timeout 120s || { \
		echo ""; \
		echo "  [ERROR] Helm deploy timed out. Pod status:"; \
		kubectl get pods 2>/dev/null | sed 's/^/  /'; \
		echo ""; \
		echo "  Recent events:"; \
		kubectl get events --sort-by='.lastTimestamp' 2>/dev/null | tail -15 | sed 's/^/  /'; \
		exit 1; \
	}

migrate: ## Run migrations via port-forward
	@echo "[10/11] Running database migrations..."
	@echo "  Waiting for PostgreSQL pod to be ready..."
	kubectl wait --for=condition=ready pod \
		--selector=app.kubernetes.io/instance=todo,app.kubernetes.io/name=postgresql \
		--timeout=120s
	@echo "  Starting port-forward and running migrations..."
	kubectl port-forward svc/todo-postgresql $(DB_PORT):5432 & \
	PF_PID=$$!; \
	sleep 2; \
	TRIES=0; \
	until nc -z localhost $(DB_PORT) 2>/dev/null; do \
		TRIES=$$((TRIES+1)); \
		if [ $$TRIES -ge 30 ]; then echo "  [ERROR] PostgreSQL not ready after 30s"; kill $$PF_PID 2>/dev/null; exit 1; fi; \
		sleep 1; \
	done; \
	migrate -path=./migrations \
		-database="postgres://$(DB_USER):$(DB_PASS)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=disable" up; \
	kill $$PF_PID 2>/dev/null

migrate-down: ## Rollback last migration
	@echo "[migrate-down] Rolling back last migration..."
	@echo "  Waiting for PostgreSQL pod to be ready..."
	kubectl wait --for=condition=ready pod \
		--selector=app.kubernetes.io/instance=todo,app.kubernetes.io/name=postgresql \
		--timeout=120s
	@echo "  Starting port-forward and rolling back..."
	kubectl port-forward svc/todo-postgresql $(DB_PORT):5432 & \
	PF_PID=$$!; \
	sleep 2; \
	TRIES=0; \
	until nc -z localhost $(DB_PORT) 2>/dev/null; do \
		TRIES=$$((TRIES+1)); \
		if [ $$TRIES -ge 30 ]; then echo "  [ERROR] PostgreSQL not ready after 30s"; kill $$PF_PID 2>/dev/null; exit 1; fi; \
		sleep 1; \
	done; \
	migrate -path=./migrations \
		-database="postgres://$(DB_USER):$(DB_PASS)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=disable" down 1; \
	kill $$PF_PID 2>/dev/null

seed: ## Load seed data via kubectl exec into the PostgreSQL pod
	@echo "[seed] Waiting for PostgreSQL pod to be ready..."
	kubectl wait --for=condition=ready pod \
		--selector=app.kubernetes.io/instance=todo,app.kubernetes.io/name=postgresql \
		--timeout=120s
	@echo "[seed] Loading seed data..."
	@PG_POD=$$(kubectl get pod \
		--selector=app.kubernetes.io/instance=todo,app.kubernetes.io/name=postgresql \
		-o jsonpath='{.items[0].metadata.name}'); \
	kubectl cp seeds/seed.sql $$PG_POD:/tmp/seed.sql; \
	kubectl exec $$PG_POD -- env PGPASSWORD=$(DB_PASS) psql -U $(DB_USER) -d $(DB_NAME) -f /tmp/seed.sql

ensure-hosts: ## Ensure todo.local is in /etc/hosts
	@echo "[11/11] Checking /etc/hosts for todo.local..."
	@if grep -q 'todo\.local' /etc/hosts; then \
		echo "  todo.local already in /etc/hosts, skipping."; \
	else \
		echo "  Adding todo.local to /etc/hosts..."; \
		echo '127.0.0.1  todo.local' | sudo tee -a /etc/hosts > /dev/null; \
	fi

delete-cluster: ## Delete kind cluster
	kind delete cluster --name $(CLUSTER_NAME) 2>/dev/null || true

delete-registry: ## Delete local registry
	docker rm -f $(REGISTRY_NAME) 2>/dev/null || true

clean: delete-cluster delete-registry ## Full cleanup (cluster + registry + images)
	@docker rmi $$(docker images $(IMAGE_NAME) -q) 2>/dev/null || true

install-docker: ## Install Docker Engine (Linux only) and add current user to docker group
	@OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	if command -v docker >/dev/null 2>&1; then \
		echo "  Already installed: $$(docker --version)"; \
	elif [ "$$OS" = "darwin" ]; then \
		echo "  [ERROR] Docker Desktop not found. Install it from https://www.docker.com/products/docker-desktop/"; \
		exit 1; \
	else \
		echo "  Installing Docker Engine..."; \
		curl -fsSL https://get.docker.com | sh; \
		sudo usermod -aG docker $$USER; \
		sudo systemctl enable --now docker; \
		echo "  Installed: $$(docker --version)"; \
		echo ""; \
		echo "  NOTE: Run 'newgrp docker' or log out and back in,"; \
		echo "  then run 'make install-tools && make deploy-kind'."; \
	fi

install-tools: ## Install all required tools (kind, kubectl, helm, golang-migrate)
	@echo "=== Installing required tools ==="
	@OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	ARCH=$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/;s/arm64/arm64/'); \
	echo "Detected OS=$$OS ARCH=$$ARCH"; \
	echo ""; \
	echo "[1/4] kind..."; \
	if command -v kind >/dev/null 2>&1; then \
		echo "  Already installed: $$(kind version)"; \
	else \
		echo "  Installing kind..."; \
		curl -Lo ./kind "https://kind.sigs.k8s.io/dl/latest/kind-$$OS-$$ARCH" && chmod +x ./kind && sudo mv ./kind /usr/local/bin/kind; \
		echo "  Installed: $$(kind version)"; \
	fi; \
	echo ""; \
	echo "[2/4] kubectl..."; \
	if command -v kubectl >/dev/null 2>&1; then \
		echo "  Already installed: $$(kubectl version --client --short 2>/dev/null || kubectl version --client)"; \
	else \
		echo "  Installing kubectl..."; \
		KUBECTL_VERSION=$$(curl -Ls https://dl.k8s.io/release/stable.txt); \
		curl -LO "https://dl.k8s.io/release/$$KUBECTL_VERSION/bin/$$OS/$$ARCH/kubectl" && chmod +x kubectl && sudo mv kubectl /usr/local/bin/; \
		echo "  Installed: $$(kubectl version --client --short 2>/dev/null || kubectl version --client)"; \
	fi; \
	echo ""; \
	echo "[3/4] helm..."; \
	if command -v helm >/dev/null 2>&1; then \
		echo "  Already installed: $$(helm version --short)"; \
	else \
		echo "  Installing helm..."; \
		curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash; \
		echo "  Installed: $$(helm version --short)"; \
	fi; \
	echo ""; \
	echo "[4/4] golang-migrate..."; \
	if command -v migrate >/dev/null 2>&1; then \
		echo "  Already installed: $$(migrate -version 2>&1 || true)"; \
	else \
		echo "  Installing golang-migrate..."; \
		if [ "$$OS" = "darwin" ]; then \
			brew install golang-migrate; \
		else \
			curl -L "https://github.com/golang-migrate/migrate/releases/latest/download/migrate.$$OS-$$ARCH.tar.gz" | tar xvz -C /tmp && sudo mv /tmp/migrate /usr/local/bin/; \
		fi; \
		echo "  Installed: $$(migrate -version 2>&1 || true)"; \
	fi; \
	echo ""; \
	echo "=== All tools ready ==="

# ─── Ansible VPS deployment ───

build-linux: ## Build Go binary for Linux amd64
	@echo "Building Linux binary..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o todo-api ./cmd/api/main.go
	@echo "Binary ready: ./todo-api"

ansible-deploy: ## Run Ansible playbook to deploy to VPS
	cd ansible && ansible-playbook playbook.yml

deploy-vps: build-linux ansible-deploy ## Full VPS deploy: build binary + run Ansible
	@echo ""
	@echo "=== VPS Deployment complete ==="
	@echo "App: http://178.104.160.66"
	@echo "API: http://178.104.160.66/api"
	@echo ""
