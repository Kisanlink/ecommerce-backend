package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load("../.env")
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Get AWS credentials from environment
	accessKey := os.Getenv("DB_DYNAMO_ACCESS_KEY_ID")
	secretKey := os.Getenv("DB_DYNAMO_SECRET_ACCESS_KEY")
	region := os.Getenv("DB_DYNAMO_REGION")

	if accessKey == "" || secretKey == "" || region == "" {
		log.Fatal("Missing DynamoDB credentials in environment variables")
	}

	fmt.Printf("Creating DynamoDB table with credentials:\n")
	fmt.Printf("Region: %s\n", region)
	fmt.Printf("Access Key: %s...\n", accessKey[:10])

	// Create AWS config with credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Create DynamoDB client
	client := dynamodb.NewFromConfig(cfg)

	// Check if table already exists
	tableName := "product-reviews"
	_, err = client.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err == nil {
		fmt.Printf("Table '%s' already exists\n", tableName)
		return
	}

	fmt.Printf("Creating table '%s'...\n", tableName)

	// Create table
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("product_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("user_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("product_id"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("user_id"),
				KeyType:       types.KeyTypeRange,
			},
		},
		TableName:   aws.String(tableName),
		BillingMode: types.BillingModePayPerRequest,
	}

	_, err = client.CreateTable(context.TODO(), input)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	fmt.Printf("✅ Successfully initiated creation of DynamoDB table: %s\n", tableName)
	fmt.Println("⏱️ Table is being created. It may take 1-2 minutes to become active.")
	fmt.Println("🚀 You can now start your server and test the review system!")
}
