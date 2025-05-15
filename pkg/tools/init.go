package tools

import (
	"fmt"
)

// InitializeTools registers all tools with the registry
func InitializeTools(registry *Registry) error {
	// Load filesystem tools
	if err := registerFilesystemTools(registry); err != nil {
		return fmt.Errorf("failed to register filesystem tools: %w", err)
	}

	// Load development tools
	if err := registerDevelopmentTools(registry); err != nil {
		return fmt.Errorf("failed to register development tools: %w", err)
	}

	// Customer support tools would be added here
	// if err := registerCustomerSupportTools(registry); err != nil {
	//     return fmt.Errorf("failed to register customer_support tools: %w", err)
	// }

	return nil
}

// Initialize creates a fully initialized tool manager with all tools registered
func Initialize() (*ToolManager, error) {
	manager := NewToolManager()

	// Register all tools
	if err := InitializeTools(manager.registry); err != nil {
		return nil, fmt.Errorf("failed to initialize tools: %w", err)
	}

	return manager, nil
}
