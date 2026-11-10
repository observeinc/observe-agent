variable "OBSERVE_URL" {
  type        = string
  description = "Observe URL for agent to send data to. Eg: https://<tenant_id>.collect.observe-staging.com/"
}

variable "OBSERVE_TOKEN" {
  type        = string
  description = "Observe Token for Datastream for observe-agent to send data to. Eg: ds1....23AB"
} 