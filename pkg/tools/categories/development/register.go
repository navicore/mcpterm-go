package development

import (
	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// Register registers development tools with the registry
func Register(registry core.ToolRegistrar) error {
	// Register shell tool
	if err := registry.RegisterTool("development", NewShellTool()); err != nil {
		return err
	}

	// Register file_write tool
	if err := registry.RegisterTool("development", NewFileWriteTool()); err != nil {
		return err
	}

	// Register patch tool
	if err := registry.RegisterTool("development", NewPatchTool()); err != nil {
		return err
	}

	// Register diff tool
	if err := registry.RegisterTool("development", NewDiffTool()); err != nil {
		return err
	}

	return nil
}
