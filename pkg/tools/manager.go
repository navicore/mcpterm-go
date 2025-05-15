package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/navicore/mcpterm-go/pkg/backend"
	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// ToolManager handles tool execution and permissions
type ToolManager struct {
	mu             sync.RWMutex
	registry       *Registry
	enabledCats    map[string]bool
	toolsEnabled   bool
	maxToolsPerMsg int
}

// NewToolManager creates a new tool manager with default settings
func NewToolManager() *ToolManager {
	return &ToolManager{
		registry:       NewRegistry(),
		enabledCats:    make(map[string]bool),
		toolsEnabled:   true, // Enabled by default
		maxToolsPerMsg: 10,   // Default limit
	}
}

// EnableTools enables or disables all tools
func (tm *ToolManager) EnableTools(enabled bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.toolsEnabled = enabled
}

// IsToolsEnabled returns whether tools are enabled
func (tm *ToolManager) IsToolsEnabled() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.toolsEnabled
}

// EnableCategory enables a specific tool category
func (tm *ToolManager) EnableCategory(categoryID string, enabled bool) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	err := tm.registry.SetCategoryEnabled(categoryID, enabled)
	if err != nil {
		return err
	}

	tm.enabledCats[categoryID] = enabled
	return nil
}

// EnableCategoriesByIDs enables specific categories by their IDs
func (tm *ToolManager) EnableCategoriesByIDs(categoryIDs []string) error {
	// First disable all categories
	tm.registry.SetAllCategoriesEnabled(false)

	// Enable only the specified categories
	for _, catID := range categoryIDs {
		if err := tm.EnableCategory(catID, true); err != nil {
			return err
		}
	}

	return nil
}

// EnableCategories parses a comma-separated list of category IDs and enables them
func (tm *ToolManager) EnableCategories(categoriesStr string) error {
	if categoriesStr == "" {
		return nil
	}

	categories := strings.Split(categoriesStr, ",")
	return tm.EnableCategoriesByIDs(categories)
}

// EnableAllCategories enables or disables all tool categories
func (tm *ToolManager) EnableAllCategories(enabled bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.registry.SetAllCategoriesEnabled(enabled)

	// Update the enabled categories map
	for id := range tm.registry.Categories {
		tm.enabledCats[id] = enabled
	}
}

// GetTools returns tool definitions for all enabled tools
func (tm *ToolManager) GetTools() []backend.ClaudeTool {
	if !tm.IsToolsEnabled() {
		return nil
	}

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.registry.GetEnabledTools()
}

// HandleToolUse processes a tool use request
func (tm *ToolManager) HandleToolUse(toolUse *core.ToolUse) (*core.ToolResult, error) {
	if !tm.IsToolsEnabled() {
		return nil, fmt.Errorf("tool use is disabled")
	}

	if toolUse == nil {
		return nil, fmt.Errorf("no tool use request provided")
	}

	// Get the tool from the registry
	tool, err := tm.registry.GetTool(toolUse.Name)
	if err != nil {
		return nil, fmt.Errorf("error finding tool %s: %w", toolUse.Name, err)
	}

	// Execute the tool
	result, err := tool.Execute(toolUse.Input)
	if err != nil {
		return nil, fmt.Errorf("error executing tool %s: %w", toolUse.Name, err)
	}

	// Convert result to JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling tool result: %w", err)
	}

	// Return as tool result
	return &core.ToolResult{
		Name:   toolUse.Name,
		Result: resultJSON,
	}, nil
}

// SetMaxToolsPerMsg sets the maximum number of tool calls allowed per message
func (tm *ToolManager) SetMaxToolsPerMsg(max int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if max > 0 {
		tm.maxToolsPerMsg = max
	}
}

// GetMaxToolsPerMsg gets the maximum number of tool calls allowed per message
func (tm *ToolManager) GetMaxToolsPerMsg() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.maxToolsPerMsg
}

// RegisterTool registers a new tool with the manager
func (tm *ToolManager) RegisterTool(categoryID string, tool core.Tool) error {
	return tm.registry.RegisterTool(categoryID, tool)
}
