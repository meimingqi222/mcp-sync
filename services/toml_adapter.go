package services

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// CodexConfig represents the structure of Codex's config.toml file
type CodexConfig struct {
	// We use a map to store the entire config to preserve all fields
	RawData map[string]interface{}
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
	var rawData map[string]interface{}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := toml.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return &CodexConfig{RawData: rawData}, nil
}

// WriteCodexConfig writes Codex config back to TOML file
func (ta *TOMLAdapter) WriteCodexConfig(filePath string, config *CodexConfig) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	// Map keys are not sorted by default in Go map iteration, but TOML encoder usually sorts them.
	return encoder.Encode(config.RawData)
}

// CodexToStandard converts Codex TOML MCP servers to standard JSON format
func (ta *TOMLAdapter) CodexToStandard(codexServers map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for name, serverData := range codexServers {
		serverMap, ok := serverData.(map[string]interface{})
		if !ok {
			continue
		}

		serverConfig := make(map[string]interface{})
		if cmd, ok := serverMap["command"].(string); ok {
			serverConfig["command"] = cmd
		}

		if args, ok := serverMap["args"].([]interface{}); ok {
			serverConfig["args"] = args
		}

		if env, ok := serverMap["env"].(map[string]interface{}); ok {
			serverConfig["env"] = env
		}

		if cwd, ok := serverMap["cwd"].(string); ok {
			serverConfig["cwd"] = cwd
		}

		result[name] = serverConfig
	}

	return result
}

// StandardToCodex converts standard JSON MCP servers to Codex TOML format
// Note: Codex only supports stdio transport. HTTP/SSE servers will be skipped.
func (ta *TOMLAdapter) StandardToCodex(standardServers map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

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

		server := make(map[string]interface{})

		if cmd, ok := serverMap["command"].(string); ok {
			server["command"] = cmd
		}

		if args, ok := serverMap["args"].([]interface{}); ok {
			server["args"] = args
		} else if args, ok := serverMap["args"].([]string); ok {
			// Convert []string to []interface{}
			ifaceArgs := make([]interface{}, len(args))
			for i, v := range args {
				ifaceArgs[i] = v
			}
			server["args"] = ifaceArgs
		}

		if env, ok := serverMap["env"].(map[string]interface{}); ok {
			server["env"] = env
		} else if env, ok := serverMap["env"].(map[string]string); ok {
			// Convert map[string]string to map[string]interface{}
			ifaceEnv := make(map[string]interface{})
			for k, v := range env {
				ifaceEnv[k] = v
			}
			server["env"] = ifaceEnv
		}

		if cwd, ok := serverMap["cwd"].(string); ok {
			server["cwd"] = cwd
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

	mcpServers, ok := config.RawData["mcp_servers"].(map[string]interface{})
	if !ok || mcpServers == nil {
		return make(map[string]interface{}), nil
	}

	// Debug: print what we read from TOML
	println(fmt.Sprintf("[TOML Debug] Read %d MCP servers from Codex config", len(mcpServers)))

	result := ta.CodexToStandard(mcpServers)

	// Debug: print what we're returning
	println(fmt.Sprintf("[TOML Debug] Converted to %d standard servers", len(result)))

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
			RawData: make(map[string]interface{}),
		}
	} else {
		// Just print existing keys count for debugging
		println(fmt.Sprintf("Read existing Codex config with %d top-level keys", len(config.RawData)))
	}

	if config.RawData == nil {
		config.RawData = make(map[string]interface{})
	}

	// Convert and update MCP servers
	originalCount := len(standardServers)
	newServers := ta.StandardToCodex(standardServers)
	convertedCount := len(newServers)
	skippedCount := originalCount - convertedCount

	// Check for changes (optional optimization, but good for logging)
	if skippedCount > 0 {
		println(fmt.Sprintf("[Warning] Skipped %d HTTP/SSE servers (Codex only supports stdio)", skippedCount))
	}
	println(fmt.Sprintf("Updated MCP servers count: %d (original: %d, skipped: %d)",
		convertedCount, originalCount, skippedCount))

	// Update the map directly
	config.RawData["mcp_servers"] = newServers

	// Write back to file
	println("Writing config back to file...")
	return ta.WriteCodexConfig(filePath, config)
}
