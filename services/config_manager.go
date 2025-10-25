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
		ID: agentID,
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

	// Update mcpServers
	mcpServersMap := make(map[string]interface{})
	for _, server := range servers {
		if !server.Enabled {
			continue
		}
		mcpServersMap[server.Name] = map[string]interface{}{
			"command": server.Command,
			"args":    server.Args,
			"env":     server.Env,
		}
	}

	config["mcpServers"] = mcpServersMap

	// Write back
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, data, 0644)
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

		config, err := cm.ReadAgentMCPConfig(agent.ID)
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
