package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Connection struct {
	ConnectionID string `dynamodbav:"connection_id"`
	ConnectedAt  string `dynamodbav:"connected_at"`
}

// saveConnection saves the connection ID to the DynamoDB table.
func saveConnection(ctx context.Context, tableName, connectionID string) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)

	conn := Connection{
		ConnectionID: connectionID,
		ConnectedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	item, err := attributevalue.MarshalMap(conn)
	if err != nil {
		return err
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &tableName,
		Item:      item,
	})
	return err
}

// deleteConnection deletes the connection ID from the DynamoDB table.
func deleteConnection(ctx context.Context, tableName, connectionID string) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)

	key, err := attributevalue.MarshalMap(map[string]string{
		"connection_id": connectionID,
	})
	if err != nil {
		return err
	}

	_, err = dynamoClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &tableName,
		Key:       key,
	})
	return err
}

// handleWebSocketEvent handles the WebSocket events.
func handleWebSocketEvent(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connectionID := request.RequestContext.ConnectionID
	tableName := os.Getenv("DYNAMODB_TABLE")

	switch request.RequestContext.RouteKey {
	case "$connect":
		log.Printf("Client connected: %s", connectionID)
		if err := saveConnection(ctx, tableName, connectionID); err != nil {
			log.Printf("Error saving connection: %v", err)
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

	case "$disconnect":
		log.Printf("Client disconnected: %s", connectionID)
		if err := deleteConnection(ctx, tableName, connectionID); err != nil {
			log.Printf("Error deleting connection: %v", err)
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

	case "$default":
		log.Printf("Message from %s: %s", connectionID, request.Body)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "OK",
	}, nil
}

// handleRequest handles the incoming requests.
func handleRequest(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	return handleWebSocketEvent(ctx, request)
}

func main() {
	lambda.Start(handleRequest)
}
