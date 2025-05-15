package filesystem

import (
	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// Register registers all filesystem tools with the registry
func Register(registry core.ToolRegistrar) error {
	// Register file_read tool
	if err := registry.RegisterTool("filesystem", NewFileReadTool()); err != nil {
		return err
	}

	// Register find tool
	if err := registry.RegisterTool("filesystem", NewFindTool()); err != nil {
		return err
	}

	// Register directory_list tool
	if err := registry.RegisterTool("filesystem", NewDirectoryListTool()); err != nil {
		return err
	}

	// Register mkdir tool
	if err := registry.RegisterTool("filesystem", NewMkdirTool()); err != nil {
		return err
	}

	// Register file_delete tool
	if err := registry.RegisterTool("filesystem", NewFileDeleteTool()); err != nil {
		return err
	}

	// Register file_rename tool
	if err := registry.RegisterTool("filesystem", NewFileRenameTool()); err != nil {
		return err
	}

	// Register grep tool
	if err := registry.RegisterTool("filesystem", NewGrepTool()); err != nil {
		return err
	}

	return nil
}
