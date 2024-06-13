variable "name_format" {
  description = "Common prefix for resource names"
  type        = string
}

variable "AWS_MACHINE_FILTER" {
  description = "This is used as filter and run againt AWS_MACHINE_CONFIGS in main.tf - if set to null, don't filter anything"
  type        = any
  default     = null
}

variable "PUBLIC_KEY_PATH" {
  description = "Public key path"
  nullable    = true
  type        = string
}

# tflint-ignore: terraform_naming_convention
variable "PRIVATE_KEY_PATH" {
  description = "Private key path"
  nullable    = true
  type        = string
}

variable "CI" {
  type        = bool
  default     = false
  description = "This variable is set to true by github actions to tell us we are running in ci"
}


variable "aws_region" {
  type    = string
  default = "us-west-1"
  description = "AWS region"
}

variable "aws_profile" {
  type = string
  description = "AWS profile"
  sensitive = true
}

variable "aws_role_arn" {
    type = string
    description = "AWS role arn"
  
}