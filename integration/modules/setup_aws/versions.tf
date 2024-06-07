# https://www.terraform.io/language/expressions/version-constraints
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.11"
    }

    random = {
      source  = "hashicorp/random"
      version = ">= 3.4.3"
    }
  }
  required_version = ">= 1.2"
}

