IMAGE_NAME := localhost:5000/todo-api
GIT_SHA    := $(shell git rev-parse --short HEAD)
IMAGE_TAG  := $(GIT_SHA)
CLUSTER_NAME := kind
HELM_RELEASE := todo
HELM_CHART   := ./deploy/helm/todo-app
INGRESS_NGINX_VERSION := v1.12.1

.PHONY: deploy-kind build-image create-cluster install-ingress load-image migrate helm-deploy delete-cluster

## Full local deployment pipeline
deploy-kind: create-cluster install-ingress build-image load-image helm-deploy migrate
	@echo ""
	@echo "=== Deployment complete ==="
	@echo "Add to /etc/hosts:  127.0.0.1  todo.local"
	@echo "App:     http://todo.local"
	@echo "Swagger: http://todo.local/docs"
	@echo ""

## Build Docker image
build-image:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

## Create kind cluster if it does not exist
create-cluster:
	@if kind get clusters 2>/dev/null | grep -q "^$(CLUSTER_NAME)$$"; then \
		echo "Cluster '$(CLUSTER_NAME)' already exists, skipping creation."; \
	else \
		echo "Creating kind cluster '$(CLUSTER_NAME)'..."; \
		kind create cluster --name $(CLUSTER_NAME) --config kind-config.yaml; \
	fi

## Install NGINX Ingress Controller for kind (pinned version)
install-ingress:
	@if kubectl get namespace ingress-nginx >/dev/null 2>&1; then \
		echo "ingress-nginx namespace exists, skipping installation."; \
	else \
		kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-$(INGRESS_NGINX_VERSION)/deploy/static/provider/kind/deploy.yaml; \
	fi
	kubectl wait --namespace ingress-nginx \
		--for=condition=ready pod \
		--selector=app.kubernetes.io/component=controller \
		--timeout=120s

## Load image into kind cluster
load-image:
	kind load docker-image $(IMAGE_NAME):$(IMAGE_TAG) --name $(CLUSTER_NAME)

## Deploy via Helm (includes API + PostgreSQL + Ingress)
helm-deploy:
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
		--set image.repository=$(IMAGE_NAME) \
		--set image.tag=$(IMAGE_TAG) \
		--wait --timeout 120s

## Run migrations via port-forward
migrate:
	@echo "Waiting for PostgreSQL pod to be ready..."
	kubectl wait --for=condition=ready pod \
		--selector=app=todo-postgresql \
		--timeout=120s
	@echo "Starting port-forward and running migrations..."
	kubectl port-forward svc/todo-postgresql 5433:5432 & \
	PF_PID=$$!; \
	sleep 4; \
	migrate -path=./migrations \
		-database="postgres://postgres:postgres@localhost:5433/todo_db?sslmode=disable" up; \
	kill $$PF_PID 2>/dev/null

## Delete kind cluster
delete-cluster:
	kind delete cluster --name $(CLUSTER_NAME)