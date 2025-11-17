package services

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// CodexConfig represents the structure of Codex's config.toml file
type CodexConfig struct {
	ModelProvider         string                       `toml:"model_provider,omitempty"`
	Model                 string                       `toml:"model,omitempty"`
	ModelReasoningEffort  string                       `toml:"model_reasoning_effort,omitempty"`
	DisableResponseStorage bool                        `toml:"disable_response_storage,omitempty"`
	MCPServers            map[string]CodexMCPServer    `toml:"mcp_servers,omitempty"`
	ModelProviders        map[string]interface{}       `toml:"model_providers,omitempty"`
}

// CodexMCPServer represents a single MCP server configuration in Codex TOML format
type CodexMCPServer struct {
	Command string            `toml:"command"`
	Args    []string          `toml:"args,omitempty"`
	Env     map[string]string `toml:"env,omitempty"`
	CWD     string            `toml:"cwd,omitempty"`
}

// TOMLAdapter handles conversion between Codex TOML format and standard JSON format
type TOMLAdapter struct{}

// NewTOMLAdapter creates a new TOML adapter
func NewTOMLAdapter() *TOMLAdapter {
	return &TOMLAdapter{}
}

// ReadCodexConfig reads and parses Codex's config.toml file
func (ta *TOMLAdapter) ReadCodexConfig(filePath string) (*CodexConfig, error) {
	var config CodexConfig
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return &config, nil
}

// WriteCodexConfig writes Codex config back to TOML file
func (ta *TOMLAdapter) WriteCodexConfig(filePath string, config *CodexConfig) error {
	// Manually build TOML to ensure inline table format for env
	var content strings.Builder

	// Write global config
	if config.ModelProvider != "" {
		content.WriteString(fmt.Sprintf("model_provider = %q\n", config.ModelProvider))
	}
	if config.Model != "" {
		content.WriteString(fmt.Sprintf("model = %q\n", config.Model))
	}
	if config.ModelReasoningEffort != "" {
		content.WriteString(fmt.Sprintf("model_reasoning_effort = %q\n", config.ModelReasoningEffort))
	}
	if config.DisableResponseStorage {
		content.WriteString(fmt.Sprintf("disable_response_storage = %t\n", config.DisableResponseStorage))
	}

	// Write model_providers if exists (preserving complex nested structure)
	if len(config.ModelProviders) > 0 {
		content.WriteString("\n")
		for providerName, providerData := range config.ModelProviders {
			content.WriteString(fmt.Sprintf("[model_providers.%s]\n", providerName))
			if providerMap, ok := providerData.(map[string]interface{}); ok {
				for key, value := range providerMap {
					switch v := value.(type) {
					case string:
						content.WriteString(fmt.Sprintf("  %s = %q\n", key, v))
					case bool:
						content.WriteString(fmt.Sprintf("  %s = %t\n", key, v))
					case int, int64, float64:
						content.WriteString(fmt.Sprintf("  %s = %v\n", key, v))
					default:
						// For complex types, try to format as string
						content.WriteString(fmt.Sprintf("  %s = %q\n", key, fmt.Sprint(v)))
					}
				}
			}
		}
	}

	// Write MCP servers with inline env tables
	if len(config.MCPServers) > 0 {
		content.WriteString("\n")
		for serverName, server := range config.MCPServers {
			content.WriteString(fmt.Sprintf("[mcp_servers.%s]\n", serverName))
			content.WriteString(fmt.Sprintf("command = %q\n", server.Command))
			
			if len(server.Args) > 0 {
				content.WriteString("args = [")
				for i, arg := range server.Args {
					if i > 0 {
						content.WriteString(", ")
					}
					content.WriteString(fmt.Sprintf("%q", arg))
				}
				content.WriteString("]\n")
			}
			
			// Write env as inline table
			if len(server.Env) > 0 {
				content.WriteString("env = { ")
				i := 0
				for key, value := range server.Env {
					if i > 0 {
						content.WriteString(", ")
					}
					content.WriteString(fmt.Sprintf("%q = %q", key, value))
					i++
				}
				content.WriteString(" }\n")
			}
			
			if server.CWD != "" {
				content.WriteString(fmt.Sprintf("cwd = %q\n", server.CWD))
			}
			
			content.WriteString("\n")
		}
	}

	// Write to file
	return os.WriteFile(filePath, []byte(content.String()), 0644)
}

// CodexToStandard converts Codex TOML MCP servers to standard JSON format
func (ta *TOMLAdapter) CodexToStandard(codexServers map[string]CodexMCPServer) map[string]interface{} {
	result := make(map[string]interface{})

	for name, server := range codexServers {
		serverConfig := make(map[string]interface{})
		serverConfig["command"] = server.Command
		
		if len(server.Args) > 0 {
			serverConfig["args"] = server.Args
		}
		
		if len(server.Env) > 0 {
			serverConfig["env"] = server.Env
		}

		if server.CWD != "" {
			serverConfig["cwd"] = server.CWD
		}

		result[name] = serverConfig
	}

	return result
}

// StandardToCodex converts standard JSON MCP servers to Codex TOML format
// Note: Codex only supports stdio transport. HTTP/SSE servers will be skipped.
func (ta *TOMLAdapter) StandardToCodex(standardServers map[string]interface{}) map[string]CodexMCPServer {
	result := make(map[string]CodexMCPServer)

	for name, serverInterface := range standardServers {
		serverMap, ok := serverInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is an HTTP or SSE server - Codex doesn't support these
		if serverType, hasType := serverMap["type"].(string); hasType {
			if serverType == "http" || serverType == "sse" {
				println(fmt.Sprintf("[TOML] Skipping server '%s': Codex does not support %s transport (only stdio is supported)", name, serverType))
				continue
			}
		}

		server := CodexMCPServer{}

		if cmd, ok := serverMap["command"].(string); ok {
			server.Command = cmd
		}

		if args, ok := serverMap["args"].([]interface{}); ok {
			for _, arg := range args {
				if argStr, ok := arg.(string); ok {
					server.Args = append(server.Args, argStr)
				}
			}
		} else if args, ok := serverMap["args"].([]string); ok {
			server.Args = args
		}

		if env, ok := serverMap["env"].(map[string]interface{}); ok {
			server.Env = make(map[string]string)
			for k, v := range env {
				if strVal, ok := v.(string); ok {
					server.Env[k] = strVal
				}
			}
		} else if env, ok := serverMap["env"].(map[string]string); ok {
			server.Env = env
		}

		if cwd, ok := serverMap["cwd"].(string); ok {
			server.CWD = cwd
		}

		result[name] = server
	}

	return result
}

// GetMCPServersAsStandard reads Codex config and returns MCP servers in standard format
func (ta *TOMLAdapter) GetMCPServersAsStandard(filePath string) (map[string]interface{}, error) {
	config, err := ta.ReadCodexConfig(filePath)
	if err != nil {
		return nil, err
	}

	if config.MCPServers == nil {
		return make(map[string]interface{}), nil
	}

	// Debug: print what we read from TOML
	println(fmt.Sprintf("[TOML Debug] Read %d MCP servers from Codex config", len(config.MCPServers)))
	for name, server := range config.MCPServers {
		println(fmt.Sprintf("  Server '%s': command=%s, args=%d, env=%d",
			name, server.Command, len(server.Args), len(server.Env)))
	}

	result := ta.CodexToStandard(config.MCPServers)
	
	// Debug: print what we're returning
	println(fmt.Sprintf("[TOML Debug] Converted to %d standard servers", len(result)))
	for name, serverInterface := range result {
		if serverMap, ok := serverInterface.(map[string]interface{}); ok {
			println(fmt.Sprintf("  Server '%s': fields=%v", name, len(serverMap)))
			for key := range serverMap {
				println(fmt.Sprintf("    - %s", key))
			}
		}
	}
	
	return result, nil
}

// SetMCPServersFromStandard updates Codex config with MCP servers from standard format
func (ta *TOMLAdapter) SetMCPServersFromStandard(filePath string, standardServers map[string]interface{}) error {
	// Read existing config to preserve other settings
	config, err := ta.ReadCodexConfig(filePath)
	if err != nil {
		// If file doesn't exist or is empty, create new config
		println(fmt.Sprintf("Creating new Codex config (file didn't exist or couldn't be read): %v", err))
		config = &CodexConfig{
			MCPServers: make(map[string]CodexMCPServer),
		}
	} else {
		println(fmt.Sprintf("Read existing Codex config - ModelProvider: %s, Model: %s, MCP Servers: %d",
			config.ModelProvider, config.Model, len(config.MCPServers)))
	}

	// Convert and update MCP servers
	originalCount := len(standardServers)
	config.MCPServers = ta.StandardToCodex(standardServers)
	convertedCount := len(config.MCPServers)
	skippedCount := originalCount - convertedCount
	
	if skippedCount > 0 {
		println(fmt.Sprintf("[Warning] Skipped %d HTTP/SSE servers (Codex only supports stdio)", skippedCount))
	}
	println(fmt.Sprintf("Updated MCP servers count: %d (original: %d, skipped: %d)", 
		convertedCount, originalCount, skippedCount))

	// Write back to file
	println("Writing config back to file...")
	return ta.WriteCodexConfig(filePath, config)
}
