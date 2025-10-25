package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mcp-sync/models"
	"os"
	"path/filepath"
)

type ConfigManager struct {
	detector *AgentDetector
}

func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		detector: NewAgentDetector(),
	}
}

type MCPServersConfig struct {
	MCPServers map[string]interface{} `json:"mcpServers"`
}

func (cm *ConfigManager) ReadAgentMCPConfig(agentID string) (models.MCPServer, error) {
	configPath, err := cm.detector.GetAgentConfigPath(agentID)
	if err != nil {
		return models.MCPServer{}, err
	}

	if !fileExists(configPath) {
		return models.MCPServer{}, fmt.Errorf("config file not found: %s", configPath)
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return models.MCPServer{}, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return models.MCPServer{}, err
	}

	server := models.MCPServer{
		ID:   agentID,
		Name: agentID,
	}

	return server, nil
}

func (cm *ConfigManager) WriteAgentMCPConfig(agentID string, servers []models.MCPServer) error {
	configPath, err := cm.detector.GetAgentConfigPath(agentID)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Read existing config or create new
	var config map[string]interface{}

	if fileExists(configPath) {
		data, err := ioutil.ReadFile(configPath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &config); err != nil {
			config = make(map[string]interface{})
		}
	} else {
		config = make(map[string]interface{})
	}

	// Get the agent's config key (could be "mcpServers", "context_servers", etc.)
	configLoader, err := NewConfigLoader()
	if err != nil {
		return err
	}

	// Find agent ID by matching config path
	var detectedAgentID string
	agentDefs := configLoader.GetAgentDefinitions()
	for _, agent := range agentDefs {
		paths := configLoader.GetConfigPathsForAgent(agent.ID)
		for _, path := range paths {
			if path == configPath {
				detectedAgentID = agent.ID
				break
			}
		}
		if detectedAgentID != "" {
			break
		}
	}

	configKey := "mcpServers" // default
	if detectedAgentID != "" {
		configKey = configLoader.GetConfigKey(detectedAgentID)
	}

	// Get existing mcpServers map or create new
	var existingMcpServers map[string]interface{}
	if mcpServers, exists := config[configKey]; exists {
		if mcpServersMap, ok := mcpServers.(map[string]interface{}); ok {
			existingMcpServers = mcpServersMap
		}
	}
	if existingMcpServers == nil {
		existingMcpServers = make(map[string]interface{})
	}

	// Apply Windows transformation if needed
	windowsSvc := NewWindowsService()
	transformedServers := servers
	if windowsSvc.IsWindows() {
		transformedServers = windowsSvc.ApplyWindowsTransformation(servers, true)
	}

	// Update mcpServers - merge with existing but override by name
	for _, server := range transformedServers {
		if !server.Enabled {
			// Remove disabled server
			delete(existingMcpServers, server.Name)
			continue
		}

		serverConfig := map[string]interface{}{
			"command": server.Command,
			"args":    server.Args,
			"env":     server.Env,
		}

		// Preserve other fields that might exist in the original config
		if existingConfig, exists := existingMcpServers[server.Name]; exists {
			if existingConfigMap, ok := existingConfig.(map[string]interface{}); ok {
				for key, value := range existingConfigMap {
					if _, shouldOverride := map[string]bool{"command": true, "args": true, "env": true}[key]; !shouldOverride {
						serverConfig[key] = value
					}
				}
			}
		}

		existingMcpServers[server.Name] = serverConfig
	}

	config[configKey] = existingMcpServers

	// Write back
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, data, 0644)
}

func (cm *ConfigManager) GetAgentMCPConfig(agentID string) (map[string]interface{}, error) {
	configPath, err := cm.detector.GetAgentConfigPath(agentID)
	if err != nil {
		return nil, err
	}

	if !fileExists(configPath) {
		return make(map[string]interface{}), nil
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Get the correct config key for this agent
	configLoader, err := NewConfigLoader()
	if err != nil {
		return nil, err
	}

	configKey := configLoader.GetConfigKey(agentID)

	// Apply Windows unwrapping if needed
	windowsSvc := NewWindowsService()
	if windowsSvc.IsWindows() {
		if serversData, exists := config[configKey]; exists {
			if serverMap, ok := serversData.(map[string]interface{}); ok {
				// Convert to MCPServer slice for unwrapping
				var servers []models.MCPServer
				for serverName, serverConfig := range serverMap {
					server := models.MCPServer{
						ID:   serverName,
						Name: serverName,
					}
					if serverMap, ok := serverConfig.(map[string]interface{}); ok {
						if cmd, ok := serverMap["command"].(string); ok {
							server.Command = cmd
						}
						if args, ok := serverMap["args"].([]interface{}); ok {
							for _, arg := range args {
								if argStr, ok := arg.(string); ok {
									server.Args = append(server.Args, argStr)
								}
							}
						}
						if env, ok := serverMap["env"].(map[string]interface{}); ok {
							server.Env = make(map[string]string)
							for k, v := range env {
								if strVal, ok := v.(string); ok {
									server.Env[k] = strVal
								}
							}
						}
					}
					servers = append(servers, server)
				}

				// Apply Windows transformation (unwrap npx commands)
				servers = windowsSvc.ApplyWindowsTransformation(servers, false)

				// Convert back to config format
				unwrappedServersData := make(map[string]interface{})
				for _, server := range servers {
					serverConfig := make(map[string]interface{})
					serverConfig["command"] = server.Command
					if len(server.Args) > 0 {
						argsInterface := make([]interface{}, len(server.Args))
						for i, arg := range server.Args {
							argsInterface[i] = arg
						}
						serverConfig["args"] = argsInterface
					}
					if len(server.Env) > 0 {
						envInterface := make(map[string]interface{})
						for k, v := range server.Env {
							envInterface[k] = v
						}
						serverConfig["env"] = envInterface
					}
					unwrappedServersData[server.Name] = serverConfig
				}

				config[configKey] = unwrappedServersData
			}
		}
	}

	return config, nil
}

func (cm *ConfigManager) GetAllAgentsConfig() (map[string]interface{}, error) {
	agents, err := cm.detector.DetectInstalledAgents()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})

	for _, agent := range agents {
		if agent.Status != "detected" {
			continue
		}

		config, err := cm.GetAgentMCPConfig(agent.ID)
		if err != nil {
			continue
		}

		result[agent.ID] = config
	}

	return result, nil
}

func (cm *ConfigManager) ExportConfigAsJSON(servers []models.MCPServer) ([]byte, error) {
	exportData := map[string]interface{}{
		"servers": servers,
	}

	return json.MarshalIndent(exportData, "", "  ")
}

func (cm *ConfigManager) ImportConfigFromJSON(data []byte) ([]models.MCPServer, error) {
	var importData struct {
		Servers []models.MCPServer `json:"servers"`
	}

	if err := json.Unmarshal(data, &importData); err != nil {
		return nil, err
	}

	return importData.Servers, nil
}

func (cm *ConfigManager) MergeConfigs(local, remote []models.MCPServer) ([]models.MCPServer, []string, error) {
	// Simple merge strategy: remote overwrites local
	// Returns merged config and list of conflicts

	localMap := make(map[string]models.MCPServer)
	for _, s := range local {
		localMap[s.ID] = s
	}

	remoteMap := make(map[string]models.MCPServer)
	for _, s := range remote {
		remoteMap[s.ID] = s
	}

	var conflicts []string
	result := make([]models.MCPServer, 0)

	// Check for conflicts
	for id, remoteServer := range remoteMap {
		if localServer, exists := localMap[id]; exists {
			// Check if configurations differ
			if !configEqual(localServer, remoteServer) {
				conflicts = append(conflicts, id)
			}
		}
		result = append(result, remoteServer)
	}

	// Add local servers not in remote
	for id, localServer := range localMap {
		if _, exists := remoteMap[id]; !exists {
			result = append(result, localServer)
		}
	}

	return result, conflicts, nil
}

func configEqual(a, b models.MCPServer) bool {
	if a.Name != b.Name || a.Command != b.Command {
		return false
	}
	if len(a.Args) != len(b.Args) {
		return false
	}
	for i, arg := range a.Args {
		if arg != b.Args[i] {
			return false
		}
	}
	return true
}
