variable "github_owner" {
  description = "Пользователь или организация GitHub"
  type        = string
  default     = "Subcore"
}

variable "github_repository" {
  description = "Название репозитория"
  type        = string
  default     = "todo-app-v2"
}

variable "github_token" {
  description = "Pat токен GitHub с правами к репозиторию"
  type        = string
  sensitive   = true
}
