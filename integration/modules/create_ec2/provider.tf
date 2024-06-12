terraform {
  required_providers {

    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.11"
    }
  }
  required_version = ">= 1.2"
}


provider "aws" {
  
  region  = var.aws_region
  profile = var.aws_profile

  assume_role {
    role_arn = var.aws_role_arn
  }
}
