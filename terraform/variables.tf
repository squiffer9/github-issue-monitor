variable "github_webhook_secret" {
  description = "Secret for GitHub webhook verification"
  type        = string
  sensitive   = true
}
