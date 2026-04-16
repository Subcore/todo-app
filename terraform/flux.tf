data "google_client_config" "default" {}

provider "kubernetes" {
  host                   = "https://${module.gke.endpoint}"
  cluster_ca_certificate = base64decode(module.gke.ca_certificate)
  token                  = data.google_client_config.default.access_token
}

provider "flux" {
  kubernetes = {
    host                   = "https://${module.gke.endpoint}"
    cluster_ca_certificate = base64decode(module.gke.ca_certificate)
    token                  = data.google_client_config.default.access_token
  }
  git = {
    url = "ssh://git@github.com/${var.github_owner}/${var.github_repository}.git"
    ssh = {
      username    = "git"
      private_key = tls_private_key.flux.private_key_pem
    }
  }
}

provider "github" {
  owner = var.github_owner
  token = var.github_token
}

resource "tls_private_key" "flux" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

resource "github_repository_deploy_key" "flux" {
  title      = "FluxCD Deploy Key"
  repository = var.github_repository
  key        = tls_private_key.flux.public_key_openssh
  read_only  = "false"
}

resource "flux_bootstrap_git" "this" {
  depends_on = [github_repository_deploy_key.flux, module.gke]

  path = "k8s/cluster"

  components_extra = [
    "image-reflector-controller",
    "image-automation-controller"
  ]
}

resource "kubernetes_namespace" "todo_app" {
  depends_on = [module.gke]

  metadata {
    name = "todo-app"
  }
}

resource "kubernetes_secret" "ghcr" {
  for_each   = toset(["todo-app", "flux-system"])
  depends_on = [kubernetes_namespace.todo_app, flux_bootstrap_git.this]

  metadata {
    name      = "ghcr-secret"
    namespace = each.value
  }

  type = "kubernetes.io/dockerconfigjson"

  data = {
    ".dockerconfigjson" = jsonencode({
      auths = {
        "ghcr.io" = {
          username = var.github_owner
          password = var.github_token
          auth     = base64encode("${var.github_owner}:${var.github_token}")
        }
      }
    })
  }
}
