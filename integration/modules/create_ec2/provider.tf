terraform {
  required_providers {

    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.11"
    }
  }
  required_version = ">= 1.7.0"
}


provider "aws" {}

#Create provider_override.tf with the following for local use
# provider "aws" {
#   region = "us-west-1" 
#   profile = "blunderdome" 
#   assume_role {
#     role_arn = "IAM ROLE IN BLUNDERDOME" 
#   }
# }

