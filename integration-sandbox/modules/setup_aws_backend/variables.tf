variable "region" {
  description = "Region the integration test instances run in"
  type        = string
  default     = "us-west-2"
}

variable "role_name" {
  description = "Name of the IAM role assumed by GitHub Actions"
  type        = string
  default     = "gh-observe_agent-repo"
}

variable "github_subject" {
  description = "GitHub OIDC subject allowed to assume the role"
  type        = string
  default     = "repo:observeinc/observe-agent:*"
}

variable "test_resource_tag" {
  description = "Value of the `purpose` tag on resources created by the integration tests"
  type        = string
  default     = "observe-agent integration tests"
}
