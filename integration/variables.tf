variable "name_format" {
  type    = string
  default = "observe-agent-test-%s"
}


# tflint-ignore: terraform_naming_convention
variable "PUBLIC_KEY_PATH" {
  type        = string
  description = "Public key path ex - \"/Users/YOU/.ssh/id_rsa\""
  #default     = null
  nullable = true
}

# tflint-ignore: terraform_naming_convention
variable "PRIVATE_KEY_PATH" {
  description = "Private key path ex - \"/Users/YOU/.ssh/id_rsa.pub\""
  default     = null
  #nullable    = true
  type = string
}

# tflint-ignore: terraform_naming_convention
variable "CI" {
  type        = bool
  default     = false
  description = "This variable is set to true by github actions to tell us we are running in ci"
}


variable "AWS_MACHINE_FILTER" {
  description = "This is used as filter and run againt AWS_MACHINE_CONFIGS in main.tf - if set to null, don't filter anything"
  type        = any
  default     = null
}

# tflint-ignore: terraform_naming_convention
# variable "PUBLIC_KEY" {
#   description = "Public key var for running in ci"
#   nullable    = true
#   default     = null
#   type        = string
# }