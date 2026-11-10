variable "name_format" {
  description = "Common prefix for resource names"
  type        = string
}

variable "AWS_MACHINE" {
  description = "This is used to choose a machine and run againt AWS_MACHINE_CONFIGS in main.tf"
  type        = string
}

variable "PUBLIC_KEY_PATH" {
  description = "Public key path. Used to attach a public key at this path to the ec2 instance"
  nullable    = true
  type        = string
}
variable "PRIVATE_KEY_PATH" {
  description = "Private key path. Used to SSH into the EC2 instance"
  nullable    = true
  type        = string
}

variable "subnet_id" {
  description = "Private subnet to launch the instance in. Must be reachable through an EC2 Instance Connect Endpoint"
  type        = string
}

variable "security_group_id" {
  description = "Security group for the instance. Must allow port 22 from the EC2 Instance Connect Endpoint"
  type        = string
}

variable "run_id" {
  description = "Tag value identifying the CI run, used to clean up instances left behind by cancelled runs"
  type        = string
  default     = "local"
}


