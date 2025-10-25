package services

import (
	"mcp-sync/models"
	"runtime"
	"strings"
)

// WindowsService handles Windows-specific platform transformations
type WindowsService struct{}

// NewWindowsService creates a new Windows service instance
func NewWindowsService() *WindowsService {
	return &WindowsService{}
}

// IsWindows returns true if running on Windows
func (ws *WindowsService) IsWindows() bool {
	return runtime.GOOS == "windows"
}

// WrapNpxCommand wraps npx commands with cmd /c for Windows compatibility
func (ws *WindowsService) WrapNpxCommand(command string, args []interface{}) (string, []interface{}) {
	if !ws.IsWindows() {
		return command, args
	}

	// Check if command is npx
	if strings.HasPrefix(command, "npx ") || command == "npx" {
		if strings.HasPrefix(command, "npx ") {
			// npx with arguments combined in command
			return "cmd", []interface{}{"/c", command}
		} else {
			// npx as separate command with args
			newArgs := []interface{}{"/c", "npx"}
			newArgs = append(newArgs, args...)
			return "cmd", newArgs
		}
	}

	return command, args
}

// UnwrapNpxCommand unwraps cmd /c from npx commands (reverse operation)
func (ws *WindowsService) UnwrapNpxCommand(command string, args []interface{}) (string, []interface{}) {
	if command != "cmd" || len(args) < 2 {
		return command, args
	}

	// Check if first arg is /c
	if firstArg, ok := args[0].(string); !ok || firstArg != "/c" {
		return command, args
	}

	// Check if second arg starts with npx
	if secondArg, ok := args[1].(string); ok {
		if strings.HasPrefix(secondArg, "npx ") {
			// npx with arguments combined
			if len(args) > 2 {
				// Append additional args to the npx command
				additionalArgs := make([]string, 0)
				for i := 2; i < len(args); i++ {
					if argStr, ok := args[i].(string); ok {
						additionalArgs = append(additionalArgs, argStr)
					}
				}
				if len(additionalArgs) > 0 {
					secondArg += " " + strings.Join(additionalArgs, " ")
				}
			}
			return secondArg, []interface{}{}
		} else if secondArg == "npx" {
			// npx as command with separate args
			if len(args) > 2 {
				var remainingArgs []interface{}
				for i := 2; i < len(args); i++ {
					remainingArgs = append(remainingArgs, args[i])
				}
				return "npx", remainingArgs
			}
			return "npx", []interface{}{}
		}
	}

	return command, args
}

// IsNpxCommand checks if the command is an npx command (wrapped or unwrapped)
func (ws *WindowsService) IsNpxCommand(command string, args []interface{}) bool {
	if strings.HasPrefix(command, "npx ") || command == "npx" {
		return true
	}

	if command == "cmd" && len(args) >= 2 {
		if firstArg, ok := args[0].(string); ok && firstArg == "/c" {
			if secondArg, ok := args[1].(string); ok {
				return strings.HasPrefix(secondArg, "npx ") || secondArg == "npx"
			}
		}
	}

	return false
}

// ShouldWrapForWindows checks if a command should be wrapped for Windows
func (ws *WindowsService) ShouldWrapForWindows(command string, args []interface{}) bool {
	if !ws.IsWindows() {
		return false
	}

	return ws.IsNpxCommand(command, args) && !ws.IsAlreadyWrapped(command, args)
}

// IsAlreadyWrapped checks if a command is already wrapped with cmd /c
func (ws *WindowsService) IsAlreadyWrapped(command string, args []interface{}) bool {
	return command == "cmd" && len(args) >= 2
}

// ApplyWindowsTransformation applies Windows-specific transformations to MCP server configs
func (ws *WindowsService) ApplyWindowsTransformation(servers []models.MCPServer, wrap bool) []models.MCPServer {
	if !ws.IsWindows() {
		return servers
	}

	var result []models.MCPServer
	for _, server := range servers {
		transformedServer := server

		serverArgs := ws.convertToInterfaceSlice(server.Args)
		if wrap && ws.ShouldWrapForWindows(server.Command, serverArgs) {
			// Wrap npx commands for Windows
			newCommand, newArgs := ws.WrapNpxCommand(server.Command, serverArgs)
			transformedServer.Command = newCommand
			transformedServer.Args = ws.convertToStringSlice(newArgs)
		} else if !wrap && ws.IsAlreadyWrapped(server.Command, serverArgs) {
			// Unwrap npx commands when leaving Windows
			newCommand, newArgs := ws.UnwrapNpxCommand(server.Command, serverArgs)
			transformedServer.Command = newCommand
			transformedServer.Args = ws.convertToStringSlice(newArgs)
		}

		result = append(result, transformedServer)
	}

	return result
}

// Helper methods for type conversions
func (ws *WindowsService) convertToInterfaceSlice(stringSlice []string) []interface{} {
	result := make([]interface{}, len(stringSlice))
	for i, v := range stringSlice {
		result[i] = v
	}
	return result
}

func (ws *WindowsService) convertToStringSlice(interfaceSlice []interface{}) []string {
	result := make([]string, len(interfaceSlice))
	for i, v := range interfaceSlice {
		if str, ok := v.(string); ok {
			result[i] = str
		}
	}
	return result
}

// ConvertToInterfaceSlice converts string slice to interface slice
func ConvertToInterfaceSlice(stringSlice []string) []interface{} {
	result := make([]interface{}, len(stringSlice))
	for i, v := range stringSlice {
		result[i] = v
	}
	return result
}
