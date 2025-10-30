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
	ID          string                    `yaml:"id"`
	Name        string                    `yaml:"name"`
	Description string                    `yaml:"description"`
	Platforms   map[string]PlatformConfig `yaml:"platforms"`
	ConfigKey   string                    `yaml:"config_key"`
	Format      string                    `yaml:"format"`
}

type PlatformConfig struct {
	ConfigPaths []string `yaml:"config_paths"`
}

type TransformRule struct {
	AddFields         map[string]interface{} `yaml:"add_fields"`
	RemoveFields      []string               `yaml:"remove_fields"`
	KeepFields        []string               `yaml:"keep_fields"`
	WrapNpxCommands   bool                   `yaml:"wrap_npx_commands"`
	UnwrapNpxCommands bool                   `yaml:"unwrap_npx_commands"`
}

type AgentsConfig struct {
	Transforms map[string]TransformRule `yaml:"transforms"`
	Agents     []AgentDefinition        `yaml:"agents"`
}

type ConfigLoader struct {
	config *AgentsConfig
}

func NewConfigLoader() (*ConfigLoader, error) {
	// Try to load from disk first (for development)
	data, err := os.ReadFile("services/agents.yaml")
	if err != nil {
		// Try current directory
		data, err = os.ReadFile("agents.yaml")
		if err != nil {
			// Fall back to embedded file
			data, err = configFS.ReadFile("agents.yaml")
			if err != nil {
				return nil, fmt.Errorf("failed to load agents.yaml: %w", err)
			}
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

		// Handle npx command wrapping/unwrapping
		if rule.WrapNpxCommands || rule.UnwrapNpxCommands {
			if command, exists := configMap["command"].(string); exists {
				if rule.WrapNpxCommands && runtime.GOOS == "windows" {
					// Wrap npx commands with cmd /c on Windows
					if strings.HasPrefix(command, "npx ") || command == "npx" {
						newConfig["command"] = "cmd"
						if strings.HasPrefix(command, "npx ") {
							newConfig["args"] = []string{"/c", command}
						} else {
							// Handle case where args are separate
							if args, ok := configMap["args"].([]interface{}); ok {
								newArgs := []string{"/c", "npx"}
								for _, arg := range args {
									if argStr, ok := arg.(string); ok {
										newArgs = append(newArgs, argStr)
									}
								}
								newConfig["args"] = newArgs
							} else {
								newConfig["args"] = []string{"/c", "npx"}
							}
						}
					} else {
						// Keep original command for non-npx commands
						newConfig["command"] = command
						if args, exists := configMap["args"]; exists {
							newConfig["args"] = args
						}
					}
				} else if rule.UnwrapNpxCommands {
					// Unwrap cmd /c from npx commands
					if command == "cmd" {
						if args, ok := configMap["args"].([]interface{}); ok && len(args) >= 2 {
							if firstArg, ok := args[0].(string); ok && firstArg == "/c" {
								if secondArg, ok := args[1].(string); ok && (strings.HasPrefix(secondArg, "npx ") || secondArg == "npx") {
									if strings.HasPrefix(secondArg, "npx ") {
										// npx with arguments combined
										newConfig["command"] = secondArg
										if len(args) > 2 {
											// Extract additional arguments
											var remainingArgs []interface{}
											for i := 2; i < len(args); i++ {
												remainingArgs = append(remainingArgs, args[i])
											}
											newConfig["args"] = remainingArgs
										}
									} else if secondArg == "npx" {
										// npx as command with separate args
										if len(args) > 2 {
											var remainingArgs []string
											for i := 2; i < len(args); i++ {
												if argStr, ok := args[i].(string); ok {
													remainingArgs = append(remainingArgs, argStr)
												}
											}
											newConfig["command"] = "npx " + strings.Join(remainingArgs, " ")
										} else {
											newConfig["command"] = "npx"
										}
									}
								} else {
									// Not an npx command, keep original
									newConfig["command"] = command
									newConfig["args"] = args
								}
							} else {
								// Not a /c command, keep original
								newConfig["command"] = command
								newConfig["args"] = args
							}
						} else {
							// Not enough args, keep original
							newConfig["command"] = command
							if args, exists := configMap["args"]; exists {
								newConfig["args"] = args
							}
						}
					} else {
						// Not a cmd command, keep original
						newConfig["command"] = command
						if args, exists := configMap["args"]; exists {
							newConfig["args"] = args
						}
					}
				} else {
					// Keep original command if no wrapping/unwrapping needed
					newConfig["command"] = command
					if args, exists := configMap["args"]; exists {
						newConfig["args"] = args
					}
				}
			}
		}

		// Add new fields from rule
		for key, value := range rule.AddFields {
			if _, exists := newConfig[key]; !exists {
				newConfig[key] = value
			}
		}

		// Keep specified fields (only if not already handled by npx logic)
		if len(rule.KeepFields) > 0 && !(rule.WrapNpxCommands || rule.UnwrapNpxCommands) {
			for _, field := range rule.KeepFields {
				if value, exists := configMap[field]; exists {
					if _, exists := newConfig[field]; !exists {
						newConfig[field] = value
					}
				}
			}
		} else if !(rule.WrapNpxCommands || rule.UnwrapNpxCommands) {
			// If no keep_fields specified and no npx handling, copy all fields except removed ones
			removeSet := make(map[string]bool)
			for _, field := range rule.RemoveFields {
				removeSet[field] = true
			}
			for key, value := range configMap {
				if !removeSet[key] {
					if _, exists := newConfig[key]; !exists {
						newConfig[key] = value
					}
				}
			}
		}

		// Copy env and other fields that weren't handled
		for key, value := range configMap {
			if key != "command" && key != "args" {
				if _, exists := newConfig[key]; !exists {
					newConfig[key] = value
				}
			}
		}

		result[name] = newConfig
	}

	return result
}
