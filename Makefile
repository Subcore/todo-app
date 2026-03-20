IMAGE_NAME := todo-api
IMAGE_TAG  := v1
CLUSTER_NAME := kind
HELM_RELEASE := todo-app
HELM_CHART   := ./helm/todo-app

.PHONY: deploy-kind build-image create-cluster load-image helm-deploy

## Full deploy to local kind cluster
deploy-kind: create-cluster build-image load-image helm-deploy

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

## Load image into kind cluster
load-image:
	kind load docker-image $(IMAGE_NAME):$(IMAGE_TAG) --name $(CLUSTER_NAME)

## Deploy via Helm
helm-deploy:
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
		--set image.repository=$(IMAGE_NAME) \
		--set image.tag=$(IMAGE_TAG)
