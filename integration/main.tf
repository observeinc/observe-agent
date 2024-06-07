# locals {
#   name_format = var.CI == true ? "gha-lht-${var.WORKFLOW_MATRIX_VALUE}-%s" : var.name_format
# }


module "aws_machines" {
  source           = "./AWS"
  PUBLIC_KEY_PATH  = var.PUBLIC_KEY_PATH
  PRIVATE_KEY_PATH = var.PRIVATE_KEY_PATH
  name_format        = var.name_format
  AWS_MACHINE_FILTER = ["AMAZON_LINUX_2023"]
  CI                 = var.CI
}