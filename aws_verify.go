//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// This is a test file to verify AWS credentials are working properly.
// Run with: go run aws_verify.go

func main() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Set region - change to your AWS region
	region := "us-west-2"

	// Load config
	fmt.Println("Loading AWS configuration...")
	loadOpts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		// Disable EC2 IMDS to prevent hanging in non-EC2 environments
		config.WithEC2IMDSEndpoint(""),
	}

	cfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		fmt.Printf("❌ Failed to load AWS config: %v\n", err)
		os.Exit(1)
	}

	// Get credentials
	fmt.Println("Retrieving AWS credentials...")
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to retrieve AWS credentials: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ AWS credentials successfully retrieved for: %s\n", creds.AccessKeyID)

	// Create Bedrock client
	fmt.Println("Creating Bedrock client...")
	client := bedrockruntime.NewFromConfig(cfg)

	// Print model list - this is commented out because it needs ListFoundationModels permission
	/*
	fmt.Println("Testing Bedrock API access...")
	req := &bedrockruntime.ListFoundationModelsInput{}
	
	resp, err := client.ListFoundationModels(ctx, req)
	if err != nil {
		fmt.Printf("❌ Failed to list Bedrock models: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully connected to AWS Bedrock! Found %d models.\n", len(resp.ModelSummaries))
	
	// List a few Claude models
	fmt.Println("\nAvailable Claude models:")
	for _, model := range resp.ModelSummaries {
		if model.ModelId != nil && 
		   (aws.ToString(model.ModelName) == "Claude" || 
		    aws.ToString(model.ProviderName) == "Anthropic") {
			fmt.Printf("- %s\n", aws.ToString(model.ModelId))
		}
	}
	*/
	
	// Make a simple invoke model request to verify permissions
	fmt.Println("Testing InvokeModel API access...")
	modelID := "anthropic.claude-3-haiku-20240307-v1:0" // Using the simplest model for testing
	
	// Simple request
	reqBody := []byte(`{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens": 50,
		"messages": [
			{
				"role": "user",
				"content": "Hello, are you working? Reply with just 'Yes' or 'No'."
			}
		]
	}`)
	
	invokeReq := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        reqBody,
	}
	
	_, err = client.InvokeModel(ctx, invokeReq)
	if err != nil {
		fmt.Printf("❌ Failed to invoke Bedrock model: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("\n✅ AWS credential test passed! Your application should work now.")
}