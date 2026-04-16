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
  description = "PostgreSQL password for the todo-app database"
  type        = string
  sensitive   = true
}