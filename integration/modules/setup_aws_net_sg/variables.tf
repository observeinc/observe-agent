variable "BASE_TAGS" {
  description = "base resource tags"
  type        = map(string)
  default = {
    owner        = "Observe"
    createdBy    = "terraform"
    team         = "Product Specialists "
    purpose      = "observe-agent integration tests"
    git_repo_url = "https://github.com/observeinc/observe-agent"
  }
}

variable "name_format" {
  description = "Common prefix for resource names"
  type        = string
  default     = "tf-observe-agent-test-%s"
}

