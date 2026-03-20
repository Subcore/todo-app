IMAGE_NAME := todo-api
IMAGE_TAG  := v1
CLUSTER_NAME := kind
HELM_RELEASE := todo-app
HELM_CHART   := ./helm/todo-app

.PHONY: deploy-kind build-image create-cluster install-ingress load-image deploy-db migrate helm-deploy delete-cluster

## Full local deployment pipeline
deploy-kind: create-cluster install-ingress build-image load-image deploy-db migrate helm-deploy

## Build Docker image
build-image:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

## Create kind cluster if it does not exist
create-cluster:
	@if kind get clusters | grep -q "^$(CLUSTER_NAME)$$"; then \
		echo "Cluster '$(CLUSTER_NAME)' already exists, skipping creation."; \
	else \
		echo "Creating kind cluster '$(CLUSTER_NAME)'..."; \
		kind create cluster --name $(CLUSTER_NAME) --config kind-config.yaml; \
	fi

## Install NGINX Ingress Controller for kind
install-ingress:
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
	kubectl wait --namespace ingress-nginx \
		--for=condition=ready pod \
		--selector=app.kubernetes.io/component=controller \
		--timeout=90s

## Load image into kind cluster
load-image:
	kind load docker-image $(IMAGE_NAME):$(IMAGE_TAG) --name $(CLUSTER_NAME)

## Deploy via Helm
helm-deploy:
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
		--set image.repository=$(IMAGE_NAME) \
		--set image.tag=$(IMAGE_TAG)

## Deploy PostgreSQL in cluster
deploy-db:
	@if helm list | grep -q todo-postgresql; then \
		echo "PostgreSQL already deployed, skipping."; \
	else \
		helm install todo-postgresql bitnami/postgresql \
			--set auth.username=postgres \
			--set auth.password=postgres \
			--set auth.database=todo_db \
			--set primary.persistence.enabled=false; \
		kubectl wait --for=condition=ready pod \
			--selector=app.kubernetes.io/instance=todo-postgresql \
			--timeout=120s; \
	fi

## Run migrations via port-forward
migrate:
	kubectl port-forward svc/todo-postgresql 5433:5432 &
	sleep 3
	migrate -path=./migrations \
		-database="postgres://postgres:postgres@localhost:5433/todo_db?sslmode=disable" up
	kill %1

## Delete kind cluster
delete-cluster:
	kind delete cluster --name $(CLUSTER_NAME)
