# Tool Architecture Design

This document describes the architecture for the hierarchical tool system in MCPTerm, allowing for different categories of tools with varying permission levels and capabilities.

## Architecture Overview

The tool system follows these key design principles:

1. **Categorized Tools**: Tools are organized into logical categories (e.g., filesystem, development, customer support)
2. **Pluggable**: New tool categories can be added without modifying core code
3. **Permission Control**: Different permission levels for different tool categories
4. **Consistent Interface**: All tools follow the same interface pattern regardless of category

## Core Components

### 1. Tool Registry

The central registry for all tools and categories:

```go
// pkg/tools/registry.go

// Category represents a group of related tools
type Category struct {
    ID          string
    Name        string
    Description string
    Enabled     bool
    Permission  PermissionLevel
    Tools       []Tool
}

// PermissionLevel defines access rights for a tool category
type PermissionLevel string

const (
    PermissionReadOnly  PermissionLevel = "read-only"   // Can only read data
    PermissionReadWrite PermissionLevel = "read-write"  // Can read and write data
    PermissionExecute   PermissionLevel = "execute"     // Can execute commands
)

// Registry manages all tool categories and their tools
type Registry struct {
    Categories map[string]*Category
}

// RegisterCategory adds a new category to the registry
func (r *Registry) RegisterCategory(cat *Category) error { /* ... */ }

// RegisterTool adds a tool to a specific category
func (r *Registry) RegisterTool(categoryID string, tool Tool) error { /* ... */ }

// GetEnabledTools returns all tools from enabled categories
func (r *Registry) GetEnabledTools() []ClaudeTool { /* ... */ }

// NewRegistry creates a new tool registry with default categories
func NewRegistry() *Registry {
    r := &Registry{Categories: make(map[string]*Category)}
    
    // Register default categories
    r.RegisterCategory(&Category{
        ID:          "filesystem",
        Name:        "Filesystem Tools",
        Description: "Tools for interacting with the local filesystem",
        Enabled:     true,
        Permission:  PermissionReadOnly,
    })
    
    r.RegisterCategory(&Category{
        ID:          "development",
        Name:        "Development Tools",
        Description: "Tools for development tasks",
        Enabled:     false, // Disabled by default
        Permission:  PermissionReadWrite,
    })
    
    r.RegisterCategory(&Category{
        ID:          "customer_support",
        Name:        "Customer Support Tools",
        Description: "Tools for accessing customer data",
        Enabled:     false, // Disabled by default
        Permission:  PermissionReadWrite,
    })
    
    return r
}
```

### 2. Tool Interface

A common interface for all tools:

```go
// pkg/tools/tool.go

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

func (t *BaseToolImpl) Name() string { return t.name }
func (t *BaseToolImpl) Description() string { return t.description }
func (t *BaseToolImpl) Category() string { return t.category }
func (t *BaseToolImpl) InputSchema() map[string]interface{} { return t.inputSchema }
```

### 3. Tool Manager

Manages tool execution and permissions:

```go
// pkg/tools/manager.go

// ToolManager handles tool operations and permissions
type ToolManager struct {
    registry    *Registry
    enabledCats map[string]bool
}

// NewToolManager creates a new tool manager
func NewToolManager() *ToolManager {
    return &ToolManager{
        registry:    NewRegistry(),
        enabledCats: make(map[string]bool),
    }
}

// EnableCategory enables a specific tool category
func (tm *ToolManager) EnableCategory(categoryID string, enabled bool) {
    if cat, exists := tm.registry.Categories[categoryID]; exists {
        cat.Enabled = enabled
        tm.enabledCats[categoryID] = enabled
    }
}

// EnableAllCategories enables or disables all tool categories
func (tm *ToolManager) EnableAllCategories(enabled bool) {
    for id, cat := range tm.registry.Categories {
        cat.Enabled = enabled
        tm.enabledCats[id] = enabled
    }
}

// GetEnabledTools returns tool definitions for all enabled tools
func (tm *ToolManager) GetEnabledTools() []backend.ClaudeTool {
    return tm.registry.GetEnabledTools()
}

// ExecuteTool executes a specific tool with given input
func (tm *ToolManager) ExecuteTool(toolName string, input json.RawMessage) (interface{}, error) {
    // Find the tool across all categories
    for _, cat := range tm.registry.Categories {
        if !cat.Enabled {
            continue
        }
        
        for _, tool := range cat.Tools {
            if tool.Name() == toolName {
                return tool.Execute(input)
            }
        }
    }
    
    return nil, fmt.Errorf("unknown tool: %s", toolName)
}
```

### 4. Tool Implementations

Organize tool implementations by category:

```go
// Directory structure:
// pkg/
//   tools/
//     registry.go
//     manager.go
//     tool.go
//     categories/
//       filesystem/
//         find.go
//         file_read.go
//         directory_list.go
//       development/
//         shell.go
//         patch.go
//         file_write.go
//         git.go
//       customer_support/
//         user.go
//         tenant.go
//         ticket.go
```

Example tool implementation:

```go
// pkg/tools/categories/filesystem/file_read.go

package filesystem

import (
    "encoding/json"
    "fmt"
    "os"
    "strings"
    
    "github.com/navicore/coder-go/pkg/tools"
)

// FileReadTool allows reading file contents
type FileReadTool struct {
    tools.BaseToolImpl
}

// FileReadInput represents parameters for reading a file
type FileReadInput struct {
    Path   string `json:"path"`
    Offset int    `json:"offset,omitempty"`
    Limit  int    `json:"limit,omitempty"`
}

// NewFileReadTool creates a new file read tool
func NewFileReadTool() *FileReadTool {
    return &FileReadTool{
        BaseToolImpl: tools.BaseToolImpl{
            name:        "file_read",
            description: "Read the contents of a file",
            category:    "filesystem",
            inputSchema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "path": map[string]interface{}{
                        "type":        "string",
                        "description": "Path to the file to read",
                    },
                    "offset": map[string]interface{}{
                        "type":        "integer",
                        "description": "Line number to start reading from (0-based)",
                    },
                    "limit": map[string]interface{}{
                        "type":        "integer",
                        "description": "Maximum number of lines to read",
                    },
                },
                "required": []string{"path"},
            },
        },
    }
}

// Execute reads a file based on the provided input
func (t *FileReadTool) Execute(input json.RawMessage) (interface{}, error) {
    var params FileReadInput
    if err := json.Unmarshal(input, &params); err != nil {
        return nil, fmt.Errorf("invalid input for file_read tool: %w", err)
    }
    
    // Validate path
    if params.Path == "" {
        return nil, fmt.Errorf("path parameter is required")
    }
    
    // Expand ~ to home directory if present
    if strings.HasPrefix(params.Path, "~") {
        homeDir, err := os.UserHomeDir()
        if err != nil {
            return nil, fmt.Errorf("failed to expand home directory: %w", err)
        }
        params.Path = strings.Replace(params.Path, "~", homeDir, 1)
    }
    
    // Read file
    content, err := os.ReadFile(params.Path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }
    
    // Convert to string and split into lines
    lines := strings.Split(string(content), "\n")
    
    // Apply offset and limit if provided
    if params.Offset > 0 || params.Limit > 0 {
        if params.Offset >= len(lines) {
            return "", nil
        }
        
        end := len(lines)
        if params.Limit > 0 && params.Offset+params.Limit < end {
            end = params.Offset + params.Limit
        }
        
        lines = lines[params.Offset:end]
    }
    
    // Join lines back together
    return strings.Join(lines, "\n"), nil
}

// Register registers this tool with the registry
func Register(registry *tools.Registry) {
    registry.RegisterTool("filesystem", NewFileReadTool())
}
```

### 5. Configuration and CLI Updates

Update configuration and CLI to support tool categories:

```go
// pkg/config/config.go

type ToolsConfig struct {
    EnableAll      bool              `json:"enable_all"`
    EnabledCategories map[string]bool `json:"enabled_categories"`
}

type Config struct {
    // ...existing fields
    Tools ToolsConfig `json:"tools"`
}

// cmd/coder/root.go

var (
    // ...existing flags
    enableTools        bool
    enabledCategories  string  // Comma-separated list of enabled categories
)

func init() {
    // ...existing flag setup
    
    // Tools flags
    rootCmd.PersistentFlags().BoolVar(&enableTools, "enable-tools", true, "Enable system tools for the LLM")
    rootCmd.PersistentFlags().StringVar(&enabledCategories, "enable-tool-categories", "", "Comma-separated list of tool categories to enable (e.g., filesystem,development)")
}
```

### 6. Integration with Chat Service

Update the chat service to work with the new tool system:

```go
// pkg/chat/service.go

// ChatService provides conversational capabilities
type ChatService struct {
    // ...existing fields
    toolManager    *tools.ToolManager
}

// NewChatService creates a new chat service
func NewChatService(opts ChatOptions) (*ChatService, error) {
    // ...existing code
    
    // Create tool manager
    toolManager := tools.NewToolManager()
    
    // Configure tool categories based on options
    if opts.EnableTools {
        if opts.EnabledToolCategories != nil && len(opts.EnabledToolCategories) > 0 {
            // Enable specific categories
            for _, cat := range opts.EnabledToolCategories {
                toolManager.EnableCategory(cat, true)
            }
        } else {
            // Enable all categories
            toolManager.EnableAllCategories(true)
        }
    } else {
        // Disable all categories
        toolManager.EnableAllCategories(false)
    }
    
    return &ChatService{
        // ...existing fields
        toolManager:  toolManager,
    }, nil
}
```

## Registration and Initialization

Tool categories are registered at startup:

```go
// pkg/tools/init.go

// Import all tool categories
import (
    "github.com/navicore/coder-go/pkg/tools/categories/filesystem"
    "github.com/navicore/coder-go/pkg/tools/categories/development"
    "github.com/navicore/coder-go/pkg/tools/categories/customer_support"
)

// Initialize registers all tools with the registry
func Initialize(registry *Registry) {
    // Register tools from each category
    filesystem.Register(registry)
    development.Register(registry)
    customer_support.Register(registry)
}
```

## Migration Path

To migrate from the current system to the new hierarchical system:

1. Create the new registry and tool manager
2. Convert existing tool implementations to the new interface
3. Organize tools into appropriate category packages
4. Update configuration to support enabling/disabling categories
5. Update CLI flags to support category-specific enabling
6. Update chat service to use the new tool manager

## Security Considerations

1. **Permission Levels**: Each category has a permission level that determines what actions its tools can perform
2. **Confirmation Prompts**: For write operations or high-risk actions, consider adding confirmation prompts
3. **Audit Logging**: Log all tool executions for security review
4. **Rate Limiting**: Apply per-category rate limits for tool usage
5. **Sandboxing**: Consider running high-risk tools in a sandboxed environment

## Next Steps

1. Implement the core registry and manager
2. Refactor existing filesystem tools to use the new architecture
3. Add configuration support for tool categories
4. Implement development tools category
5. Implement customer support tools category
6. Add comprehensive testing for all components
