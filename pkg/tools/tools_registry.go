package tools

import (
	"github.com/navicore/mcpterm-go/pkg/tools/categories/development"
	"github.com/navicore/mcpterm-go/pkg/tools/categories/filesystem"
)

// registerFilesystemTools registers filesystem tools
func registerFilesystemTools(registry *Registry) error {
	return filesystem.Register(registry)
}

// registerDevelopmentTools registers development tools
func registerDevelopmentTools(registry *Registry) error {
	return development.Register(registry)
}

// registerCustomerSupportTools would register customer support tools
// func registerCustomerSupportTools(registry *Registry) error {
//     return customersupport.Register(registry)
// }
