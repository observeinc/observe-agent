terraform {
  required_providers {

    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.11"
    }
    time = {
      source  = "hashicorp/time"
      version = "0.11.2"
    }

  }
  required_version = ">= 1.2"


}

provider "time" {
  # Configuration options
}
provider "aws" {
  region  = "us-west-1" # Specify the AWS region
  profile = "blunderdome"

  assume_role {
    role_arn = "arn:aws:iam::767397788203:role/OrganizationAccountAccessRole"
  }
}