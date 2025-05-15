package context

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// File logger setup - All logging goes to a single file in /tmp
var persistenceLogger *log.Logger

func init() {
	// Create or append to log file in /tmp
	logFile, err := os.OpenFile("/tmp/mcpterm_context.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// If we can't open the log file, create a dummy logger that discards output
		persistenceLogger = log.New(os.NewFile(0, os.DevNull), "", 0)
		return
	}

	// Create logger with timestamp and prefix
	persistenceLogger = log.New(logFile, "PERSISTENCE: ", log.LstdFlags|log.Lshortfile)
}

// PersistenceConfig contains configuration for context persistence
type PersistenceConfig struct {
	// Base directory for storing context data
	BaseDir string

	// Whether to compress stored data
	Compress bool

	// Maximum number of context files to keep
	MaxContextFiles int

	// File permissions for created files
	FileMode os.FileMode

	// Retention policy settings
	MaxRetentionDays int

	// Whether to include system messages in persisted data
	IncludeSystemMessages bool
}

// DefaultPersistenceConfig returns the default persistence configuration
func DefaultPersistenceConfig() PersistenceConfig {
	homeDir, err := os.UserHomeDir()
	basePath := filepath.Join(homeDir, ".mcpterm", "context")
	if err != nil {
		basePath = "./.mcpterm/context"
	}

	return PersistenceConfig{
		BaseDir:               basePath,
		Compress:              true,
		MaxContextFiles:       10,
		FileMode:              0600,
		MaxRetentionDays:      30,
		IncludeSystemMessages: false,
	}
}

// ContextMetadata contains metadata about a persisted context
type ContextMetadata struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	MessageCount   int       `json:"message_count"`
	SummaryCount   int       `json:"summary_count"`
	SessionTokens  int       `json:"session_tokens"`
	PrimaryModelID string    `json:"primary_model_id"`
	Topics         []string  `json:"topics"`
	Tags           []string  `json:"tags"`
}

// ContextData contains the full context data for serialization
type ContextData struct {
	Metadata  ContextMetadata   `json:"metadata"`
	Messages  []EnhancedMessage `json:"messages"`
	Summaries []Summary         `json:"summaries"`
	Version   string            `json:"version"`
}

// ContextPersistenceManager handles saving and loading context data
type ContextPersistenceManager struct {
	config PersistenceConfig
}

// NewContextPersistenceManager creates a new persistence manager
func NewContextPersistenceManager(config PersistenceConfig) (*ContextPersistenceManager, error) {
	// Ensure the base directory exists
	if config.BaseDir != "" {
		if err := os.MkdirAll(config.BaseDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create context directory: %w", err)
		}
	}

	return &ContextPersistenceManager{
		config: config,
	}, nil
}

// SaveContext saves the context to disk
func (p *ContextPersistenceManager) SaveContext(
	id string,
	name string,
	primaryModelID string,
	messages []EnhancedMessage,
	summaries []Summary,
) error {
	if p.config.BaseDir == "" {
		return fmt.Errorf("base directory not configured")
	}

	// Filter system messages if configured
	messagesToSave := messages
	if !p.config.IncludeSystemMessages {
		var filtered []EnhancedMessage
		for _, msg := range messages {
			if msg.Type != MessageTypeSystem {
				filtered = append(filtered, msg)
			}
		}
		messagesToSave = filtered
	}

	// Create metadata
	metadata := ContextMetadata{
		ID:             id,
		Name:           name,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		MessageCount:   len(messagesToSave),
		SummaryCount:   len(summaries),
		PrimaryModelID: primaryModelID,
	}

	// Extract topics and tags from messages
	topicMap := make(map[string]bool)
	tagMap := make(map[string]bool)
	totalTokens := 0

	for _, msg := range messagesToSave {
		// Add topics
		for _, topic := range msg.Topics {
			topicMap[topic] = true
		}

		// Add tags
		for _, tag := range msg.Tags {
			tagMap[tag] = true
		}

		// Count tokens
		totalTokens += msg.TokenCount
	}

	// Convert maps to slices
	for topic := range topicMap {
		metadata.Topics = append(metadata.Topics, topic)
	}

	for tag := range tagMap {
		metadata.Tags = append(metadata.Tags, tag)
	}

	metadata.SessionTokens = totalTokens

	// Create data object
	data := ContextData{
		Metadata:  metadata,
		Messages:  messagesToSave,
		Summaries: summaries,
		Version:   "1.0.0",
	}

	// Create the file path - make sure ID doesn't have extension
	baseID := extractContextID(id)
	filePath := filepath.Join(p.config.BaseDir, baseID+".json")

	// Create the directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context data: %w", err)
	}

	// Write to file
	if p.config.Compress {
		// Log compression info
		persistenceLogger.Printf("Compressing context data (%d bytes) to %s.gz", len(jsonData), filePath)

		// Open file for writing
		file, err := os.OpenFile(filePath+".gz", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, p.config.FileMode)
		if err != nil {
			return fmt.Errorf("failed to create compressed context file: %w", err)
		}
		defer file.Close()

		// Create gzip writer
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()

		// Write data
		if _, err := gzWriter.Write(jsonData); err != nil {
			return fmt.Errorf("failed to write compressed context data: %w", err)
		}

		// Make sure to flush the gzip writer BEFORE closing
		if err := gzWriter.Flush(); err != nil {
			return fmt.Errorf("failed to flush compressed data: %w", err)
		}

		persistenceLogger.Printf("Successfully compressed and saved context data to %s.gz", filePath)
	} else {
		// Write directly to file without compression
		persistenceLogger.Printf("Saving uncompressed context data (%d bytes) to %s", len(jsonData), filePath)

		if err := os.WriteFile(filePath, jsonData, p.config.FileMode); err != nil {
			return fmt.Errorf("failed to write context file: %w", err)
		}

		persistenceLogger.Printf("Successfully saved context data to %s", filePath)
	}

	// Clean up old files if needed
	return p.cleanupOldFiles()
}

// LoadContext loads context data from disk
func (p *ContextPersistenceManager) LoadContext(id string) (*ContextData, error) {
	persistenceLogger.Printf("LoadContext: Loading context with ID: %s", id)

	// Try all possible file variations
	variations := []string{
		// Try original ID (no processing)
		filepath.Join(p.config.BaseDir, id),
		// Try with .json extension
		filepath.Join(p.config.BaseDir, id+".json"),
		// Try with .json.gz extension
		filepath.Join(p.config.BaseDir, id+".json.gz"),
		// Try with .gz extension
		filepath.Join(p.config.BaseDir, id+".gz"),
		// Try extracting ID then adding .json
		filepath.Join(p.config.BaseDir, extractContextID(id)+".json"),
		// Try extracting ID then adding .json.gz
		filepath.Join(p.config.BaseDir, extractContextID(id)+".json.gz"),
	}

	persistenceLogger.Printf("LoadContext: Will try these file paths: %v", variations)

	var file *os.File
	var filePath string
	compressed := false

	// Try each variation until we find one that exists
	for _, path := range variations {
		persistenceLogger.Printf("Trying path: %s", path)
		if _, err := os.Stat(path); err == nil {
			filePath = path
			compressed = strings.HasSuffix(path, ".gz")
			persistenceLogger.Printf("Found file at %s (compressed=%v)", filePath, compressed)

			// Open the file
			var openErr error
			file, openErr = os.Open(filePath)
			if openErr != nil {
				persistenceLogger.Printf("ERROR: Found file but failed to open: %v", openErr)
				continue
			}

			// Successfully opened the file
			break
		} else {
			persistenceLogger.Printf("File not found at %s", path)
		}
	}

	// If no file was found or opened
	if file == nil {
		persistenceLogger.Printf("ERROR: No context file found for ID %s after trying all variations", id)
		return nil, fmt.Errorf("context file not found for ID: %s", id)
	}

	defer file.Close()

	// Create reader based on compression
	var reader io.Reader = file

	if compressed {
		persistenceLogger.Printf("Creating gzip reader for compressed file")
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			persistenceLogger.Printf("ERROR: Failed to create gzip reader: %v", err)
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Read data
	jsonData, err := io.ReadAll(reader)
	if err != nil {
		persistenceLogger.Printf("ERROR: Failed to read context data: %v", err)
		return nil, fmt.Errorf("failed to read context data: %w", err)
	}

	// Unmarshal data
	var data ContextData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		persistenceLogger.Printf("ERROR: Failed to unmarshal context data: %v", err)
		// Log first 100 bytes of the data to help diagnose
		if len(jsonData) > 100 {
			persistenceLogger.Printf("Data preview (first 100 bytes): %s", string(jsonData[:100]))
		} else {
			persistenceLogger.Printf("Data preview (all %d bytes): %s", len(jsonData), string(jsonData))
		}
		return nil, fmt.Errorf("failed to unmarshal context data: %w", err)
	}

	persistenceLogger.Printf("Successfully unmarshalled context data: ID=%s, MessageCount=%d, SummaryCount=%d",
		data.Metadata.ID, data.Metadata.MessageCount, len(data.Summaries))

	return &data, nil
}

// ListContexts lists available context files
func (p *ContextPersistenceManager) ListContexts() ([]ContextMetadata, error) {
	if p.config.BaseDir == "" {
		persistenceLogger.Printf("ERROR: Base directory not configured for persistence")
		return nil, fmt.Errorf("base directory not configured")
	}

	persistenceLogger.Printf("ListContexts: Looking for context files in %s", p.config.BaseDir)

	// Check if the directory exists
	if _, err := os.Stat(p.config.BaseDir); os.IsNotExist(err) {
		persistenceLogger.Printf("ListContexts: Directory %s does not exist", p.config.BaseDir)
		return []ContextMetadata{}, nil
	}

	persistenceLogger.Printf("ListContexts: Directory %s exists, reading contents", p.config.BaseDir)

	// Read directory
	entries, err := os.ReadDir(p.config.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read context directory: %w", err)
	}

	// Find context files
	var results []ContextMetadata

	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Check if it's a context file
		name := entry.Name()
		persistenceLogger.Printf("DEBUG: Found file: %s", name)

		// Only process files that look like context files
		if !strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".json.gz") && !strings.HasSuffix(name, ".gz") {
			persistenceLogger.Printf("DEBUG: Skipping non-context file: %s", name)
			continue
		}

		// Extract the ID using our helper function
		id := extractContextID(name)
		persistenceLogger.Printf("DEBUG: File name %s extracted to ID %s", name, id)

		// Load the file directly instead of using LoadContext
		compressed := strings.HasSuffix(name, ".gz")
		filePath := filepath.Join(p.config.BaseDir, name)

		// Open the file directly
		persistenceLogger.Printf("Attempting to open file directly: %s", filePath)
		file, err := os.Open(filePath)
		if err != nil {
			persistenceLogger.Printf("ERROR: Failed to open context file %s: %v", filePath, err)
			continue
		}

		// Create reader based on compression
		var reader io.Reader = file

		if compressed {
			gzReader, err := gzip.NewReader(file)
			if err != nil {
				file.Close()
				persistenceLogger.Printf("ERROR: Failed to create gzip reader: %v", err)
				continue
			}
			defer gzReader.Close()
			reader = gzReader
		}

		// Read data
		jsonData, err := io.ReadAll(reader)
		file.Close() // Close the file after reading

		if err != nil {
			persistenceLogger.Printf("ERROR: Failed to read context data: %v", err)
			continue
		}

		// Unmarshal data
		var data ContextData
		if err := json.Unmarshal(jsonData, &data); err != nil {
			persistenceLogger.Printf("ERROR: Failed to unmarshal context data: %v", err)
			continue
		}

		persistenceLogger.Printf("Successfully loaded metadata for context %s: Name=%s, MessageCount=%d",
			id, data.Metadata.Name, data.Metadata.MessageCount)
		results = append(results, data.Metadata)
	}

	// Sort by updated time (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].UpdatedAt.After(results[j].UpdatedAt)
	})

	return results, nil
}

// DeleteContext deletes a saved context file
func (p *ContextPersistenceManager) DeleteContext(id string) error {
	// Extract base ID without extensions
	baseID := extractContextID(id)

	// Try all possible file variations to ensure complete cleanup
	variations := []string{
		// Try with .json extension
		filepath.Join(p.config.BaseDir, baseID+".json"),
		// Try with .json.gz extension
		filepath.Join(p.config.BaseDir, baseID+".json.gz"),
		// Try with just .gz extension
		filepath.Join(p.config.BaseDir, baseID+".gz"),
		// Try with original ID
		filepath.Join(p.config.BaseDir, id),
		// Try with original ID and .json
		filepath.Join(p.config.BaseDir, id+".json"),
		// Try with original ID and .gz
		filepath.Join(p.config.BaseDir, id+".gz"),
		// Try with original ID and .json.gz
		filepath.Join(p.config.BaseDir, id+".json.gz"),
	}

	// Track if at least one deletion was successful
	anySuccess := false
	var lastError error

	// Try to delete all variations
	for _, path := range variations {
		err := os.Remove(path)
		if err == nil {
			persistenceLogger.Printf("Successfully deleted context file: %s", path)
			anySuccess = true
		} else if !os.IsNotExist(err) {
			// Only care about errors other than "file not found"
			persistenceLogger.Printf("Error deleting context file %s: %v", path, err)
			lastError = err
		}
	}

	if anySuccess {
		return nil
	}

	if lastError != nil {
		return fmt.Errorf("failed to delete context file: %w", lastError)
	}

	// If we got here, all files were not found, which is fine
	return nil
}

// extractContextID extracts the base ID from a filename by removing extensions
func extractContextID(filename string) string {
	// Start with the full filename
	id := filename

	// Remove .gz extension if present
	if strings.HasSuffix(id, ".gz") {
		id = id[:len(id)-3]
		persistenceLogger.Printf("DEBUG: Removed .gz extension, now: %s", id)
	}

	// Remove .json extension if present
	if strings.HasSuffix(id, ".json") {
		id = id[:len(id)-5]
		persistenceLogger.Printf("DEBUG: Removed .json extension, now: %s", id)
	}

	return id
}

// cleanupOldFiles removes old files to stay within limits
func (p *ContextPersistenceManager) cleanupOldFiles() error {
	// Skip if no limits are set
	if p.config.MaxContextFiles <= 0 && p.config.MaxRetentionDays <= 0 {
		return nil
	}

	// Get all contexts
	contexts, err := p.ListContexts()
	if err != nil {
		return fmt.Errorf("failed to list contexts: %w", err)
	}

	// If we're under the limit, nothing to do
	if p.config.MaxContextFiles <= 0 || len(contexts) <= p.config.MaxContextFiles {
		// No need to delete based on count, but still check date
	} else {
		// Need to delete oldest contexts
		toDelete := len(contexts) - p.config.MaxContextFiles

		// Delete the oldest ones
		for i := len(contexts) - 1; i >= len(contexts)-toDelete; i-- {
			if err := p.DeleteContext(contexts[i].ID); err != nil {
				// Log but continue
				// Log to file instead of stderr
				persistenceLogger.Printf("Error deleting context %s: %v", contexts[i].ID, err)
			}
		}
	}

	// Now check for old files
	if p.config.MaxRetentionDays > 0 {
		cutoffTime := time.Now().AddDate(0, 0, -p.config.MaxRetentionDays)

		for _, ctx := range contexts {
			if ctx.UpdatedAt.Before(cutoffTime) {
				if err := p.DeleteContext(ctx.ID); err != nil {
					// Log but continue
					// Log to file instead of stderr
					persistenceLogger.Printf("Error deleting old context %s: %v", ctx.ID, err)
				}
			}
		}
	}

	return nil
}
