package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type IssueEvent struct {
	Action string `json:"action"`
	Issue  struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		HTMLURL string `json:"html_url"`
	} `json:"issue"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

type Connection struct {
	ConnectionID string `dynamodbav:"connection_id"`
	ConnectedAt  string `dynamodbav:"connected_at"`
}

// verifySignature verifies the GitHub webhook signature.
func verifySignature(signature string, body, secret []byte) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	actualMAC, err := hex.DecodeString(signature[7:])
	if err != nil {
		return false
	}

	// Compare the expected and actual MAC values
	return hmac.Equal(actualMAC, expectedMAC)
}

// getAllConnections retrieves all connection IDs from the DynamoDB table.
func getAllConnections(ctx context.Context, tableName string) ([]Connection, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)

	result, err := dynamoClient.Scan(ctx, &dynamodb.ScanInput{
		TableName: &tableName,
	})
	if err != nil {
		return nil, err
	}

	var connections []Connection
	err = attributevalue.UnmarshalListOfMaps(result.Items, &connections)
	if err != nil {
		return nil, err
	}

	return connections, nil
}

// handleWebhookEvent handles the incoming GitHub webhook event.
func handleWebhookEvent(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Verify the GitHub webhook signature
	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	signature := request.Headers["X-Hub-Signature-256"]
	if !verifySignature(signature, []byte(request.Body), []byte(secret)) {
		return events.APIGatewayProxyResponse{StatusCode: 401, Body: "Invalid signature"}, nil
	}

	var event IssueEvent
	if err := json.Unmarshal([]byte(request.Body), &event); err != nil {
		log.Printf("Error parsing webhook: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 400}, err
	}

	message := fmt.Sprintf("New issue #%d: %s\nRepository: %s\nURL: %s",
		event.Issue.Number,
		event.Issue.Title,
		event.Repository.FullName,
		event.Issue.HTMLURL,
	)

	wsApiEndpoint := os.Getenv("WEBSOCKET_API_ENDPOINT")
	if wsApiEndpoint == "" {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "WEBSOCKET_API_ENDPOINT not set"}, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	apiClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = &wsApiEndpoint
	})

	// Get all active connections from DynamoDB
	tableName := os.Getenv("DYNAMODB_TABLE")
	connections, err := getAllConnections(ctx, tableName)
	if err != nil {
		log.Printf("Error getting connections: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	// Broadcast to all connected clients
	for _, conn := range connections {
		_, err = apiClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
			ConnectionId: &conn.ConnectionID,
			Data:         []byte(message),
		})
		if err != nil {
			log.Printf("Error sending to connection %s: %v", conn.ConnectionID, err)
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "OK",
	}, nil
}

// handleRequest handles the incoming requests.
func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return handleWebhookEvent(ctx, request)
}

func main() {
	lambda.Start(handleRequest)
}
