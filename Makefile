IMAGE_NAME := localhost:5000/todo-api
GIT_SHA    := $(shell git rev-parse --short HEAD)
IMAGE_TAG  := $(GIT_SHA)
CLUSTER_NAME := kind
HELM_RELEASE := todo
HELM_CHART   := ./deploy/helm/todo-app
INGRESS_NGINX_VERSION := v1.12.1
REGISTRY_NAME := kind-registry
REGISTRY_PORT := 5000

.PHONY: deploy-kind build-image create-registry create-cluster connect-registry configure-registry install-ingress push-image helm-deps migrate seed helm-deploy ensure-hosts delete-cluster delete-registry clean

## Full local deployment pipeline
deploy-kind: create-registry create-cluster connect-registry configure-registry install-ingress build-image push-image helm-deps helm-deploy migrate ensure-hosts
	@echo ""
	@echo "=== Deployment complete ==="
	@echo "App:     http://todo.local"
	@echo "Swagger: http://todo.local/docs"
	@echo ""

## Build Docker image
build-image:
	@echo "[6/11] Building Docker image $(IMAGE_NAME):$(IMAGE_TAG)..."
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

## Create local Docker registry if not running
create-registry:
	@echo "[1/11] Setting up local Docker registry..."
	@if docker inspect $(REGISTRY_NAME) >/dev/null 2>&1; then \
		echo "  Registry '$(REGISTRY_NAME)' already running, skipping."; \
	else \
		echo "  Starting local Docker registry on port $(REGISTRY_PORT)..."; \
		docker run -d --restart=always -p "127.0.0.1:$(REGISTRY_PORT):5000" --network bridge --name $(REGISTRY_NAME) registry:2; \
	fi

## Create kind cluster if it does not exist
create-cluster:
	@echo "[2/11] Setting up kind cluster '$(CLUSTER_NAME)'..."
	@if kind get clusters 2>/dev/null | grep -q "^$(CLUSTER_NAME)$$"; then \
		echo "  Cluster '$(CLUSTER_NAME)' already exists, skipping creation."; \
	else \
		echo "  Creating kind cluster '$(CLUSTER_NAME)'..."; \
		kind create cluster --name $(CLUSTER_NAME) --config kind-config.yaml; \
	fi

## Connect registry to kind network
connect-registry:
	@echo "[3/11] Connecting registry to kind network..."
	@if docker network inspect kind | grep -q '"$(REGISTRY_NAME)"'; then \
		echo "  Registry already connected to kind network."; \
	else \
		docker network connect kind $(REGISTRY_NAME) || true; \
	fi

## Configure registry access on kind nodes (containerd 2.x hosts.toml)
configure-registry:
	@echo "[4/11] Configuring registry on kind nodes..."
	@for node in $$(kind get nodes --name $(CLUSTER_NAME)); do \
		echo "  Configuring node: $$node"; \
		docker exec $$node mkdir -p /etc/containerd/certs.d/localhost:$(REGISTRY_PORT); \
		printf '[host."http://$(REGISTRY_NAME):5000"]\n  capabilities = ["pull", "resolve", "push"]\n' \
			| docker exec -i $$node cp /dev/stdin /etc/containerd/certs.d/localhost:$(REGISTRY_PORT)/hosts.toml; \
	done

## Install NGINX Ingress Controller for kind (pinned version)
install-ingress:
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

## Push image to local registry
push-image:
	@echo "[7/11] Pushing image to local registry..."
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

## Build Helm chart dependencies
helm-deps:
	@echo "[8/11] Building Helm chart dependencies..."
	helm dependency build $(HELM_CHART)

## Deploy via Helm (includes API + PostgreSQL + Ingress)
helm-deploy:
	@echo "[9/11] Deploying via Helm (timeout: 120s)..."
	@echo "  Tip: watch progress in another terminal: kubectl get pods -w"
	@helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
		--set image.repository=$(IMAGE_NAME) \
		--set image.tag=$(IMAGE_TAG) \
		--wait --timeout 120s || { \
		echo ""; \
		echo "  [ERROR] Helm deploy timed out. Pod status:"; \
		kubectl get pods 2>/dev/null | sed 's/^/  /'; \
		echo ""; \
		echo "  Recent events:"; \
		kubectl get events --sort-by='.lastTimestamp' 2>/dev/null | tail -15 | sed 's/^/  /'; \
		exit 1; \
	}

## Run migrations via port-forward
migrate:
	@echo "[10/11] Running database migrations..."
	@echo "  Waiting for PostgreSQL pod to be ready..."
	kubectl wait --for=condition=ready pod \
		--selector=app.kubernetes.io/instance=todo,app.kubernetes.io/name=postgresql \
		--timeout=120s
	@echo "  Starting port-forward and running migrations..."
	kubectl port-forward svc/todo-postgresql 5433:5432 & \
	PF_PID=$$!; \
	sleep 4; \
	migrate -path=./migrations \
		-database="postgres://postgres:postgres@localhost:5433/todo_db?sslmode=disable" up; \
	kill $$PF_PID 2>/dev/null

## Load seed data via port-forward
seed:
	@echo "Waiting for PostgreSQL pod to be ready..."
	kubectl wait --for=condition=ready pod \
		--selector=app.kubernetes.io/instance=todo,app.kubernetes.io/name=postgresql \
		--timeout=120s
	@echo "Loading seed data..."
	kubectl port-forward svc/todo-postgresql 5433:5432 & \
	PF_PID=$$!; \
	sleep 4; \
	PGPASSWORD=postgres psql -h localhost -p 5433 -U postgres -d todo_db -f seeds/seed.sql; \
	kill $$PF_PID 2>/dev/null

## Ensure todo.local is in /etc/hosts
ensure-hosts:
	@echo "[11/11] Checking /etc/hosts for todo.local..."
	@if grep -q 'todo\.local' /etc/hosts; then \
		echo "  todo.local already in /etc/hosts, skipping."; \
	else \
		echo "  Adding todo.local to /etc/hosts (requires sudo)..."; \
		echo '127.0.0.1  todo.local' | sudo tee -a /etc/hosts > /dev/null; \
	fi

## Delete kind cluster
delete-cluster:
	kind delete cluster --name $(CLUSTER_NAME)

## Delete local registry
delete-registry:
	docker rm -f $(REGISTRY_NAME) 2>/dev/null || true

## Full cleanup
clean: delete-cluster delete-registry
