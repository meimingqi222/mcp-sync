package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mcp-sync/models"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AppService struct {
	detector       *AgentDetector
	configManager  *ConfigManager
	configLoader   *ConfigLoader
	gistSync       *GistSyncService
	storage        *StorageService
	securityMgr    *SecurityManager
}

func NewAppService() (*AppService, error) {
	// Initialize storage
	// 优先使用 USERPROFILE (Windows)，其次是 HOME (Unix/Linux/macOS)
	homeDir := os.Getenv("USERPROFILE")
	if homeDir == "" {
		homeDir = os.Getenv("HOME")
	}

	if homeDir == "" {
		return nil, fmt.Errorf("could not determine user home directory")
	}

	dataDir := filepath.Join(homeDir, ".mcp-sync")
	storage, err := NewStorageService(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	configLoader, err := NewConfigLoader()
	if err != nil {
		return nil, err
	}

	// 创建安全管理器（使用 gist ID 作为加密密钥的一部分）
	securityMgr := NewSecurityManager(homeDir)

	return &AppService{
		detector:       NewAgentDetector(),
		configManager:  NewConfigManager(),
		configLoader:   configLoader,
		storage:        storage,
		securityMgr:    securityMgr,
	}, nil
}

func (as *AppService) DetectAgents() ([]models.Agent, error) {
	return as.detector.DetectInstalledAgents()
}

func (as *AppService) InitializeGistSync(token, gistID string) (string, error) {
	// If no gistID provided, create a new gist
	if gistID == "" {
		gs := NewGistSyncService(token, "")
		var err error
		gistID, err = gs.CreateGist([]models.MCPServer{}, "MCP Sync Configuration")
		if err != nil {
			return "", fmt.Errorf("failed to create new gist: %w", err)
		}
		println(fmt.Sprintf("Created new Gist with ID: %s", gistID))
	}
	
	as.gistSync = NewGistSyncService(token, gistID)
	
	// Save sync config to storage
	config, _ := as.storage.LoadSyncConfig()
	config.GitHubToken = token
	config.GistID = gistID
	config.LastUpdateTime = nowTime()
	
	if err := as.storage.SaveSyncConfig(config); err != nil {
		return "", err
	}
	
	return gistID, nil
}

// SetupGistEncryption 配置 Gist 同步的加密
func (as *AppService) SetupGistEncryption(enabled bool, password string) error {
	if as.gistSync == nil {
		return fmt.Errorf("gist sync not initialized")
	}

	// Also save encryption config to storage
	config, _ := as.storage.LoadSyncConfig()
	config.EnableEncryption = enabled
	config.EncryptionVersion = "2.0" // 标记使用新版本加密系统
	config.LastUpdateTime = nowTime()
	
	// 如果提供了密码，说明是从旧版本迁移
	if password != "" {
		config.EncryptionPassword = password // 临时保存用于迁移
	} else {
		config.EncryptionPassword = "" // 旧版本密码字段，新版本不再使用
	}

	if err := as.storage.SaveSyncConfig(config); err != nil {
		return fmt.Errorf("failed to save encryption config: %w", err)
	}

	// Enable local storage encryption
	if enabled {
		as.storage.EnableEncryption(password) // 密码参数为空时使用新系统
	}

	return as.gistSync.SetEncryption(enabled, password)
}

func (as *AppService) ValidateGitHubToken(token string) error {
	gs := NewGistSyncService(token, "")
	return gs.ValidateToken()
}

// PushAllAgentsToGist 推送所有已安装 agents 的完整配置到 Gist（保留完整的原始配置）
func (as *AppService) PushAllAgentsToGist() error {
	// Load sync config to get credentials
	config, err := as.storage.LoadSyncConfig()
	if err != nil {
		return fmt.Errorf("failed to load sync config: %w", err)
	}
	
	if config.GitHubToken == "" || config.GistID == "" {
		return fmt.Errorf("GitHub token or Gist ID not configured")
	}
	
	// Initialize gist sync if not already done
	if as.gistSync == nil {
		as.gistSync = NewGistSyncService(config.GitHubToken, config.GistID)
		
		// Setup encryption if enabled
		if config.EnableEncryption && config.EncryptionPassword != "" {
			as.gistSync.SetEncryption(config.EnableEncryption, config.EncryptionPassword)
		}
	}

	// Collect all agents' COMPLETE configurations (not just servers)
	agents, err := as.detector.DetectInstalledAgents()
	if err != nil {
		return fmt.Errorf("failed to detect agents: %w", err)
	}
	
	// Prepare a map of all agent configs to push
	allAgentConfigs := make(map[string]interface{})
	pushedCount := 0
	
	for _, agent := range agents {
		if agent.Status == "detected" {
			agentConfig, err := as.GetAgentMCPConfig(agent.ID)
			if err != nil {
				println(fmt.Sprintf("Warning: failed to read config from %s: %v", agent.ID, err))
				continue
			}
			
			// Store the COMPLETE config for this agent
			allAgentConfigs[agent.ID] = agentConfig
			pushedCount++
			println(fmt.Sprintf("Collected complete config from agent: %s", agent.ID))
		}
	}

	println(fmt.Sprintf("Pushing complete configurations from %d agents to Gist", pushedCount))

	// Save version before push
	configContent, _ := json.MarshalIndent(allAgentConfigs, "", "  ")
	version := models.ConfigVersion{
		ID:        "local_" + nowStr(),
		Timestamp: nowTime(),
		Content:   string(configContent),
		Source:    "local",
		Note:      fmt.Sprintf("Pushed complete config from %d agents", pushedCount),
	}
	as.storage.SaveConfigVersion(version)

	// Push complete configs to Gist
	if pushErr := as.gistSync.PushAgentConfigsToGist(allAgentConfigs); pushErr != nil {
		as.storage.SaveSyncLog(models.SyncLog{
			ID:        genID(),
			Timestamp: nowTime(),
			Action:    "push",
			Status:    "failed",
			Message:   pushErr.Error(),
		})
		return pushErr
	}

	// Update sync time
	updatedConfig, _ := as.storage.LoadSyncConfig()
	updatedConfig.LastSyncTime = nowTime()
	updatedConfig.LastSyncStatus = "success"
	as.storage.SaveSyncConfig(updatedConfig)

	as.storage.SaveSyncLog(models.SyncLog{
		ID:        genID(),
		Timestamp: nowTime(),
		Action:    "push",
		Status:    "success",
		Message:   fmt.Sprintf("Pushed complete configurations from %d agents to Gist", pushedCount),
	})

	return nil
}

func (as *AppService) PushToGist(servers []models.MCPServer) error {
	// Load sync config to get credentials
	config, err := as.storage.LoadSyncConfig()
	if err != nil {
		return fmt.Errorf("failed to load sync config: %w", err)
	}
	
	if config.GitHubToken == "" || config.GistID == "" {
		return fmt.Errorf("GitHub token or Gist ID not configured")
	}
	
	// Initialize gist sync if not already done
	if as.gistSync == nil {
		as.gistSync = NewGistSyncService(config.GitHubToken, config.GistID)
		
		// Setup encryption if enabled
		if config.EnableEncryption && config.EncryptionPassword != "" {
			as.gistSync.SetEncryption(config.EnableEncryption, config.EncryptionPassword)
		}
	}

	// Save version before push
	configContent, _ := as.configManager.ExportConfigAsJSON(servers)
	version := models.ConfigVersion{
		ID:        "local_" + nowStr(),
		Timestamp: nowTime(),
		Content:   string(configContent),
		Source:    "local",
		Note:      "Pushed to Gist",
	}
	as.storage.SaveConfigVersion(version)

	// Push to Gist
	if pushErr := as.gistSync.PushToGist(servers); pushErr != nil {
		as.storage.SaveSyncLog(models.SyncLog{
			ID:        genID(),
			Timestamp: nowTime(),
			Action:    "push",
			Status:    "failed",
			Message:   pushErr.Error(),
		})
		return pushErr
	}

	// Update sync time
	updatedConfig, _ := as.storage.LoadSyncConfig()
	updatedConfig.LastSyncTime = nowTime()
	updatedConfig.LastSyncStatus = "success"
	as.storage.SaveSyncConfig(updatedConfig)

	as.storage.SaveSyncLog(models.SyncLog{
		ID:        genID(),
		Timestamp: nowTime(),
		Action:    "push",
		Status:    "success",
		Message:   "Configuration pushed to Gist",
	})

	return nil
}

func (as *AppService) PullFromGist() ([]models.MCPServer, error) {
	// Load sync config to get credentials
	config, err := as.storage.LoadSyncConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load sync config: %w", err)
	}
	
	if config.GitHubToken == "" || config.GistID == "" {
		return nil, fmt.Errorf("GitHub token or Gist ID not configured")
	}
	
	// Initialize gist sync if not already done
	if as.gistSync == nil {
		as.gistSync = NewGistSyncService(config.GitHubToken, config.GistID)
		
		// Setup encryption if enabled
		if config.EnableEncryption && config.EncryptionPassword != "" {
			as.gistSync.SetEncryption(config.EnableEncryption, config.EncryptionPassword)
		}
	}

	// Pull complete agent configs from Gist
	agentConfigs, err := as.gistSync.PullAgentConfigsFromGist()
	if err != nil {
		as.storage.SaveSyncLog(models.SyncLog{
			ID:        genID(),
			Timestamp: nowTime(),
			Action:    "pull",
			Status:    "failed",
			Message:   err.Error(),
		})
		return nil, err
	}

	// Save version
	configContent, _ := json.MarshalIndent(agentConfigs, "", "  ")
	version := models.ConfigVersion{
		ID:        "remote_" + nowStr(),
		Timestamp: nowTime(),
		Content:   string(configContent),
		Source:    "gist",
		Note:      "Pulled complete configs from Gist",
	}
	as.storage.SaveConfigVersion(version)

	// Apply downloaded complete configurations to each agent
	appliedCount := 0
	if len(agentConfigs) > 0 {
		for agentID, agentConfig := range agentConfigs {
			// Apply the complete config to this specific agent
			err := as.SaveAgentMCPConfig(agentID, agentConfig.(map[string]interface{}))
			if err == nil {
				appliedCount++
				println(fmt.Sprintf("Applied complete configuration to agent: %s", agentID))
			} else {
				println(fmt.Sprintf("Warning: failed to apply config to %s: %v", agentID, err))
			}
		}
	}
	println(fmt.Sprintf("Applied complete configurations to %d agents", appliedCount))

	// Update sync time
	updatedConfig, _ := as.storage.LoadSyncConfig()
	updatedConfig.LastSyncTime = nowTime()
	updatedConfig.LastSyncStatus = "success"
	as.storage.SaveSyncConfig(updatedConfig)

	as.storage.SaveSyncLog(models.SyncLog{
		ID:        genID(),
		Timestamp: nowTime(),
		Action:    "pull",
		Status:    "success",
		Message:   fmt.Sprintf("Complete configurations pulled from Gist and applied to %d agents", appliedCount),
	})

	// Convert back to servers list for compatibility
	servers := []models.MCPServer{}
	for _, config := range agentConfigs {
		if configMap, ok := config.(map[string]interface{}); ok {
			// Try to extract servers from any config key
			for _, serversData := range configMap {
				if serverMap, ok := serversData.(map[string]interface{}); ok {
					for serverName, serverConfig := range serverMap {
						server := models.MCPServer{
							ID:   serverName,
							Name: serverName,
						}
						if serverMap, ok := serverConfig.(map[string]interface{}); ok {
							if cmd, ok := serverMap["command"].(string); ok {
								server.Command = cmd
							}
						}
						servers = append(servers, server)
					}
				}
			}
		}
	}

	return servers, nil
}

func (as *AppService) ApplyConfigToAgents(agentID string, servers []models.MCPServer) error {
	return as.configManager.WriteAgentMCPConfig(agentID, servers)
}

func (as *AppService) ApplyConfigToAllAgents(servers []models.MCPServer) error {
	agents, err := as.detector.DetectInstalledAgents()
	if err != nil {
		return err
	}

	for _, agent := range agents {
		if agent.Status == "detected" {
			as.configManager.WriteAgentMCPConfig(agent.ID, servers)
		}
	}

	return nil
}

func (as *AppService) GetSyncConfig() (models.SyncConfig, error) {
	return as.storage.LoadSyncConfig()
}

func (as *AppService) SaveSyncConfig(config models.SyncConfig) error {
	return as.storage.SaveSyncConfig(config)
}

func (as *AppService) GetConfigVersions(limit int) ([]models.ConfigVersion, error) {
	return as.storage.ListConfigVersions(limit)
}

func (as *AppService) GetSyncLogs(limit int) ([]models.SyncLog, error) {
	return as.storage.GetSyncLogs(limit)
}

func (as *AppService) GetAgentMCPConfig(agentID string) (map[string]interface{}, error) {
	configPath, err := as.detector.GetAgentConfigPath(agentID)
	if err != nil {
		return nil, err
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Remove comments from JSON if present (Zed and other editors support JSON with comments)
	dataStr := string(data)
	lines := strings.Split(dataStr, "\n")
	var cleanedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//") {
			cleanedLines = append(cleanedLines, line)
		}
	}
	cleanedData := []byte(strings.Join(cleanedLines, "\n"))

	var config map[string]interface{}
	if err := json.Unmarshal(cleanedData, &config); err != nil {
		return nil, err
	}

	// Get key name from config loader based on agent definition
	keyName := as.configLoader.GetConfigKey(agentID)

	// Extract only the MCP servers section
	mcpServers, ok := config[keyName]
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	return map[string]interface{}{
		keyName: mcpServers,
	}, nil
}

func (as *AppService) SaveAgentMCPConfig(agentID string, mcpServersConfig map[string]interface{}) error {
	configPath, err := as.detector.GetAgentConfigPath(agentID)
	if err != nil {
		return err
	}

	// Read the full config file first
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Remove comments if present
	dataStr := string(data)
	lines := strings.Split(dataStr, "\n")
	var cleanedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//") {
			cleanedLines = append(cleanedLines, line)
		}
	}
	cleanedData := []byte(strings.Join(cleanedLines, "\n"))

	var fullConfig map[string]interface{}
	if err := json.Unmarshal(cleanedData, &fullConfig); err != nil {
		return err
	}

	// Get config key from agent definition
	targetKeyName := as.configLoader.GetConfigKey(agentID)
	sourceFormat := as.configLoader.GetFormat(agentID)

	// Determine source key name from input
	sourceKeyName := ""
	if _, hasZedKey := mcpServersConfig["context_servers"]; hasZedKey {
		sourceKeyName = "context_servers"
	} else if _, hasStdKey := mcpServersConfig["mcpServers"]; hasStdKey {
		sourceKeyName = "mcpServers"
	}

	// Get the servers data
	var serversData interface{}
	if sourceKeyName != "" {
		serversData = mcpServersConfig[sourceKeyName]
	}

	// Transform format if needed
	if sourceKeyName != targetKeyName && sourceKeyName != "" {
		// Need to convert between formats
		if sourceKeyName == "context_servers" && targetKeyName == "mcpServers" {
			serversData = convertZedToStandard(serversData)
		} else if sourceKeyName == "mcpServers" && targetKeyName == "context_servers" {
			serversData = convertStandardToZed(serversData)
		}
	}

	// For Zed format, add required fields
	if sourceFormat == "zed" && targetKeyName == "context_servers" {
		serversData = convertStandardToZed(serversData)
	}

	// Update the config with target format
	if serversData != nil {
		fullConfig[targetKeyName] = serversData
	}

	// Write back the full config
	updatedData, err := json.MarshalIndent(fullConfig, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return err
	}

	return nil
}

// convertZedToStandard converts Zed context_servers format to standard mcpServers format
func convertZedToStandard(data interface{}) interface{} {
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

		// Extract relevant fields and convert to standard format
		newConfig := make(map[string]interface{})
		if cmd, ok := configMap["command"]; ok {
			newConfig["command"] = cmd
		}
		if args, ok := configMap["args"]; ok {
			newConfig["args"] = args
		}
		if env, ok := configMap["env"]; ok {
			newConfig["env"] = env
		}

		result[name] = newConfig
	}

	return result
}

// convertStandardToZed converts standard mcpServers format to Zed context_servers format
func convertStandardToZed(data interface{}) interface{} {
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

		// Convert to Zed format
		newConfig := make(map[string]interface{})
		newConfig["source"] = "custom"
		newConfig["enabled"] = true

		if cmd, ok := configMap["command"]; ok {
			newConfig["command"] = cmd
		}
		if args, ok := configMap["args"]; ok {
			newConfig["args"] = args
		}
		if env, ok := configMap["env"]; ok {
			newConfig["env"] = env
		}

		result[name] = newConfig
	}

	return result
}

// SyncConfigBetweenAgents syncs configuration from source agent to target agent, automatically handling format conversion
func (as *AppService) SyncConfigBetweenAgents(sourceAgentID, targetAgentID string) error {
	// Read config from source agent
	sourceConfig, err := as.GetAgentMCPConfig(sourceAgentID)
	if err != nil {
		return fmt.Errorf("failed to read source agent config: %w", err)
	}

	// Get source and target agent definitions
	sourceKey := as.configLoader.GetConfigKey(sourceAgentID)
	sourceFormat := as.configLoader.GetFormat(sourceAgentID)
	targetKey := as.configLoader.GetConfigKey(targetAgentID)
	targetFormat := as.configLoader.GetFormat(targetAgentID)

	// Extract servers data from source
	serversData, ok := sourceConfig[sourceKey]
	if !ok {
		serversData = make(map[string]interface{})
	}

	println(fmt.Sprintf("同步配置: %s (%s/%s) -> %s (%s/%s)", 
		sourceAgentID, sourceKey, sourceFormat,
		targetAgentID, targetKey, targetFormat))

	// Convert format if needed
	if sourceFormat != targetFormat {
		println(fmt.Sprintf("  转换格式: %s -> %s", sourceFormat, targetFormat))
		
		// Try to use the configuration-based transform rule first
		transformRule := as.configLoader.GetTransformRule(sourceFormat, targetFormat)
		if transformRule != nil {
			serversData = as.configLoader.ApplyTransformRule(serversData, transformRule)
			println(fmt.Sprintf("  使用配置规则进行转换"))
		} else {
			// Fall back to hardcoded conversions
			println(fmt.Sprintf("  未找到配置规则，使用内置转换"))
			if sourceFormat == "standard" && targetFormat == "zed" {
				serversData = convertStandardToZed(serversData)
			} else if sourceFormat == "zed" && targetFormat == "standard" {
				serversData = convertZedToStandard(serversData)
			}
		}
	}

	// Save to target agent with appropriate key name
	targetConfig := map[string]interface{}{
		targetKey: serversData,
	}

	return as.SaveAgentMCPConfig(targetAgentID, targetConfig)
}

// GetGistSecurityWarnings 获取 Gist 同步的安全警告
func (as *AppService) GetGistSecurityWarnings() []map[string]string {
	return []map[string]string{
		{
			"level":       "warning",
			"title":       "Encryption Required",
			"description": "All configurations are encrypted with your password before syncing to Gist.",
			"suggestion":  "Use a strong password with at least 8 characters including uppercase, lowercase, numbers and special characters.",
		},
		{
			"level":       "warning",
			"title":       "Password Safety",
			"description": "Your encryption password is critical for security. Never share it with anyone.",
			"suggestion":  "Store your password securely and never commit it to version control.",
		},
	}
}

func nowTime() time.Time {
	return time.Now().UTC()
}

func nowStr() string {
	return time.Now().UTC().Format("20060102150405")
}

func genID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// computeHash 计算内容的 SHA256 hash
func computeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// getLatestLocalVersion 获取最新的本地配置版本
func (as *AppService) getLatestLocalVersion() (*models.ConfigVersion, error) {
	versions, err := as.storage.ListConfigVersions(1)
	if err != nil || len(versions) == 0 {
		return nil, err
	}
	
	// 计算最新版本的 hash
	versions[0].Hash = computeHash(versions[0].Content)
	return &versions[0], nil
}

// DetectPushConflict 检测推送冲突 - 比较本地和云端版本
func (as *AppService) DetectPushConflict() (*models.SyncConflict, error) {
	// Load sync config
	config, err := as.storage.LoadSyncConfig()
	if err != nil {
		return nil, err
	}
	
	if config.GitHubToken == "" || config.GistID == "" {
		return nil, fmt.Errorf("GitHub token or Gist ID not configured")
	}
	
	// Initialize gist sync if needed
	if as.gistSync == nil {
		as.gistSync = NewGistSyncService(config.GitHubToken, config.GistID)
		if config.EnableEncryption && config.EncryptionPassword != "" {
			as.gistSync.SetEncryption(config.EnableEncryption, config.EncryptionPassword)
		}
	}
	
	// Get local version
	localVersion, err := as.getLatestLocalVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get local version: %w", err)
	}
	
	if localVersion == nil {
		return &models.SyncConflict{HasConflict: false}, nil
	}
	
	// Get remote version from Gist
	remoteVersion, err := as.gistSync.GetLatestVersion()
	if err != nil {
		// If Gist is empty, no conflict
		return &models.SyncConflict{HasConflict: false}, nil
	}
	
	// Compare hashes
	if remoteVersion != nil && localVersion.Hash != remoteVersion.Hash {
		return &models.SyncConflict{
			HasConflict:   true,
			ConflictType:  "push_conflict",
			LocalVersion:  localVersion,
			RemoteVersion: remoteVersion,
			Message:       "Local configuration differs from cloud version. Choose to keep local, use remote, or merge.",
		}, nil
	}
	
	return &models.SyncConflict{HasConflict: false}, nil
}

// DetectPullConflict 检测拉取冲突 - 检查本地是否有未推送的改动
func (as *AppService) DetectPullConflict() (*models.SyncConflict, error) {
	// Load sync config
	config, err := as.storage.LoadSyncConfig()
	if err != nil {
		return nil, err
	}
	
	if config.GitHubToken == "" || config.GistID == "" {
		return nil, fmt.Errorf("GitHub token or Gist ID not configured")
	}
	
	// Initialize gist sync if needed
	if as.gistSync == nil {
		as.gistSync = NewGistSyncService(config.GitHubToken, config.GistID)
		if config.EnableEncryption && config.EncryptionPassword != "" {
			as.gistSync.SetEncryption(config.EnableEncryption, config.EncryptionPassword)
		}
	}
	
	// Get local version
	localVersion, err := as.getLatestLocalVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get local version: %w", err)
	}
	
	// Get remote version
	remoteVersion, err := as.gistSync.GetLatestVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote version: %w", err)
	}
	
	if remoteVersion == nil {
		return &models.SyncConflict{HasConflict: false}, nil
	}
	
	// Compare hashes - if local is newer than remote, there's unsaved local changes
	if localVersion != nil && localVersion.Timestamp.After(remoteVersion.Timestamp) && localVersion.Hash != remoteVersion.Hash {
		return &models.SyncConflict{
			HasConflict:   true,
			ConflictType:  "pull_conflict",
			LocalVersion:  localVersion,
			RemoteVersion: remoteVersion,
			Message:       "You have local changes not yet pushed to cloud. Choose to keep local, use remote, or merge.",
		}, nil
	}
	
	return &models.SyncConflict{HasConflict: false}, nil
}

// ResolveConflict 解决冲突 - 根据用户选择
func (as *AppService) ResolveConflict(conflictType string, resolution string) error {
	// resolution: "keep_local", "use_remote", "merge"
	
	switch resolution {
	case "keep_local":
		// Just push local to remote
		return as.PushAllAgentsToGist()
	
	case "use_remote":
		// Just pull remote to local
		_, err := as.PullFromGist()
		return err
	
	case "merge":
		// TODO: Implement smart merge logic
		// For now, just use remote
		_, err := as.PullFromGist()
		return err
	
	default:
		return fmt.Errorf("unknown resolution type: %s", resolution)
	}
}
