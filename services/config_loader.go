package services

import (
	"embed"
	"fmt"
	"os"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"
)

//go:embed agents.yaml
var configFS embed.FS

type AgentDefinition struct {
	ID            string                   `yaml:"id"`
	Name          string                   `yaml:"name"`
	Description   string                   `yaml:"description"`
	Platforms     map[string]PlatformConfig `yaml:"platforms"`
	ConfigKey     string                   `yaml:"config_key"`
	Format        string                   `yaml:"format"`
}

type PlatformConfig struct {
	ConfigPaths []string `yaml:"config_paths"`
}

type TransformRule struct {
	AddFields   map[string]interface{} `yaml:"add_fields"`
	RemoveFields []string              `yaml:"remove_fields"`
	KeepFields  []string               `yaml:"keep_fields"`
}

type AgentsConfig struct {
	Transforms map[string]TransformRule `yaml:"transforms"`
	Agents     []AgentDefinition        `yaml:"agents"`
}

type ConfigLoader struct {
	config *AgentsConfig
}

func NewConfigLoader() (*ConfigLoader, error) {
	// Try to load from embedded file first
	data, err := configFS.ReadFile("agents.yaml")
	if err != nil {
		// Try to load from disk in current directory
		data, err = os.ReadFile("agents.yaml")
		if err != nil {
			return nil, fmt.Errorf("failed to load agents.yaml: %w", err)
		}
	}

	var config AgentsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse agents.yaml: %w", err)
	}

	return &ConfigLoader{config: &config}, nil
}

func (cl *ConfigLoader) GetAgentDefinitions() []AgentDefinition {
	return cl.config.Agents
}

func (cl *ConfigLoader) GetAgentDefinition(agentID string) *AgentDefinition {
	for _, agent := range cl.config.Agents {
		if agent.ID == agentID {
			return &agent
		}
	}
	return nil
}

// ExpandPath expands paths like ~, $APPDATA, $ProgramData
func (cl *ConfigLoader) ExpandPath(path string) string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = os.Getenv("USERPROFILE")
	}

	// Replace ~ with home directory
	path = strings.ReplaceAll(path, "~", homeDir)

	// Replace $APPDATA
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		path = strings.ReplaceAll(path, "$APPDATA", appData)

		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = "C:\\ProgramData"
		}
		path = strings.ReplaceAll(path, "$ProgramData", programData)
	}

	return path
}

// GetConfigPathsForAgent returns all possible config paths for an agent on current platform
func (cl *ConfigLoader) GetConfigPathsForAgent(agentID string) []string {
	agent := cl.GetAgentDefinition(agentID)
	if agent == nil {
		return nil
	}

	goos := runtime.GOOS
	platformConfig, exists := agent.Platforms[goos]
	if !exists {
		return nil
	}

	var expandedPaths []string
	for _, path := range platformConfig.ConfigPaths {
		expandedPath := cl.ExpandPath(path)
		expandedPaths = append(expandedPaths, expandedPath)
	}

	return expandedPaths
}

// GetExistingConfigPaths returns only the paths that actually exist
func (cl *ConfigLoader) GetExistingConfigPaths(agentID string) []string {
	paths := cl.GetConfigPathsForAgent(agentID)
	var existing []string

	for _, path := range paths {
		if fileExists(path) {
			existing = append(existing, path)
		}
	}

	return existing
}

// GetConfigKey returns the key used for MCP servers in the config file
func (cl *ConfigLoader) GetConfigKey(agentID string) string {
	agent := cl.GetAgentDefinition(agentID)
	if agent == nil {
		return "mcpServers"
	}
	return agent.ConfigKey
}

// GetFormat returns the format type for the agent
func (cl *ConfigLoader) GetFormat(agentID string) string {
	agent := cl.GetAgentDefinition(agentID)
	if agent == nil {
		return "standard"
	}
	return agent.Format
}

// GetFirstExistingPath returns the first path that exists
func (cl *ConfigLoader) GetFirstExistingPath(agentID string) (string, error) {
	paths := cl.GetConfigPathsForAgent(agentID)
	if len(paths) == 0 {
		return "", fmt.Errorf("no config paths defined for agent: %s", agentID)
	}

	for _, path := range paths {
		if fileExists(path) {
			return path, nil
		}
	}

	// Return the first path even if it doesn't exist (for creating new config)
	return paths[0], nil
}

// GetTransformRule returns the transform rule for converting between two formats
func (cl *ConfigLoader) GetTransformRule(fromFormat, toFormat string) *TransformRule {
	key := fromFormat + "_to_" + toFormat
	rule, exists := cl.config.Transforms[key]
	if !exists {
		return nil
	}
	return &rule
}

// ApplyTransformRule applies a transformation rule to the server data
func (cl *ConfigLoader) ApplyTransformRule(data interface{}, rule *TransformRule) interface{} {
	if rule == nil {
		return data
	}

	servers, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})

	for name, config := range servers {
		configMap, ok := config.(map[string]interface{})
		if !ok {
			continue
		}

		newConfig := make(map[string]interface{})

		// Add new fields from rule
		for key, value := range rule.AddFields {
			newConfig[key] = value
		}

		// Keep specified fields
		if len(rule.KeepFields) > 0 {
			for _, field := range rule.KeepFields {
				if value, exists := configMap[field]; exists {
					newConfig[field] = value
				}
			}
		} else {
			// If no keep_fields specified, copy all fields except removed ones
			removeSet := make(map[string]bool)
			for _, field := range rule.RemoveFields {
				removeSet[field] = true
			}
			for key, value := range configMap {
				if !removeSet[key] {
					newConfig[key] = value
				}
			}
		}

		result[name] = newConfig
	}

	return result
}
