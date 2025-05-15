# MCPTerm Tool System

This package contains the hierarchical tool system that allows Claude to interact with your local system in a controlled manner.

## Architecture

The tool system is organized into categories, each containing related tools:

- `filesystem` - Tools for interacting with the local filesystem (find, file_read, directory_list)
- `development` - Tools for development tasks (shell command execution)
- `customer_support` - Tools for accessing customer data (planned)

## Testing

All tools have unit tests that can be run with Go's built-in test framework:

```bash
# Run all tool tests
go test ./pkg/tools/... -v

# Run only filesystem tool tests
go test ./pkg/tools/categories/filesystem -v

# Run only development tool tests
go test ./pkg/tools/categories/development -v
```

## Adding New Tools

To add a new tool:

1. Identify which category the tool belongs in (or create a new category)
2. Create a new tool implementation file in the appropriate category directory
3. Implement the `core.Tool` interface
4. Add tests for your tool
5. Register the tool in the category's `register.go` file

Example of a new tool implementation:

```go
package myCategory

import (
	"encoding/json"
	"fmt"

	"github.com/navicore/coder-go/pkg/tools/core"
)

// MyToolInput represents parameters for my tool
type MyToolInput struct {
	Param1 string `json:"param1"`
	Param2 int    `json:"param2,omitempty"`
}

// MyTool implements a custom tool
type MyTool struct {
	core.BaseToolImpl
}

// NewMyTool creates a new instance of MyTool
func NewMyTool() *MyTool {
	tool := &MyTool{}
	tool.BaseToolImpl = *core.NewBaseTool(
		"my_tool",
		"Description of what my tool does",
		"myCategory",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "Description of param1",
				},
				"param2": map[string]interface{}{
					"type":        "integer",
					"description": "Description of param2",
				},
			},
			"required": []string{"param1"},
		},
	)
	return tool
}

// Execute implements the Tool interface
func (t *MyTool) Execute(input json.RawMessage) (interface{}, error) {
	var params MyToolInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input for my_tool: %w", err)
	}

	// Validate parameters
	if params.Param1 == "" {
		return nil, fmt.Errorf("param1 is required")
	}

	// Implement tool logic here
	result := fmt.Sprintf("Processed %s with param2=%d", params.Param1, params.Param2)
	
	return result, nil
}
```

Then register it in your category's `register.go`:

```go
func Register(registry core.ToolRegistrar) error {
	// Register existing tools...
	
	// Register your new tool
	if err := registry.RegisterTool("myCategory", NewMyTool()); err != nil {
		return err
	}
	
	return nil
}
```
