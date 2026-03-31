📦 Build: Small To-Do App (API + Static Frontend)

━━━━━━━━━━━━━━━━━━━━
Requirements
━━━━━━━━━━━━━━━━━━━━

Choose one language:
• Go
• .NET
• Node.js

REST API must be documented with OpenAPI 3.0 (Swagger)  
Serve Swagger UI at:

/docs


━━━━━━━━━━━━━━━━━━━━
Data Layer
━━━━━━━━━━━━━━━━━━━━

Choose one database:
• Postgres
• MongoDB

Justify the choice:
• Why this DB? (e.g. relational constraints vs schemaless flexibility)

ORM choice (or no ORM) and why:

Go:
• GORM
• sqlc
• ent
• no-ORM

Node:
• Prisma
• TypeORM
• Knex
• Mongoose

.NET:
• EF Core
• Dapper
• no-ORM


━━━━━━━━━━━━━━━━━━━━
Migrations
━━━━━━━━━━━━━━━━━━━━

SQL databases:
• golang-migrate
• Flyway
• Prisma Migrate

MongoDB:
• collection indexes
• idempotent initialization scripts


━━━━━━━━━━━━━━━━━━━━
Minimal Feature Set
━━━━━━━━━━━━━━━━━━━━

CRUD Todos with fields:
• title
• completed
• dueDate
• optional tags

Filtering:
• completed
• dueBefore / dueAfter
• search in title

Health endpoints:

/healthz  (liveness)
/readyz   (readiness + DB ping)

Tests:
• service-level unit tests for domain
• at least one API test

Frontend:
• simple static page
• list / add / complete todos
• any framework or vanilla JS


━━━━━━━━━━━━━━━━━━━━
Deliverables
━━━━━━━━━━━━━━━━━━━━

• openapi.yaml (or generated during build)
• Source code
• Architecture & Tradeoffs note in README


━━━━━━━━━━━━━━━━━━━━
Local Dev Environment
━━━━━━━━━━━━━━━━━━━━

Goal: one-command local run using Docker Compose

README must include:
• prerequisites
• how to run docker compose up
• how to run migrations
• how to seed data
• how to open the app locally
• how to open Swagger UI
• common troubleshooting


━━━━━━━━━━━━━━━━━━━━
Local Kubernetes Cluster
━━━━━━━━━━━━━━━━━━━━

Goal: create local cluster with:

• 1 control-plane
• 1 worker node

Use:
• kind
• k3d


━━━━━━━━━━━━━━━━━━━━
Build Pipeline (Local-Friendly CI)
━━━━━━━━━━━━━━━━━━━━

Use:
• Azure DevOps
• GitHub Actions
• or any free CI

Goal: build production-ready Docker image

Example:

docker build -t localhost:5000/todo-api:${GIT_SHA} .
docker push localhost:5000/todo-api:${GIT_SHA}

Pipeline must include:
• lint
• tests
• OpenAPI generation/validation


━━━━━━━━━━━━━━━━━━━━
Helm Packaging
━━━━━━━━━━━━━━━━━━━━

Create chart:

helm create todo-app

Trim chart to include:

• API Deployment
• API Service (ClusterIP)
• Web Deployment (optional)

OR serve web statically from API

Also include:

• ConfigMap / Secret
• Ingress

Image configuration values:

• repository
• tag
• pullPolicy

Provide sane defaults in:

values.yaml


━━━━━━━━━━━━━━━━━━━━
Cluster Configuration
━━━━━━━━━━━━━━━━━━━━

Ensure kubeconfig targets the kind/k3d cluster.

Database options:

Option 1 — Postgres via Helm

bitnami/postgresql

(persistence disabled for simplicity)

Option 2 — reuse Docker Compose DB


━━━━━━━━━━━━━━━━━━━━
Deploy Application
━━━━━━━━━━━━━━━━━━━━

helm upgrade --install todo ./deploy/helm/todo-app \
  --set image.repository=localhost:5000/todo-api \
  --set image.tag=$GIT_SHA


━━━━━━━━━━━━━━━━━━━━
Release Workflow
━━━━━━━━━━━━━━━━━━━━

Provide script or Makefile target:

make deploy-kind

Steps:
1. create kind cluster (if needed)
2. load image into cluster

kind load docker-image ...

3. run Helm deploy


━━━━━━━━━━━━━━━━━━━━
Ingress / Service Exposure
━━━━━━━━━━━━━━━━━━━━

Goal: expose the app locally.

Recommended:

NGINX Ingress Controller (via Helm)

Optional:

MetalLB  
(simulate LoadBalancer on bare-metal)


━━━━━━━━━━━━━━━━━━━━
Explain Choice in README
━━━━━━━━━━━━━━━━━━━━

• Use NGINX Ingress only if NodePort/localhost access is enough.

• Use MetalLB if you want a real LoadBalancer IP on your LAN.