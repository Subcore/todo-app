terraform {
  backend "gcs" {
    bucket = "todo-app-v2-vlcv-fluxcd-02-tfstate"
    prefix = "terraform/state"
  }
}