terraform {
  required_providers {

    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.11"
    }
  }
  required_version = ">= 1.7.0"
}


provider "aws" {
  region = "us-west-1"
  # profile = "blunderdome" #Local Use Only
  # assume_role {
  #   role_arn = "IAM ROLE IN BLUNDERDOME" #Local Use Only 
  # }
}



