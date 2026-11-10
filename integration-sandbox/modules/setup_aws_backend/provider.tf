terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  required_version = ">= 1.7.0"
}

# Credentials come from the environment (e.g. AWS_PROFILE).
provider "aws" {
  region = var.region
}
