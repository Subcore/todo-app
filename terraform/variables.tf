variable "project_id" {
  type = string
}

variable "region" {
  type    = string
  default = "asia-southeast1"
}

variable "zone" {
  type    = string
  default = "asia-southeast1-b"
}

variable "db_password" {
  description = "PostgreSQL password for todo-app database"
  type        = string
  sensitive   = true
}

variable "github_owner" {
  description = "GitHub repository owner (user or org)"
  type        = string
}

variable "github_repository" {
  description = "GitHub repository name for FluxCD"
  type        = string
}

variable "github_token" {
  description = "GitHub personal access token (repo scope) for FluxCD"
  type        = string
  sensitive   = true
}