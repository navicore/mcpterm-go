package chat

import (
	"encoding/json"
	"fmt"

	"github.com/navicore/mcpterm-go/pkg/backend"
	"github.com/navicore/mcpterm-go/pkg/tools"
)

// ToolUser interface for things that can execute tools
type ToolUser interface {
	GetToolDefinitions() []backend.ClaudeTool
	ExecuteTool(toolName string, input json.RawMessage) (interface{}, error)
}

// ToolManager handles tool integration with chat service
type ToolManager struct {
	toolUser     ToolUser
	toolsEnabled bool
}

// NewToolManager creates a new tool manager
func NewToolManager() *ToolManager {
	return &ToolManager{
		toolUser:     tools.NewToolExecutor(),
		toolsEnabled: true,
	}
}

// EnableTools enables or disables tool usage
func (tm *ToolManager) EnableTools(enabled bool) {
	tm.toolsEnabled = enabled
}

// IsToolsEnabled returns whether tools are enabled
func (tm *ToolManager) IsToolsEnabled() bool {
	return tm.toolsEnabled
}

// GetTools returns tool definitions for the chat service
func (tm *ToolManager) GetTools() []backend.ClaudeTool {
	if !tm.toolsEnabled {
		return nil
	}
	return tm.toolUser.GetToolDefinitions()
}

// HandleToolUse processes a tool use request from the LLM
func (tm *ToolManager) HandleToolUse(toolUse *backend.ToolUse) (*backend.ToolResult, error) {
	if !tm.toolsEnabled {
		return nil, fmt.Errorf("tool use is disabled")
	}

	if toolUse == nil {
		return nil, fmt.Errorf("no tool use request provided")
	}

	// Execute the tool
	result, err := tm.toolUser.ExecuteTool(toolUse.Name, toolUse.Input)
	if err != nil {
		return nil, fmt.Errorf("error executing tool %s: %w", toolUse.Name, err)
	}

	// Convert result to JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling tool result: %w", err)
	}

	// Return as tool result
	return &backend.ToolResult{
		Name:   toolUse.Name,
		Result: resultJSON,
	}, nil
}
