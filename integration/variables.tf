# # your local key path (assumes it exists) - this will allow you to access ec2 instances
# # tflint-ignore: terraform_naming_convention
# variable "PUBLIC_KEY_PATH" {
#   description = "Public key path"
#   nullable    = true
#   type        = string
# }

# # tflint-ignore: terraform_naming_convention
# variable "PRIVATE_KEY_PATH" {
#   description = "Private key path"
#   nullable    = true
#   type        = string
# }

# variable "name_format" {
#   description = "Common prefix for resource names"
#   type        = string
# }


# variable "AWS_MACHINE_FILTER" {
#   description = "This is used as filter and run againt AWS_MACHINE_CONFIGS in main.tf - if set to null, don't filter anything"
#   type        = string
#   default     = null
# }
