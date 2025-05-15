package core

import (
	"encoding/json"

	"github.com/navicore/mcpterm-go/pkg/backend"
)

// PermissionLevel defines access rights for a tool category
type PermissionLevel string

const (
	PermissionReadOnly  PermissionLevel = "read-only"  // Can only read data
	PermissionReadWrite PermissionLevel = "read-write" // Can read and write data
	PermissionExecute   PermissionLevel = "execute"    // Can execute commands
)

// Tool represents a capability that can be provided to Claude
type Tool interface {
	// Name returns the name of the tool as seen by Claude
	Name() string

	// Description returns the description of the tool as seen by Claude
	Description() string

	// Category returns the category this tool belongs to
	Category() string

	// InputSchema returns the JSON schema for the tool's input
	InputSchema() map[string]interface{}

	// Execute performs the tool operation with given input
	Execute(input json.RawMessage) (interface{}, error)
}

// BaseToolImpl provides common functionality for tool implementations
type BaseToolImpl struct {
	name        string
	description string
	category    string
	inputSchema map[string]interface{}
}

// Name returns the name of the tool
func (t *BaseToolImpl) Name() string { return t.name }

// Description returns the description of the tool
func (t *BaseToolImpl) Description() string { return t.description }

// Category returns the category this tool belongs to
func (t *BaseToolImpl) Category() string { return t.category }

// InputSchema returns the JSON schema for the tool's input
func (t *BaseToolImpl) InputSchema() map[string]interface{} { return t.inputSchema }

// NewBaseTool creates a new basic tool implementation
func NewBaseTool(name, description, category string, schema map[string]interface{}) *BaseToolImpl {
	return &BaseToolImpl{
		name:        name,
		description: description,
		category:    category,
		inputSchema: schema,
	}
}

// ToolResult represents the result of a tool execution
type ToolResult = backend.ToolResult

// ToolUse represents a tool use request from the LLM
type ToolUse = backend.ToolUse

// ClaudeTool represents a tool definition for Claude models
type ClaudeTool = backend.ClaudeTool
