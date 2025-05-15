package core

// ToolRegistrar defines the interface for registering tools
type ToolRegistrar interface {
	// RegisterTool adds a tool to a specific category
	RegisterTool(categoryID string, tool Tool) error
}
