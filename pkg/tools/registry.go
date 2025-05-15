package tools

import (
	"fmt"
	"sync"

	"github.com/navicore/mcpterm-go/pkg/backend"
	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// Category represents a group of related tools
type Category struct {
	ID          string
	Name        string
	Description string
	Enabled     bool
	Permission  core.PermissionLevel
	Tools       []core.Tool
}

// Registry manages all tool categories and their tools
// It implements the core.ToolRegistrar interface
type Registry struct {
	mu         sync.RWMutex
	Categories map[string]*Category
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	r := &Registry{
		Categories: make(map[string]*Category),
	}

	// Register default categories
	r.RegisterCategory(&Category{
		ID:          "filesystem",
		Name:        "Filesystem Tools",
		Description: "Tools for interacting with the local filesystem",
		Enabled:     true,
		Permission:  core.PermissionReadOnly,
		Tools:       []core.Tool{},
	})

	r.RegisterCategory(&Category{
		ID:          "development",
		Name:        "Development Tools",
		Description: "Tools for development tasks",
		Enabled:     false, // Disabled by default
		Permission:  core.PermissionReadWrite,
		Tools:       []core.Tool{},
	})

	r.RegisterCategory(&Category{
		ID:          "customer_support",
		Name:        "Customer Support Tools",
		Description: "Tools for accessing customer data",
		Enabled:     false, // Disabled by default
		Permission:  core.PermissionReadWrite,
		Tools:       []core.Tool{},
	})

	return r
}

// RegisterCategory adds a new category to the registry
func (r *Registry) RegisterCategory(cat *Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Categories[cat.ID]; exists {
		return fmt.Errorf("category with ID %s already exists", cat.ID)
	}

	r.Categories[cat.ID] = cat
	return nil
}

// RegisterTool adds a tool to a specific category
func (r *Registry) RegisterTool(categoryID string, tool core.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cat, exists := r.Categories[categoryID]
	if !exists {
		return fmt.Errorf("category with ID %s does not exist", categoryID)
	}

	// Check for tool name collision in the category
	for _, existingTool := range cat.Tools {
		if existingTool.Name() == tool.Name() {
			return fmt.Errorf("tool with name %s already exists in category %s", tool.Name(), categoryID)
		}
	}

	cat.Tools = append(cat.Tools, tool)
	return nil
}

// GetEnabledTools returns all tools from enabled categories
func (r *Registry) GetEnabledTools() []backend.ClaudeTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []backend.ClaudeTool

	for _, cat := range r.Categories {
		if !cat.Enabled {
			continue
		}

		for _, tool := range cat.Tools {
			result = append(result, backend.ClaudeTool{
				Name:        tool.Name(),
				Description: tool.Description(),
				InputSchema: tool.InputSchema(),
			})
		}
	}

	return result
}

// GetTool finds a tool by name across all categories
func (r *Registry) GetTool(name string) (core.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, cat := range r.Categories {
		if !cat.Enabled {
			continue
		}

		for _, tool := range cat.Tools {
			if tool.Name() == name {
				return tool, nil
			}
		}
	}

	return nil, fmt.Errorf("tool %s not found or not enabled", name)
}

// SetCategoryEnabled enables or disables an entire category
func (r *Registry) SetCategoryEnabled(categoryID string, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cat, exists := r.Categories[categoryID]
	if !exists {
		return fmt.Errorf("category with ID %s does not exist", categoryID)
	}

	cat.Enabled = enabled
	return nil
}

// SetAllCategoriesEnabled enables or disables all categories
func (r *Registry) SetAllCategoriesEnabled(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, cat := range r.Categories {
		cat.Enabled = enabled
	}
}
