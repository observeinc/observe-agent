#The Canonical User ID data source allows access to the canonical user ID for the effective account in which Terraform is working.
# tflint-ignore: terraform_unused_declarations
# data "aws_canonical_user_id" "current_user" {
# }

locals {
  test_key_value = {
    for key, value in random_string.output : key => "${key}_${value.id}"
  }
}

# # rando value for filtering output and validating results
resource "random_string" "output" {
  for_each = local.compute_instances
  length   = 6
  special  = false
  # keepers = {
  #   # Generate a new id each time script files change from linux_host_script
  #   output = var.script_hash[each.key]
  # }
}


data "aws_security_group" "ec2_public" {
  name = "tf-observe-agent-test-ec2_sg"
}

data "aws_subnet" "subnet_public" {
  filter {
    name   = "tag:Name"
    values = ["tf-observe-agent-test-subnet"]
  }
}

data "aws_key_pair" "ec2" {
  key_name           = "tf-observe-agent-test-publicKey"
  include_public_key = true  
}
