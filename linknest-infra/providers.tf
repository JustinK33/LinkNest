# providers.tf for the provider setup

terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "4.6.0"
    }
  }
}

# terraform init: downloads the provider into .terraform/ and records the version in .terraform.lock.hcl
# basically like install the tool before job starts
# re run init when we add a new provider or module
# also downloads modules if we have any

provider "docker" {}