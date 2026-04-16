provider "google" {
  project = var.project_id
  region  = var.region
}

module "vpc" {
  source  = "terraform-google-modules/network/google"
  version = "~> 9.0"

  project_id   = var.project_id
  network_name = "todo-vpc"

  subnets = [
    {
      subnet_name           = "gke-subnet"
      subnet_ip             = "10.0.0.0/20"
      subnet_region         = var.region
      subnet_private_access = "true"
    }
  ]

  secondary_ranges = {
    gke-subnet = [
      {
        range_name    = "pods"
        ip_cidr_range = "10.1.0.0/16"
      },
      {
        range_name    = "services"
        ip_cidr_range = "10.2.0.0/20"
      }
    ]
  }
}

module "gke" {
  source  = "terraform-google-modules/kubernetes-engine/google"
  version = "~> 36.0"

  project_id = var.project_id
  name       = "todo-cluster"
  region     = var.region
  zones      = [var.zone]

  network           = module.vpc.network_name
  subnetwork        = module.vpc.subnets_names[0]
  ip_range_pods     = "pods"
  ip_range_services = "services"

  regional                   = false
  remove_default_node_pool   = true
  deletion_protection        = false

  node_pools = [
    {
      name         = "spot-pool"
      machine_type = "e2-medium"
      node_count   = 1
      min_count    = 1
      max_count    = 3
      spot         = true
      disk_size_gb = 50
      disk_type    = "pd-standard"
      auto_repair  = true
      auto_upgrade = true
    }
  ]

  node_pools_oauth_scopes = {
    all = ["https://www.googleapis.com/auth/cloud-platform"]
  }
}

resource "google_compute_address" "ingress" {
  name    = "todo-ingress-ip"
  region  = var.region
  project = var.project_id
}