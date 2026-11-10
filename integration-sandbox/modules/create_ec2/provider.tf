terraform {
  required_providers {

    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.11"
    }
  }
  required_version = ">= 1.7.0"
}


# Credentials and region come from the environment (AWS_PROFILE / AWS_REGION locally,
# aws-actions/configure-aws-credentials in CI).
provider "aws" {}

