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


