package services

import (
	"fmt"
	"mcp-sync/models"
	"os"
)

type AgentDetector struct {
	configLoader *ConfigLoader
}

func NewAgentDetector() *AgentDetector {
	loader, err := NewConfigLoader()
	if err != nil {
		println(fmt.Sprintf("Warning: failed to load agent config: %v", err))
		loader, _ = NewConfigLoader()
	}
	return &AgentDetector{
		configLoader: loader,
	}
}

func (ad *AgentDetector) DetectInstalledAgents() ([]models.Agent, error) {
	var agents []models.Agent
	agentDefs := ad.configLoader.GetAgentDefinitions()

	for _, agentDef := range agentDefs {
		configPaths := ad.configLoader.GetConfigPathsForAgent(agentDef.ID)
		if len(configPaths) == 0 {
			continue
		}

		existingPaths := ad.configLoader.GetExistingConfigPaths(agentDef.ID)

		status := "detected"
		if len(existingPaths) == 0 {
			status = "not_installed"
		}

		for _, path := range configPaths {
			println(fmt.Sprintf("检查 %s: %s", agentDef.ID, path))
			if fileExists(path) {
				println(fmt.Sprintf("  ✓ 找到: %s", path))
			} else {
				println(fmt.Sprintf("  ✗ 未找到: %s", path))
			}
		}

		agent := models.Agent{
			ID:            agentDef.ID,
			Name:          agentDef.Name,
			Platform:      os.Getenv("GOOS"),
			Status:        status,
			ConfigPaths:   configPaths,
			ExistingPaths: existingPaths,
			Enabled:       status == "detected",
		}

		agents = append(agents, agent)
		println(fmt.Sprintf("Agent %s: %s", agentDef.ID, status))
	}

	return agents, nil
}

func (ad *AgentDetector) GetAgentConfigPath(agentID string) (string, error) {
	return ad.configLoader.GetFirstExistingPath(agentID)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
