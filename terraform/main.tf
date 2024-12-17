terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "ap-northeast-1"

  # timeout
  http_proxy = ""
  retry_mode = "standard"
  skip_requesting_account_id = false
  max_retries = 5

  default_tags {
    tags = {
      Project     = "github-issue-monitor"
      Environment = "portfolio"
      Terraform   = "true"
    }
  }
}

# Archive Files for Lambda - moved to top
data "archive_file" "websocket_lambda" {
  type        = "zip"
  source_file = "${path.module}/websocket/bootstrap"
  output_path = "${path.module}/websocket.zip"

  depends_on = [data.local_file.websocket_bootstrap]
}

data "archive_file" "webhook_lambda" {
  type        = "zip"
  source_file = "${path.module}/webhook/bootstrap"
  output_path = "${path.module}/webhook.zip"

  depends_on = [data.local_file.webhook_bootstrap]
}

# Check for the existence of build artifacts
data "local_file" "websocket_bootstrap" {
  filename = "${path.module}/websocket/bootstrap"
}

data "local_file" "webhook_bootstrap" {
  filename = "${path.module}/webhook/bootstrap"
}

# API Gateway CloudWatch role
resource "aws_iam_role" "apigateway_cloudwatch" {
  name = "api-gateway-cloudwatch-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "apigateway.amazonaws.com"
      }
    }]
  })
}

# Permissions for API Gateway to write to CloudWatch
resource "aws_iam_role_policy" "apigateway_cloudwatch" {
  name = "api-gateway-cloudwatch-policy"
  role = aws_iam_role.apigateway_cloudwatch.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams",
        "logs:PutLogEvents",
        "logs:GetLogEvents",
        "logs:FilterLogEvents"
      ]
      Resource = "*"
    }]
  })
}

# Settings for API Gateway at the account level
resource "aws_api_gateway_account" "main" {
  cloudwatch_role_arn = aws_iam_role.apigateway_cloudwatch.arn
}
# Budget alert
resource "aws_budgets_budget" "cost_control" {
  name         = "portfolio-monthly-budget"
  budget_type  = "COST"
  limit_amount = "5"
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  notification {
    comparison_operator        = "GREATER_THAN"
    threshold                  = 80
    threshold_type            = "PERCENTAGE"
    notification_type         = "ACTUAL"
    subscriber_email_addresses = ["squiffer9@gmail.com"]
  }
}

# DynamoDB table for WebSocket connections
resource "aws_dynamodb_table" "connections" {
  name         = "github-issue-monitor-connections"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "connection_id"

  attribute {
    name = "connection_id"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }
}

# Lambda execution role
resource "aws_iam_role" "lambda_role" {
  name = "github-issue-monitor-lambda-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })
}

# Basic Lambda execution policy
resource "aws_iam_role_policy_attachment" "lambda_basic" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# API Gateway Management policy
resource "aws_iam_role_policy" "api_gateway_policy" {
  name = "api-gateway-management"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "execute-api:ManageConnections"
        ]
        Resource = "${aws_apigatewayv2_api.websocket.execution_arn}/*"
      }
    ]
  })
}

# DynamoDB access policy
resource "aws_iam_role_policy" "dynamodb_policy" {
  name = "dynamodb-access"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:DeleteItem",
          "dynamodb:Scan"
        ]
        Resource = aws_dynamodb_table.connections.arn
      }
    ]
  })
}

# WebSocket API
resource "aws_apigatewayv2_api" "websocket" {
  name                       = "github-issue-monitor-websocket"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
}

# WebSocket stage with cost-optimized settings
resource "aws_apigatewayv2_stage" "websocket" {
  api_id      = aws_apigatewayv2_api.websocket.id
  name        = "prod"
  auto_deploy = true

  default_route_settings {
    data_trace_enabled       = false
    detailed_metrics_enabled = false
    throttling_burst_limit   = 100
    throttling_rate_limit    = 100
  }

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api_gateway.arn
    format = jsonencode({
      requestId    = "$context.requestId"
      ip           = "$context.identity.sourceIp"
      requestTime  = "$context.requestTime"
      routeKey     = "$context.routeKey"
      status       = "$context.status"
      connectionId = "$context.connectionId"
      error        = "$context.error.message"
    })
  }
}

# Lambda functions with optimized settings
resource "aws_lambda_function" "websocket" {
  filename         = data.archive_file.websocket_lambda.output_path
  function_name    = "github-issue-monitor-websocket"
  role            = aws_iam_role.lambda_role.arn
  handler         = "bootstrap"
  runtime         = "provided.al2"
  architectures   = ["arm64"]
  timeout         = 10
  memory_size     = 128
  environment {
    variables = {
      API_GATEWAY_ENDPOINT = aws_apigatewayv2_api.websocket.api_endpoint
      DYNAMODB_TABLE      = aws_dynamodb_table.connections.name
    }
  }
}

# Lambda function for GitHub webhook
resource "aws_lambda_function" "webhook" {
  filename         = data.archive_file.webhook_lambda.output_path
  function_name    = "github-issue-monitor-webhook"
  role            = aws_iam_role.lambda_role.arn
  handler         = "bootstrap"
  runtime         = "provided.al2"
  architectures   = ["arm64"]
  timeout         = 10
  memory_size     = 128
  environment {
    variables = {
      WEBSOCKET_API_ENDPOINT = "${aws_apigatewayv2_api.websocket.api_endpoint}/${aws_apigatewayv2_stage.websocket.name}"
      GITHUB_WEBHOOK_SECRET  = var.github_webhook_secret
      DYNAMODB_TABLE        = aws_dynamodb_table.connections.name
    }
  }
}

# Function URL with auth
resource "aws_lambda_function_url" "webhook" {
  function_name      = aws_lambda_function.webhook.function_name
  authorization_type = "NONE"

  cors {
    allow_credentials = false
    allow_origins     = ["*"]
    allow_methods     = ["POST"]
    max_age          = 86400
  }
}

# WebSocket API Integration
resource "aws_apigatewayv2_integration" "websocket" {
  api_id           = aws_apigatewayv2_api.websocket.id
  integration_type = "AWS_PROXY"
  integration_uri  = aws_lambda_function.websocket.invoke_arn
}

# WebSocket routes
resource "aws_apigatewayv2_route" "connect" {
  api_id    = aws_apigatewayv2_api.websocket.id
  route_key = "$connect"
  target    = "integrations/${aws_apigatewayv2_integration.websocket.id}"
}

# WebSocket route_selection_expression
resource "aws_apigatewayv2_route" "disconnect" {
  api_id    = aws_apigatewayv2_api.websocket.id
  route_key = "$disconnect"
  target    = "integrations/${aws_apigatewayv2_integration.websocket.id}"
}

# WebSocket route_selection_expression
resource "aws_apigatewayv2_route" "default" {
  api_id    = aws_apigatewayv2_api.websocket.id
  route_key = "$default"
  target    = "integrations/${aws_apigatewayv2_integration.websocket.id}"
}

# Lambda permission for WebSocket API
resource "aws_lambda_permission" "websocket_api" {
  statement_id  = "AllowWebSocketAPIInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.websocket.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.websocket.execution_arn}/*/*"
}

# CloudWatch Log Groups with 3-day retention
resource "aws_cloudwatch_log_group" "websocket" {
  name              = "/aws/lambda/${aws_lambda_function.websocket.function_name}"
  retention_in_days = 3
}

# CloudWatch Log Groups with 3-day retention
resource "aws_cloudwatch_log_group" "webhook" {
  name              = "/aws/lambda/${aws_lambda_function.webhook.function_name}"
  retention_in_days = 3
}

# CloudWatch Log Groups with 3-day retention
resource "aws_cloudwatch_log_group" "api_gateway" {
  name              = "/aws/lambda/${aws_apigatewayv2_api.websocket.name}"
  retention_in_days = 3
}
