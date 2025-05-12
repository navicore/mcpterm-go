package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

const (
	// Claude models - AWS Bedrock model IDs
	// Some models use the us.anthropic.* prefix (US region specific models)
	// Others use the anthropic.* prefix (available in multiple regions)
	ModelClaude37Sonnet      = "us.anthropic.claude-3-7-sonnet-20250219-v1:0"  // US region model
	ModelClaude3Sonnet       = "anthropic.claude-3-sonnet-20240229-v1:0"       // Multi-region model
	ModelClaude3Haiku        = "anthropic.claude-3-haiku-20240307-v1:0"        // Multi-region model
	ModelClaude3Opus         = "anthropic.claude-3-opus-20240229-v1:0"         // Multi-region model

	// Default parameters
	DefaultMaxTokens   = 4096
	DefaultTemperature = 0.7
	DefaultTopP        = 0.9

	// Anthropic API version for Bedrock
	AnthropicVersion   = "bedrock-2023-05-31"
)

func init() {
	RegisterBackend(BackendAWSBedrock, NewBedrockBackend)
}

// BedrockBackend implements the Backend interface for AWS Bedrock
type BedrockBackend struct {
	client    *bedrockruntime.Client
	config    Config
	modelID   string
}

// ClaudeMessage represents a message in the Claude format
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequest represents the request format for Claude models in Bedrock
type AnthropicRequest struct {
	AnthropicVersion string          `json:"anthropic_version"`
	Messages         []ClaudeMessage `json:"messages"`
	MaxTokens        int             `json:"max_tokens"`
	Temperature      float64         `json:"temperature,omitempty"`
	TopP             float64         `json:"top_p,omitempty"`
	TopK             int             `json:"top_k,omitempty"`
	StopSequences    []string        `json:"stop_sequences,omitempty"`
	System           string          `json:"system,omitempty"`
}

// AnthropicResponse represents the response format from Claude models in Bedrock
type AnthropicResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Content      []struct {
		Type  string `json:"type"`
		Text  string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// NewBedrockBackend creates a new AWS Bedrock backend
func NewBedrockBackend(config Config) (Backend, error) {
	// Validate config
	if config.ModelID == "" {
		return nil, NewBackendError(
			ErrCodeInvalidConfiguration,
			"model ID is required",
			nil,
		)
	}

	// Create context with timeout for AWS operations
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use AWS SDK to create a Bedrock client with credentials and region
	cfg, err := LoadAWSConfig(ctx, config.Options)
	if err != nil {
		return nil, NewBackendError(
			ErrCodeAuthentication,
			"failed to load AWS configuration",
			err,
		)
	}

	// Create the Bedrock client
	client := bedrockruntime.NewFromConfig(cfg)

	// Set default parameters if not specified
	if config.MaxTokens <= 0 {
		config.MaxTokens = DefaultMaxTokens
	}
	if config.Temperature <= 0 {
		config.Temperature = DefaultTemperature
	}

	// Validate parameters
	if config.Temperature < 0 {
		config.Temperature = 0
	} else if config.Temperature > 1 {
		config.Temperature = 1
	}

	return &BedrockBackend{
		client:  client,
		config:  config,
		modelID: config.ModelID,
	}, nil
}

// LoadAWSConfig loads AWS configuration with optional overrides
func LoadAWSConfig(ctx context.Context, options map[string]any) (aws.Config, error) {
	loadOpts := []func(*config.LoadOptions) error{}

	// Apply custom region if specified
	if region, ok := options["region"].(string); ok && region != "" {
		loadOpts = append(loadOpts, config.WithRegion(region))
	}

	// Apply AWS profile if specified
	if profile, ok := options["profile"].(string); ok && profile != "" {
		loadOpts = append(loadOpts, config.WithSharedConfigProfile(profile))
	}

	// Add specific credentials if provided
	if accessKey, ok := options["access_key"].(string); ok && accessKey != "" {
		if secretKey, ok := options["secret_key"].(string); ok && secretKey != "" {
			// Create static credentials provider
			loadOpts = append(loadOpts, config.WithCredentialsProvider(
				aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
					return aws.Credentials{
						AccessKeyID:     accessKey,
						SecretAccessKey: secretKey,
					}, nil
				}),
			))
		}
	}

	// Setting shared config files
	loadOpts = append(loadOpts, config.WithSharedConfigFiles([]string{
		"~/.aws/config",
	}))

	loadOpts = append(loadOpts, config.WithSharedCredentialsFiles([]string{
		"~/.aws/credentials",
	}))

	// Set maximum retry attempts for AWS operations
	loadOpts = append(loadOpts, config.WithRetryMaxAttempts(5))

	// Load the configuration
	return config.LoadDefaultConfig(ctx, loadOpts...)
}

// Name returns the name of the backend
func (b *BedrockBackend) Name() string {
	return "AWS Bedrock"
}

// Type returns the type of the backend
func (b *BedrockBackend) Type() BackendType {
	return BackendAWSBedrock
}

// ModelID returns the model identifier
func (b *BedrockBackend) ModelID() string {
	return b.modelID
}

// SendMessage sends a message to Claude via AWS Bedrock
func (b *BedrockBackend) SendMessage(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// Convert to Anthropic format
	claudeMessages := make([]ClaudeMessage, 0, len(req.Messages))
	var systemPrompt string
	
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			// Claude expects system prompts in a separate field
			systemPrompt = msg.Content
		} else {
			claudeMessages = append(claudeMessages, ClaudeMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}
	
	// Set parameters
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = b.config.MaxTokens
	}

	// Validate and set temperature (must be between 0 and 1)
	temperature := req.Temperature
	if temperature <= 0 {
		temperature = b.config.Temperature
	}
	if temperature < 0 {
		temperature = 0
	} else if temperature > 1 {
		temperature = 1
	}

	// Validate and set topP (must be between 0 and 1)
	topP := req.TopP
	if topP <= 0 {
		topP = DefaultTopP
	}
	if topP < 0 {
		topP = 0
	} else if topP > 1 {
		topP = 1
	}
	
	// Extract any Claude-specific options
	var topK int
	var stopSequences []string
	
	if req.Options != nil {
		if val, ok := req.Options["top_k"].(int); ok {
			topK = val
		}
		if val, ok := req.Options["stop_sequences"].([]string); ok {
			stopSequences = val
		}
	}
	
	// Create the Claude request payload
	claudeReq := AnthropicRequest{
		AnthropicVersion: AnthropicVersion,
		Messages:         claudeMessages,
		MaxTokens:        maxTokens,
		Temperature:      temperature,
		TopP:             topP,
		System:           systemPrompt,
	}
	
	if topK > 0 {
		claudeReq.TopK = topK
	}
	
	if len(stopSequences) > 0 {
		claudeReq.StopSequences = stopSequences
	}
	
	// Marshal the request to JSON
	reqJSON, err := json.Marshal(claudeReq)
	if err != nil {
		return ChatResponse{Error: err}, NewBackendError(
			ErrCodeInvalidRequest,
			"failed to marshal Claude request",
			err,
		)
	}
	
	// Create a context with timeout for the API call
	apiCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Call the Bedrock API
	bedrockReq := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(b.modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        reqJSON,
	}

	bedrockResp, err := b.client.InvokeModel(apiCtx, bedrockReq)
	if err != nil {
		// Check for context timeout
		if apiCtx.Err() == context.DeadlineExceeded {
			return ChatResponse{Error: err}, NewBackendError(
				ErrCodeServiceUnavailable,
				"request to AWS Bedrock timed out after 60 seconds",
				err,
			)
		}

		return ChatResponse{Error: err}, mapBedrockError(err)
	}
	
	// Parse the response
	var claudeResp AnthropicResponse
	if err := json.Unmarshal(bedrockResp.Body, &claudeResp); err != nil {
		return ChatResponse{Error: err}, NewBackendError(
			ErrCodeUnknown,
			"failed to unmarshal Claude response",
			err,
		)
	}
	
	// Extract content from the response
	var content strings.Builder
	for _, c := range claudeResp.Content {
		if c.Type == "text" {
			content.WriteString(c.Text)
		}
	}
	
	// Build the response
	usage := make(map[string]int)
	usage["prompt_tokens"] = claudeResp.Usage.InputTokens
	usage["completion_tokens"] = claudeResp.Usage.OutputTokens
	usage["total_tokens"] = claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens
	
	return ChatResponse{
		Content:     content.String(),
		FinishReason: claudeResp.StopReason,
		Usage:       usage,
	}, nil
}

// Close closes any resources held by the backend
func (b *BedrockBackend) Close() error {
	// No resources to close for Bedrock
	return nil
}

// mapBedrockError maps AWS Bedrock errors to our error types
func mapBedrockError(err error) error {
	// Check for common error patterns based on the error message
	errMsg := err.Error()

	// Throttling and rate limiting errors
	if strings.Contains(errMsg, "rate limit") ||
	   strings.Contains(errMsg, "throttled") ||
	   strings.Contains(errMsg, "ThrottlingException") ||
	   strings.Contains(errMsg, "TooManyRequestsException") {
		return NewBackendError(
			ErrCodeRateLimited,
			"API rate limit exceeded. Please try again in a few moments.",
			err,
		)
	}

	// Authentication and permission errors
	if strings.Contains(errMsg, "AccessDeniedException") ||
	   strings.Contains(errMsg, "AuthorizationException") ||
	   strings.Contains(errMsg, "UnrecognizedClientException") ||
	   strings.Contains(errMsg, "InvalidSignatureException") ||
	   strings.Contains(errMsg, "not authorized") {
		return NewBackendError(
			ErrCodeAuthentication,
			"Authentication failed. Please check your AWS credentials and permissions.",
			err,
		)
	}

	// Content filtering errors
	if strings.Contains(errMsg, "content filter") ||
	   strings.Contains(errMsg, "safety") ||
	   strings.Contains(errMsg, "ContentFilterException") ||
	   strings.Contains(errMsg, "violated content policy") {
		return NewBackendError(
			ErrCodeContentFiltered,
			"Content was filtered due to safety or content policy concerns.",
			err,
		)
	}

	// Context length and token limit errors
	if strings.Contains(errMsg, "context length") ||
	   strings.Contains(errMsg, "token limit") ||
	   strings.Contains(errMsg, "too many tokens") ||
	   strings.Contains(errMsg, "ModelTokenLimitExceededException") {
		return NewBackendError(
			ErrCodeContextLengthExceeded,
			"Input exceeded maximum context length for the model.",
			err,
		)
	}

	// Validation and invalid parameter errors
	if strings.Contains(errMsg, "validation") ||
	   strings.Contains(errMsg, "invalid") ||
	   strings.Contains(errMsg, "ValidationException") ||
	   strings.Contains(errMsg, "InvalidRequestException") {
		return NewBackendError(
			ErrCodeInvalidRequest,
			"Invalid request parameters. Please check your model configuration.",
			err,
		)
	}

	// Network and connectivity errors
	if strings.Contains(errMsg, "connection") ||
	   strings.Contains(errMsg, "timeout") ||
	   strings.Contains(errMsg, "network") ||
	   strings.Contains(errMsg, "dial") ||
	   strings.Contains(errMsg, "EOF") {
		return NewBackendError(
			ErrCodeNetwork,
			"Network error occurred. Please check your internet connection.",
			err,
		)
	}

	// Service unavailable errors
	if strings.Contains(errMsg, "ServiceUnavailableException") ||
	   strings.Contains(errMsg, "service unavailable") ||
	   strings.Contains(errMsg, "InternalServerException") ||
	   strings.Contains(errMsg, "500") {
		return NewBackendError(
			ErrCodeServiceUnavailable,
			"AWS Bedrock service is currently unavailable. Please try again later.",
			err,
		)
	}

	// Model specific errors
	if strings.Contains(errMsg, "ModelNotReadyException") {
		return NewBackendError(
			ErrCodeServiceUnavailable,
			"The requested model is not ready or available in this region.",
			err,
		)
	}

	if strings.Contains(errMsg, "ModelNotFoundException") ||
	   strings.Contains(errMsg, "model not found") {
		return NewBackendError(
			ErrCodeInvalidConfiguration,
			fmt.Sprintf("Model not found: %s. Please check the model ID and region.",
				strings.Split(errMsg, ":")[0]),
			err,
		)
	}

	// Default to unknown error
	return NewBackendError(
		ErrCodeUnknown,
		fmt.Sprintf("Unknown AWS Bedrock error: %v", err),
		err,
	)
}