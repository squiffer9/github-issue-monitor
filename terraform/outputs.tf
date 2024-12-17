output "websocket_url" {
  description = "WebSocket URL for client connections"
  value       = trimsuffix(trimprefix("${aws_apigatewayv2_api.websocket.api_endpoint}/${aws_apigatewayv2_stage.websocket.name}", "https://"), "/")
}

output "webhook_url" {
  description = "Webhook URL for GitHub"
  value       = aws_lambda_function_url.webhook.function_url
}

output "lambda_role_arn" {
  description = "ARN of the Lambda IAM role"
  value       = aws_iam_role.lambda_role.arn
}
