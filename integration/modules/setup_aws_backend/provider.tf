provider "aws" {
  region  = "us-west-1"
  profile = "blunderdome"
  assume_role {
    role_arn = "arn:aws:iam::767397788203:role/OrganizationAccountAccessRole"
  }
}

