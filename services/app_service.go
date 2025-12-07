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
	detector      *AgentDetector
	configManager *ConfigManager
	configLoader  *ConfigLoader
	gistSync      *GistSyncService
	storage       *StorageService
	securityMgr   *SecurityManager
	windowsSvc    *WindowsService
	converter     *ConfigConverter
	tomlAdapter   *TOMLAdapter
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

	// Inject legacy security manager into storage to handle migration
	storage.securityMgr = securityMgr
	storage.oldEnabled = true

	converter := NewConfigConverter(configLoader)
	tomlAdapter := NewTOMLAdapter()

	return &AppService{
		detector:      NewAgentDetector(),
		configManager: NewConfigManager(),
		configLoader:  configLoader,
		storage:       storage,
		securityMgr:   securityMgr,
		windowsSvc:    NewWindowsService(),
		converter:     converter,
		tomlAdapter:   tomlAdapter,
	}, nil
}

func (as *AppService) DetectAgents() ([]models.Agent, error) {
	return as.detector.DetectInstalledAgents()
}

func (as *AppService) InitializeGistSync(token, gistID string) (string, error) {
	// If gistID provided, validate it exists
	if gistID != "" {
		tempGs := NewGistSyncService(token, gistID)
		_, err := tempGs.GetLatestVersion()
		if err != nil {
			if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
				println(fmt.Sprintf("Warning: Provided Gist ID %s not found (404). Will create a new one.", gistID))
				gistID = ""
			} else {
				// Other error (e.g. auth failed), return it
				return "", fmt.Errorf("failed to validate gist: %w", err)
			}
		}
	}

	// If no gistID provided (or invalid/not found), create a new gist
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

	// 保存 Gist 加密密码到新字段
	if password != "" {
		config.GistEncryptionPassword = password
	} else {
		config.GistEncryptionPassword = ""
	}

	// 清空旧版本密码字段（如果存在）
	if config.EncryptionPassword != "" {
		// 保留旧密码字段标记，但实际不再使用
		config.EncryptionPassword = ""
	}

	if err := as.storage.SaveSyncConfig(config); err != nil {
		return fmt.Errorf("failed to save encryption config: %w", err)
	}

	// Enable local storage encryption (使用系统密钥环)
	if enabled {
		as.storage.EnableEncryption("") // 新版本不需要密码参数
	}

	return as.gistSync.SetEncryption(enabled, password)
}

func (as *AppService) ValidateGitHubToken(token string) error {
	gs := NewGistSyncService(token, "")
	return gs.ValidateToken()
}

// GetGistStatus checks the current status of the Gist connection
// returns: "valid", "invalid_token", "gist_not_found", "error"
func (as *AppService) GetGistStatus() (string, error) {
	config, err := as.storage.LoadSyncConfig()
	if err != nil {
		return "error", err
	}

	if config.GitHubToken == "" {
		return "unconfigured", nil
	}

	gs := NewGistSyncService(config.GitHubToken, config.GistID)

	// Setup encryption if enabled (required to validate encrypted Gist content)
	if config.EnableEncryption {
		password := config.GistEncryptionPassword
		// 如果新字段为空但旧字段有值，使用旧字段（迁移场景）
		if password == "" && config.EncryptionPassword != "" {
			password = config.EncryptionPassword
		}
		gs.SetEncryption(config.EnableEncryption, password)
	}

	// 1. Validate Token
	if err := gs.ValidateToken(); err != nil {
		return "invalid_token", err
	}

	if config.GistID == "" {
		return "no_gist", nil
	}

	// 2. Validate Gist ID Existence
	_, err = gs.GetLatestVersion()
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return "gist_not_found", fmt.Errorf("gist not found")
		}
		// If it's just empty or other non-404 error, we might consider it valid or error
		// treating other errors as generic errors
		return "error", err
	}

	return "valid", nil
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
		if config.EnableEncryption {
			password := config.GistEncryptionPassword
			// 如果新字段为空但旧字段有值，使用旧字段（迁移场景）
			if password == "" && config.EncryptionPassword != "" {
				password = config.EncryptionPassword
			}
			as.gistSync.SetEncryption(config.EnableEncryption, password)
		}
	}

	// Collect all agents' COMPLETE configurations (not just servers)
	agents, err := as.detector.DetectInstalledAgents()
	if err != nil {
		return fmt.Errorf("failed to detect agents: %w", err)
	}

	// CRITICAL FIX: First fetch remote configs to preserve agents not installed locally
	// This prevents cross-device data loss where uninstalled agents' configs would be deleted
	remoteConfigs, err := as.gistSync.PullAgentConfigsFromGist()
	if err != nil {
		// If Gist is not found (404) or empty, start with empty map
		if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") ||
			strings.Contains(err.Error(), "mcp-config.json not found") {
			remoteConfigs = make(map[string]interface{})
			println("No existing remote config found, starting fresh")
		} else {
			return fmt.Errorf("failed to fetch remote configs before push: %w", err)
		}
	}

	// Start with remote configs to preserve agents not installed locally
	allAgentConfigs := make(map[string]interface{})
	preservedCount := 0
	for agentID, config := range remoteConfigs {
		allAgentConfigs[agentID] = config
		preservedCount++
	}
	if preservedCount > 0 {
		println(fmt.Sprintf("Preserved %d agent configs from remote (may include agents not installed locally)", preservedCount))
	}

	// Now overlay with local detected agents (local takes precedence for installed agents)
	pushedCount := 0
	for _, agent := range agents {
		if agent.Status == "detected" {
			agentConfig, err := as.GetAgentMCPConfig(agent.ID)
			if err != nil {
				println(fmt.Sprintf("Warning: failed to read config from %s: %v", agent.ID, err))
				continue
			}

			// Store the COMPLETE config for this agent (overwrites remote if exists)
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
		if config.EnableEncryption {
			password := config.GistEncryptionPassword
			// 如果新字段为空但旧字段有值，使用旧字段（迁移场景）
			if password == "" && config.EncryptionPassword != "" {
				password = config.EncryptionPassword
			}
			as.gistSync.SetEncryption(config.EnableEncryption, password)
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
		if config.EnableEncryption {
			password := config.GistEncryptionPassword
			// 如果新字段为空但旧字段有值，使用旧字段（迁移场景）
			if password == "" && config.EncryptionPassword != "" {
				password = config.EncryptionPassword
			}
			as.gistSync.SetEncryption(config.EnableEncryption, password)
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

// SyncReadyStatus represents the complete sync configuration status
type SyncReadyStatus struct {
	Ready           bool     `json:"ready"`            // True if sync is fully configured and ready
	HasToken        bool     `json:"has_token"`        // Has GitHub token
	HasGistID       bool     `json:"has_gist_id"`      // Has Gist ID
	EncryptionReady bool     `json:"encryption_ready"` // Encryption is properly configured
	Message         string   `json:"message"`          // Human-readable status message
	MissingItems    []string `json:"missing_items"`    // List of missing configuration items
}

// GetSyncReadyStatus checks if sync is fully configured and ready to use
// This performs a complete check including encryption status
func (as *AppService) GetSyncReadyStatus() (*SyncReadyStatus, error) {
	config, err := as.storage.LoadSyncConfig()
	if err != nil {
		return &SyncReadyStatus{
			Ready:   false,
			Message: "Failed to load sync configuration",
		}, nil
	}

	status := &SyncReadyStatus{
		HasToken:     config.GitHubToken != "",
		HasGistID:    config.GistID != "",
		MissingItems: []string{},
	}

	// Check GitHub Token
	if !status.HasToken {
		status.MissingItems = append(status.MissingItems, "GitHub Token")
	}

	// Check Gist ID
	if !status.HasGistID {
		status.MissingItems = append(status.MissingItems, "Gist ID")
	}

	// Check Encryption (CRITICAL: Gist sync requires encryption)
	// Encryption is ready if:
	// 1. EnableEncryption is true AND
	// 2. Either GistEncryptionPassword is set OR system crypto is available
	if config.EnableEncryption {
		hasPassword := config.GistEncryptionPassword != "" || config.EncryptionPassword != ""

		// Also check if system crypto (SecureCrypto) is available
		crypto, cryptoErr := NewSecureCrypto()
		hasSystemCrypto := cryptoErr == nil && crypto != nil && crypto.IsEnabled()

		status.EncryptionReady = hasPassword || hasSystemCrypto
	} else {
		status.EncryptionReady = false
	}

	if !status.EncryptionReady {
		status.MissingItems = append(status.MissingItems, "Encryption Password")
	}

	// Determine overall readiness
	status.Ready = status.HasToken && status.HasGistID && status.EncryptionReady

	// Generate message
	if status.Ready {
		status.Message = "Sync is fully configured and ready"
	} else if len(status.MissingItems) > 0 {
		status.Message = fmt.Sprintf("Missing configuration: %s", strings.Join(status.MissingItems, ", "))
	} else {
		status.Message = "Sync is not configured"
	}

	return status, nil
}

// ResetEncryptedConfig resets encrypted config when decryption fails
// This backs up the corrupted file and allows user to reconfigure
func (as *AppService) ResetEncryptedConfig() error {
	return as.storage.ResetEncryptedConfig()
}

// SaveSyncConfig saves the sync configuration
func (as *AppService) SaveSyncConfig(config models.SyncConfig) error {
	// Auto-detect existing Gist if ID is missing but token is provided
	if config.GistID == "" && config.GitHubToken != "" {
		tempGistSync := NewGistSyncService(config.GitHubToken, "")
		existingID, err := tempGistSync.FindExistingGist()
		if err == nil && existingID != "" {
			config.GistID = existingID
			println(fmt.Sprintf("Auto-detected existing Gist ID: %s", existingID))
		}
	}

	// Update gitignore if token is present
	if config.GitHubToken != "" {
		if err := as.updateGitignore(); err != nil {
			fmt.Printf("Warning: failed to update .gitignore: %v\n", err)
		}
	}

	// Initialize gist sync service if configured
	if config.GitHubToken != "" && config.GistID != "" {
		as.gistSync = NewGistSyncService(config.GitHubToken, config.GistID)
		if config.EnableEncryption {
			password := config.GistEncryptionPassword
			// 如果新字段为空但旧字段有值，使用旧字段（迁移场景）
			if password == "" && config.EncryptionPassword != "" {
				password = config.EncryptionPassword
			}
			as.gistSync.SetEncryption(config.EnableEncryption, password)
		}
	}

	return as.storage.SaveSyncConfig(config)
}

func (as *AppService) GetSyncLogs(limit int) ([]models.SyncLog, error) {
	return as.storage.GetSyncLogs(limit)
}

func (as *AppService) GetAgentMCPConfig(agentID string) (map[string]interface{}, error) {
	configPath, err := as.detector.GetAgentConfigPath(agentID)
	if err != nil {
		return nil, err
	}

	// Check if this is a TOML format (Codex)
	format := as.configLoader.GetFormat(agentID)
	if format == "codex_toml" {
		// Use TOML adapter for Codex
		servers, err := as.tomlAdapter.GetMCPServersAsStandard(configPath)
		if err != nil {
			return nil, err
		}
		keyName := as.configLoader.GetConfigKey(agentID)
		return map[string]interface{}{
			keyName: servers,
		}, nil
	}

	// Read the config file (JSON format)
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
	configPaths, err := as.detector.GetAllAgentConfigPaths(agentID)
	// If no existing paths found, try to get at least one default path to create
	if err != nil || len(configPaths) == 0 {
		defaultPath, err := as.detector.GetAgentConfigPath(agentID)
		if err != nil {
			return err
		}
		configPaths = []string{defaultPath}
	}

	var errorMessages []string

	for _, configPath := range configPaths {
		// Check if this is a TOML format (Codex)
		format := as.configLoader.GetFormat(agentID)
		if format == "codex_toml" {
			// Extract servers from input config
			keyName := as.configLoader.GetConfigKey(agentID)
			var servers map[string]interface{}
			if serversInterface, ok := mcpServersConfig[keyName]; ok {
				if serversMap, ok := serversInterface.(map[string]interface{}); ok {
					servers = serversMap
				}
			}

			// Apply Windows-specific transformation based on this specific path
			// Windows paths: wrap npx with cmd /c
			// WSL paths: UNWRAP cmd /c to get bare npx (source may already have cmd /c from Windows)
			if as.windowsSvc.ShouldApplyWindowsTransformation(configPath) {
				println(fmt.Sprintf("  [%s] 应用 Windows npx 命令转换 (wrap)", configPath))
				// Convert to MCPServer slice, apply transformation, and convert back
				mcpServers := as.convertServersDataToMCPServers(servers)
				mcpServers = as.windowsSvc.ApplyWindowsTransformation(mcpServers, true)
				servers = as.convertMCPServersToServersData(mcpServers)
			} else if as.windowsSvc.IsWindows() && as.windowsSvc.IsWSLPath(configPath) {
				println(fmt.Sprintf("  [%s] WSL环境，解包 Windows cmd /c 命令", configPath))
				// WSL is Linux - UNWRAP any cmd /c wrappers from the source config
				mcpServers := as.convertServersDataToMCPServers(servers)
				mcpServers = as.windowsSvc.ApplyWindowsTransformation(mcpServers, false) // false = unwrap
				servers = as.convertMCPServersToServersData(mcpServers)
			}

			// Use TOML adapter for Codex
			if err := as.tomlAdapter.SetMCPServersFromStandard(configPath, servers); err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("failed to save to %s: %v", configPath, err))
			}
			continue
		}

		// Read the full config file first (JSON format)
		// If file doesn't exist, create an empty map
		var fullConfig map[string]interface{}
		data, err := os.ReadFile(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				fullConfig = make(map[string]interface{})
			} else {
				errorMessages = append(errorMessages, fmt.Sprintf("failed to read %s: %v", configPath, err))
				continue
			}
		} else {
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

			if err := json.Unmarshal(cleanedData, &fullConfig); err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("failed to parse %s: %v", configPath, err))
				continue
			}
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

		// Apply Windows-specific transformation based on this specific path
		// Windows paths: wrap npx with cmd /c
		// WSL paths: UNWRAP cmd /c to get bare npx (source may already have cmd /c from Windows)
		if as.windowsSvc.ShouldApplyWindowsTransformation(configPath) {
			println(fmt.Sprintf("  [%s] 应用 Windows npx 命令转换 (wrap)", configPath))
			if serversDataMap, ok := serversData.(map[string]interface{}); ok {
				mcpServers := as.convertServersDataToMCPServers(serversDataMap)
				mcpServers = as.windowsSvc.ApplyWindowsTransformation(mcpServers, true)
				serversData = as.convertMCPServersToServersData(mcpServers)
			}
		} else if as.windowsSvc.IsWindows() && as.windowsSvc.IsWSLPath(configPath) {
			println(fmt.Sprintf("  [%s] WSL环境，解包 Windows cmd /c 命令", configPath))
			// WSL is Linux - UNWRAP any cmd /c wrappers from the source config
			if serversDataMap, ok := serversData.(map[string]interface{}); ok {
				mcpServers := as.convertServersDataToMCPServers(serversDataMap)
				mcpServers = as.windowsSvc.ApplyWindowsTransformation(mcpServers, false) // false = unwrap
				serversData = as.convertMCPServersToServersData(mcpServers)
			}
		}

		// Update the config with target format
		if serversData != nil {
			fullConfig[targetKeyName] = serversData
		}

		// Write back the full config
		updatedData, err := json.MarshalIndent(fullConfig, "", "  ")
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("failed to marshal config for %s: %v", configPath, err))
			continue
		}

		if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("failed to write to %s: %v", configPath, err))
			continue
		}

		println(fmt.Sprintf("Successfully saved config to: %s", configPath))
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("errors occurred while saving: %s", strings.Join(errorMessages, "; "))
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

	// Debug: show initial read data
	sourceKeyDebug := as.configLoader.GetConfigKey(sourceAgentID)
	if serversData, ok := sourceConfig[sourceKeyDebug]; ok {
		if serverMap, ok := serversData.(map[string]interface{}); ok {
			println(fmt.Sprintf("[Debug] 初始读取到 %d 个服务器", len(serverMap)))
			for name, serverInterface := range serverMap {
				if sc, ok := serverInterface.(map[string]interface{}); ok {
					println(fmt.Sprintf("  服务器 '%s': %d 个字段", name, len(sc)))
					for key := range sc {
						println(fmt.Sprintf("    - %s", key))
					}
				}
			}
		}
	}

	// Unwrap Windows-specific commands from source if it's already wrapped
	if as.windowsSvc.IsWindows() {
		println("  检测源配置中是否需要解包 Windows npx 命令")

		// Extract servers data from source
		sourceKey := as.configLoader.GetConfigKey(sourceAgentID)
		serversData, ok := sourceConfig[sourceKey]
		if !ok {
			serversData = make(map[string]interface{})
		}

		// Convert to MCPServer slice, unwrap, and convert back
		servers := as.convertServersDataToMCPServers(serversData)
		servers = as.windowsSvc.ApplyWindowsTransformation(servers, false)
		serversData = as.convertMCPServersToServersData(servers)

		sourceConfig[sourceKey] = serversData
		println("  源配置 Windows npx 命令解包完成")
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

	// NOTE: Windows-specific npx wrapping is now handled per-path in SaveAgentMCPConfig
	// This allows proper handling when an agent has both Windows and WSL config paths

	// Normalize format names (codex_toml is already converted to standard by GetAgentMCPConfig)
	normalizedSourceFormat := sourceFormat
	normalizedTargetFormat := targetFormat
	if normalizedSourceFormat == "codex_toml" {
		normalizedSourceFormat = "standard"
	}
	if normalizedTargetFormat == "codex_toml" {
		normalizedTargetFormat = "standard"
	}

	// Convert format if needed
	if normalizedSourceFormat != normalizedTargetFormat {
		println(fmt.Sprintf("  转换格式: %s -> %s", normalizedSourceFormat, normalizedTargetFormat))

		// Try to use the configuration-based transform rule first
		transformRule := as.configLoader.GetTransformRule(normalizedSourceFormat, normalizedTargetFormat)
		if transformRule != nil {
			serversData = as.configLoader.ApplyTransformRule(serversData, transformRule)
			println(fmt.Sprintf("  使用配置规则进行转换"))
		} else {
			// Fall back to hardcoded conversions
			println(fmt.Sprintf("  未找到配置规则，使用内置转换"))
			if normalizedSourceFormat == "standard" && normalizedTargetFormat == "zed" {
				serversData = convertStandardToZed(serversData)
			} else if normalizedSourceFormat == "zed" && normalizedTargetFormat == "standard" {
				serversData = convertZedToStandard(serversData)
			}
		}
	} else {
		println("  格式相同,无需转换")
	}

	// Save to target agent with appropriate key name
	targetConfig := map[string]interface{}{
		targetKey: serversData,
	}

	return as.SaveAgentMCPConfig(targetAgentID, targetConfig)
}

// Helper methods for Windows transformations
func (as *AppService) convertServersDataToMCPServers(serversData interface{}) []models.MCPServer {
	var servers []models.MCPServer
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

				// Handle args - support both []interface{} and []string
				if argsInterface, ok := serverMap["args"]; ok {
					switch args := argsInterface.(type) {
					case []interface{}:
						for _, arg := range args {
							if argStr, ok := arg.(string); ok {
								server.Args = append(server.Args, argStr)
							}
						}
					case []string:
						server.Args = args
					}
				}

				// Handle env - support both map[string]interface{} and map[string]string
				if envInterface, ok := serverMap["env"]; ok {
					switch env := envInterface.(type) {
					case map[string]interface{}:
						server.Env = make(map[string]string)
						for k, v := range env {
							if strVal, ok := v.(string); ok {
								server.Env[k] = strVal
							}
						}
					case map[string]string:
						server.Env = env
					}
				}
			}
			servers = append(servers, server)
		}
	}
	return servers
}

func (as *AppService) convertMCPServersToServersData(servers []models.MCPServer) map[string]interface{} {
	serversData := make(map[string]interface{})
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
		serversData[server.Name] = serverConfig
	}
	return serversData
}

// ConvertAgentConfig converts MCP config from one agent format to another
func (as *AppService) ConvertAgentConfig(sourceAgentID, targetAgentID string, sourceConfig map[string]interface{}) (*ConversionResult, error) {
	return as.converter.ConvertAgentConfig(sourceAgentID, targetAgentID, sourceConfig)
}

// ConvertToCodex converts any agent config to Codex format
func (as *AppService) ConvertToCodex(sourceAgentID string, sourceConfig map[string]interface{}) (*ConversionResult, error) {
	return as.converter.ConvertToCodex(sourceAgentID, sourceConfig)
}

// ConvertFromCodex converts Codex config to any agent format
func (as *AppService) ConvertFromCodex(targetAgentID string, codexConfig map[string]interface{}) (*ConversionResult, error) {
	return as.converter.ConvertFromCodex(targetAgentID, codexConfig)
}

// BatchConvertConfig converts config to multiple target formats
func (as *AppService) BatchConvertConfig(sourceAgentID string, sourceConfig map[string]interface{}, targetAgentIDs []string) ([]*ConversionResult, error) {
	return as.converter.BatchConvertConfig(sourceAgentID, sourceConfig, targetAgentIDs)
}

// ValidateConfigFormat validates if a config matches expected format
func (as *AppService) ValidateConfigFormat(agentID string, config map[string]interface{}) (bool, []string) {
	return as.converter.ValidateConfigFormat(agentID, config)
}

// ExportConversionAsJSON exports a conversion result as JSON string
func (as *AppService) ExportConversionAsJSON(result *ConversionResult) (string, error) {
	return as.converter.ExportConversionAsJSON(result)
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
		if config.EnableEncryption {
			password := config.GistEncryptionPassword
			// 如果新字段为空但旧字段有值，使用旧字段（迁移场景）
			if password == "" && config.EncryptionPassword != "" {
				password = config.EncryptionPassword
			}
			as.gistSync.SetEncryption(config.EnableEncryption, password)
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
		// If Gist is not found (404), no conflict, treat as fresh push
		if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return &models.SyncConflict{HasConflict: false}, nil
		}
		// If other error, return it
		return nil, err
	}

	// Compare hashes
	if remoteVersion != nil && localVersion.Hash != remoteVersion.Hash {
		// Double check with deep comparison to avoid false positives due to formatting
		details, err := as.GetConflictDetails()
		if err == nil && details != nil {
			hasRealDiff := false
			for _, diff := range details.Agents {
				if len(diff.LocalOnly) > 0 || len(diff.RemoteOnly) > 0 || len(diff.Conflicts) > 0 {
					hasRealDiff = true
					break
				}
			}
			if !hasRealDiff {
				// Hashes differ but content is effectively the same
				return &models.SyncConflict{HasConflict: false}, nil
			}
		}

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
		if config.EnableEncryption {
			password := config.GistEncryptionPassword
			// 如果新字段为空但旧字段有值，使用旧字段（迁移场景）
			if password == "" && config.EncryptionPassword != "" {
				password = config.EncryptionPassword
			}
			as.gistSync.SetEncryption(config.EnableEncryption, password)
		}
	}

	// Get local version
	localVersion, err := as.getLatestLocalVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get local version: %w", err)
	}

	// Get remote version from Gist
	remoteVersion, err := as.gistSync.GetLatestVersion()
	if err != nil {
		// If Gist is not found (404), there is no remote version to conflict with
		if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return &models.SyncConflict{HasConflict: false}, nil
		}
		return nil, fmt.Errorf("failed to get remote version: %w", err)
	}

	if remoteVersion == nil {
		return &models.SyncConflict{HasConflict: false}, nil
	}

	// Compare hashes - if local is newer than remote, there's unsaved local changes
	if localVersion != nil && localVersion.Timestamp.After(remoteVersion.Timestamp) && localVersion.Hash != remoteVersion.Hash {
		// Double check with deep comparison to avoid false positives due to formatting
		details, err := as.GetConflictDetails()
		if err == nil && details != nil {
			hasRealDiff := false
			for _, diff := range details.Agents {
				if len(diff.LocalOnly) > 0 || len(diff.RemoteOnly) > 0 || len(diff.Conflicts) > 0 {
					hasRealDiff = true
					break
				}
			}
			if !hasRealDiff {
				// Hashes differ but content is effectively the same
				return &models.SyncConflict{HasConflict: false}, nil
			}
		}

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

	// Ensure gistSync is initialized before any operation
	if as.gistSync == nil {
		config, err := as.storage.LoadSyncConfig()
		if err != nil {
			return fmt.Errorf("failed to load sync config: %w", err)
		}
		if config.GitHubToken == "" || config.GistID == "" {
			return fmt.Errorf("GitHub token or Gist ID not configured")
		}
		as.gistSync = NewGistSyncService(config.GitHubToken, config.GistID)
		if config.EnableEncryption {
			password := config.GistEncryptionPassword
			if password == "" && config.EncryptionPassword != "" {
				password = config.EncryptionPassword
			}
			as.gistSync.SetEncryption(config.EnableEncryption, password)
		}
	}

	switch resolution {
	case "keep_local":
		// Just push local to remote
		return as.PushAllAgentsToGist()

	case "use_remote":
		// Just pull remote to local
		_, err := as.PullFromGist()
		return err

	case "merge":
		// 1. Pull remote configs
		remoteConfigs, err := as.gistSync.PullAgentConfigsFromGist()
		if err != nil {
			// If Gist is not found (404), treat as empty remote config instead of error
			if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
				remoteConfigs = make(map[string]interface{})
			} else {
				return fmt.Errorf("failed to pull remote configs for merge: %w", err)
			}
		}

		// 2. Perform merge
		if err := as.mergeAndApplyConfigs(remoteConfigs); err != nil {
			return fmt.Errorf("failed to merge configs: %w", err)
		}

		// 3. Push merged result back to remote to sync state
		if err := as.PushAllAgentsToGist(); err != nil {
			return fmt.Errorf("merged successfully locally but failed to push to gist: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown resolution type: %s", resolution)
	}
}

// mergeAndApplyConfigs performs the smart merge logic
func (as *AppService) mergeAndApplyConfigs(remoteConfigs map[string]interface{}) error {
	// Detect local agents
	localAgents, err := as.detector.DetectInstalledAgents()
	if err != nil {
		return err
	}

	// Try to find a base configuration (last synced version) for 3-way merge
	baseConfigs, err := as.findBaseConfig()
	if err != nil {
		println(fmt.Sprintf("Warning: failed to load base config, falling back to 2-way merge: %v", err))
		baseConfigs = make(map[string]interface{})
	}

	// Track which remote agents are processed
	processedAgentIDs := make(map[string]bool)
	mergedCount := 0

	for _, agent := range localAgents {
		if agent.Status != "detected" {
			continue
		}

		processedAgentIDs[agent.ID] = true
		keyName := as.configLoader.GetConfigKey(agent.ID)

		// 1. Get Local Config
		localConfigWrap, err := as.GetAgentMCPConfig(agent.ID)
		if err != nil {
			println(fmt.Sprintf("Warning: skipping merge for agent %s (read local failed): %v", agent.ID, err))
			continue
		}

		var localServers []models.MCPServer
		if localServersData, ok := localConfigWrap[keyName]; ok {
			localServers = as.convertServersDataToMCPServers(localServersData)
		} else {
			localServers = []models.MCPServer{}
		}

		// 2. Get Remote Config
		var remoteServers []models.MCPServer
		if remoteConfig, ok := remoteConfigs[agent.ID]; ok {
			if remoteConfigMap, ok := remoteConfig.(map[string]interface{}); ok {
				if remoteServersData, ok := remoteConfigMap[keyName]; ok {
					remoteServers = as.convertServersDataToMCPServers(remoteServersData)
				}
			}
		}

		// 3. Get Base Config
		var baseServers []models.MCPServer
		if baseConfig, ok := baseConfigs[agent.ID]; ok {
			if baseConfigMap, ok := baseConfig.(map[string]interface{}); ok {
				if baseServersData, ok := baseConfigMap[keyName]; ok {
					baseServers = as.convertServersDataToMCPServers(baseServersData)
				}
			}
		}

		// 4. Merge Logic (3-way)
		mergedServers := as.mergeServerLists(localServers, remoteServers, baseServers)

		// 5. Convert back and Save
		mergedServersData := as.convertMCPServersToServersData(mergedServers)

		newConfigWrap := map[string]interface{}{
			keyName: mergedServersData,
		}

		if err := as.SaveAgentMCPConfig(agent.ID, newConfigWrap); err != nil {
			return fmt.Errorf("failed to save merged config for agent %s: %w", agent.ID, err)
		}

		mergedCount++
	}

	// Log remote-only agents that will be preserved during push
	// These are agents configured on other devices but not installed locally
	remoteOnlyCount := 0
	for agentID := range remoteConfigs {
		if !processedAgentIDs[agentID] {
			remoteOnlyCount++
			println(fmt.Sprintf("Remote-only agent (not installed locally, will be preserved): %s", agentID))
		}
	}

	if remoteOnlyCount > 0 {
		println(fmt.Sprintf("Note: %d remote agent(s) not installed locally - their configs will be preserved during push", remoteOnlyCount))
	}

	println(fmt.Sprintf("Successfully merged configurations for %d local agents", mergedCount))
	return nil
}

// findBaseConfig attempts to find the last known consistent configuration
func (as *AppService) findBaseConfig() (map[string]interface{}, error) {
	// Look for the latest version with source="gist"
	versions, err := as.storage.ListConfigVersions(10)
	if err != nil {
		return nil, err
	}

	var baseVersion *models.ConfigVersion
	for _, v := range versions {
		if v.Source == "gist" {
			baseVersion = &v
			break
		}
	}

	if baseVersion == nil {
		return nil, fmt.Errorf("no base version found")
	}

	// Parse the content
	var configs map[string]interface{}
	if err := json.Unmarshal([]byte(baseVersion.Content), &configs); err != nil {
		return nil, fmt.Errorf("failed to parse base version: %w", err)
	}

	return configs, nil
}

// mergeServerLists implements a 3-way merge strategy
func (as *AppService) mergeServerLists(local, remote, base []models.MCPServer) []models.MCPServer {
	localMap := toServerMap(local)
	remoteMap := toServerMap(remote)
	baseMap := toServerMap(base)

	mergedMap := make(map[string]models.MCPServer)

	// Collect all IDs
	allIDs := make(map[string]bool)
	for id := range localMap {
		allIDs[id] = true
	}
	for id := range remoteMap {
		allIDs[id] = true
	}
	for id := range baseMap {
		allIDs[id] = true
	}

	for id := range allIDs {
		l, hasLocal := localMap[id]
		r, hasRemote := remoteMap[id]
		b, hasBase := baseMap[id]

		if hasLocal && hasRemote {
			if !hasBase {
				mergedMap[id] = mergeTwoServers(l, r)
			} else {
				mergedMap[id] = mergeThreeWay(l, r, b)
			}
		} else if hasLocal && !hasRemote {
			if hasBase {
				if !isDeepEqual(l, b) {
					mergedMap[id] = l
				}
			} else {
				mergedMap[id] = l
			}
		} else if !hasLocal && hasRemote {
			if hasBase {
				if !isDeepEqual(r, b) {
					mergedMap[id] = r
				}
			} else {
				mergedMap[id] = r
			}
		}
	}

	result := make([]models.MCPServer, 0, len(mergedMap))
	for _, s := range mergedMap {
		result = append(result, s)
	}
	return result
}

func toServerMap(servers []models.MCPServer) map[string]models.MCPServer {
	m := make(map[string]models.MCPServer)
	for _, s := range servers {
		m[s.ID] = s
	}
	return m
}

func mergeTwoServers(local, remote models.MCPServer) models.MCPServer {
	merged := local
	merged.Env = mergeEnv(local.Env, remote.Env)
	if merged.Description == "" {
		merged.Description = remote.Description
	}
	return merged
}

func mergeThreeWay(local, remote, base models.MCPServer) models.MCPServer {
	merged := local

	if local.Command == base.Command && remote.Command != base.Command {
		merged.Command = remote.Command
	}

	argsLocalChanged := !areStringSlicesEqual(local.Args, base.Args)
	argsRemoteChanged := !areStringSlicesEqual(remote.Args, base.Args)
	if !argsLocalChanged && argsRemoteChanged {
		merged.Args = remote.Args
	}

	finalEnv := make(map[string]string)
	allKeys := make(map[string]bool)
	for k := range base.Env {
		allKeys[k] = true
	}
	for k := range local.Env {
		allKeys[k] = true
	}
	for k := range remote.Env {
		allKeys[k] = true
	}

	for k := range allKeys {
		vBase, inBase := base.Env[k]
		vLocal, inLocal := local.Env[k]
		vRemote, inRemote := remote.Env[k]

		if inLocal && inRemote {
			if vLocal == vRemote {
				finalEnv[k] = vLocal
			} else {
				if inBase && vLocal == vBase && vRemote != vBase {
					finalEnv[k] = vRemote
				} else {
					finalEnv[k] = vLocal
				}
			}
		} else if inLocal && !inRemote {
			if inBase && vLocal == vBase {
				// deleted
			} else {
				finalEnv[k] = vLocal
			}
		} else if !inLocal && inRemote {
			if inBase && vRemote == vBase {
				// deleted
			} else {
				finalEnv[k] = vRemote
			}
		}
	}
	merged.Env = finalEnv

	if local.Description == base.Description && remote.Description != base.Description {
		merged.Description = remote.Description
	}

	return merged
}

func mergeEnv(local, remote map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range remote {
		result[k] = v
	}
	for k, v := range local {
		result[k] = v
	}
	return result
}

func areStringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isDeepEqual(a, b models.MCPServer) bool {
	if a.ID != b.ID || a.Command != b.Command || a.Description != b.Description {
		return false
	}
	if !areStringSlicesEqual(a.Args, b.Args) {
		return false
	}
	if len(a.Env) != len(b.Env) {
		return false
	}
	for k, v := range a.Env {
		if b.Env[k] != v {
			return false
		}
	}
	return true
}

// GetConflictDetails generates a detailed comparison report between Local and Remote configurations
func (as *AppService) GetConflictDetails() (*models.ConflictDetails, error) {
	// 1. Pull Remote Configs (In-memory only)
	if as.gistSync == nil {
		// Initialize temporarily if needed (should be initialized by DetectConflict usually)
		config, err := as.storage.LoadSyncConfig()
		if err != nil {
			return nil, err
		}
		as.gistSync = NewGistSyncService(config.GitHubToken, config.GistID)
		// Setup encryption if enabled
		if config.EnableEncryption {
			password := config.GistEncryptionPassword
			// 如果新字段为空但旧字段有值，使用旧字段（迁移场景）
			if password == "" && config.EncryptionPassword != "" {
				password = config.EncryptionPassword
			}
			as.gistSync.SetEncryption(config.EnableEncryption, password)
		}
	}

	remoteConfigs, err := as.gistSync.PullAgentConfigsFromGist()
	if err != nil {
		// If Gist is not found (404), treat as empty remote config instead of error
		if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
			remoteConfigs = make(map[string]interface{})
		} else {
			return nil, fmt.Errorf("failed to fetch remote configs: %w", err)
		}
	}

	// 2. Get Local Agents
	localAgents, err := as.detector.DetectInstalledAgents()
	if err != nil {
		return nil, fmt.Errorf("failed to detect local agents: %w", err)
	}

	details := &models.ConflictDetails{
		Agents: make(map[string]models.AgentDiff),
	}

	// Track which remote agents have been processed
	processedRemoteAgents := make(map[string]bool)

	// 3. Compare Items for locally detected agents
	for _, agent := range localAgents {
		if agent.Status != "detected" {
			continue
		}

		processedRemoteAgents[agent.ID] = true

		// Get Local Servers
		localConfigWrap, err := as.GetAgentMCPConfig(agent.ID)
		if err != nil {
			continue
		}
		keyName := as.configLoader.GetConfigKey(agent.ID)
		var localServers []models.MCPServer
		if localData, ok := localConfigWrap[keyName]; ok {
			localServers = as.convertServersDataToMCPServers(localData)
		}

		// Get Remote Servers
		var remoteServers []models.MCPServer
		if remoteConfig, ok := remoteConfigs[agent.ID]; ok {
			if remoteConfigMap, ok := remoteConfig.(map[string]interface{}); ok {
				if remoteData, ok := remoteConfigMap[keyName]; ok {
					remoteServers = as.convertServersDataToMCPServers(remoteData)
				}
			}
		}

		// Compare to generate Diff
		diff := as.generateAgentDiff(agent.ID, localServers, remoteServers)
		details.Agents[agent.ID] = diff
	}

	// 4. CRITICAL: Also show agents that exist only in remote (not installed locally)
	// This ensures users see all remote configurations and know they won't be lost
	for agentID, remoteConfig := range remoteConfigs {
		if processedRemoteAgents[agentID] {
			continue // Already processed above
		}

		// This agent exists in remote but not installed locally
		// Show all its servers as "RemoteOnly" with a special note
		var remoteServers []models.MCPServer
		if remoteConfigMap, ok := remoteConfig.(map[string]interface{}); ok {
			// Try common config keys since we don't know the agent definition
			for _, keyName := range []string{"mcpServers", "context_servers", "mcp_servers"} {
				if remoteData, ok := remoteConfigMap[keyName]; ok {
					remoteServers = as.convertServersDataToMCPServers(remoteData)
					if len(remoteServers) > 0 {
						break
					}
				}
			}
		}

		if len(remoteServers) > 0 {
			diff := models.AgentDiff{
				AgentID:      agentID,
				LocalOnly:    []models.MCPServer{}, // No local servers (not installed)
				RemoteOnly:   remoteServers,        // All remote servers
				Conflicts:    []models.ServerConflict{},
				NotInstalled: true, // Mark as not installed locally
			}
			details.Agents[agentID] = diff
			println(fmt.Sprintf("Remote-only agent in conflict details: %s (not installed locally, %d servers)", agentID, len(remoteServers)))
		}
	}

	return details, nil
}

func (as *AppService) generateAgentDiff(agentID string, local, remote []models.MCPServer) models.AgentDiff {
	diff := models.AgentDiff{
		AgentID:    agentID,
		LocalOnly:  []models.MCPServer{},
		RemoteOnly: []models.MCPServer{},
		Conflicts:  []models.ServerConflict{},
	}

	localMap := toServerMap(local)
	remoteMap := toServerMap(remote)

	allIDs := make(map[string]bool)
	for id := range localMap {
		allIDs[id] = true
	}
	for id := range remoteMap {
		allIDs[id] = true
	}

	for id := range allIDs {
		l, hasLocal := localMap[id]
		r, hasRemote := remoteMap[id]

		if hasLocal && !hasRemote {
			diff.LocalOnly = append(diff.LocalOnly, l)
		} else if !hasLocal && hasRemote {
			diff.RemoteOnly = append(diff.RemoteOnly, r)
		} else {
			// Both exist - Check for conflict
			if !isDeepEqual(l, r) {
				diff.Conflicts = append(diff.Conflicts, models.ServerConflict{
					ServerID: id,
					Local:    l,
					Remote:   r,
				})
			}
		}
	}
	return diff
}

// ResolveConflictSelective resolves conflicts by applying specific decisions for specific items
// decisions map key format: "AgentID:ServerID" (e.g. "claude:filesystem")
// decisions map value: "local" | "remote"
func (as *AppService) ResolveConflictSelective(decisions map[string]string) error {
	// 1. Pull Remote
	remoteConfigs, err := as.gistSync.PullAgentConfigsFromGist()
	if err != nil {
		return fmt.Errorf("failed to pull remote for selective merge: %w", err)
	}

	// 2. Load Base (for smart merge fallbacks)
	baseConfigs, _ := as.findBaseConfig()

	// 3. Iterate and Merge
	localAgents, _ := as.detector.DetectInstalledAgents()
	mergedCount := 0

	for _, agent := range localAgents {
		if agent.Status != "detected" {
			continue
		}

		keyName := as.configLoader.GetConfigKey(agent.ID)

		// Local
		localConfigWrap, _ := as.GetAgentMCPConfig(agent.ID)
		var localServers []models.MCPServer
		if localData, ok := localConfigWrap[keyName]; ok {
			localServers = as.convertServersDataToMCPServers(localData)
		}

		// Remote
		var remoteServers []models.MCPServer
		if remoteConfig, ok := remoteConfigs[agent.ID]; ok {
			if remoteMap, ok := remoteConfig.(map[string]interface{}); ok {
				if remoteData, ok := remoteMap[keyName]; ok {
					remoteServers = as.convertServersDataToMCPServers(remoteData)
				}
			}
		}

		// Base
		var baseServers []models.MCPServer
		if baseConfig, ok := baseConfigs[agent.ID]; ok {
			if baseMap, ok := baseConfig.(map[string]interface{}); ok {
				if baseData, ok := baseMap[keyName]; ok {
					baseServers = as.convertServersDataToMCPServers(baseData)
				}
			}
		}

		// Perform Selective Merge
		mergedServers := as.mergeSelective(agent.ID, localServers, remoteServers, baseServers, decisions)

		// Save
		mergedData := as.convertMCPServersToServersData(mergedServers)
		newWrap := map[string]interface{}{keyName: mergedData}

		if err := as.SaveAgentMCPConfig(agent.ID, newWrap); err != nil {
			return err
		}
		mergedCount++
	}

	// 4. Push Result
	return as.PushAllAgentsToGist()
}

func (as *AppService) mergeSelective(agentID string, local, remote, base []models.MCPServer, decisions map[string]string) []models.MCPServer {
	localMap := toServerMap(local)
	remoteMap := toServerMap(remote)
	baseMap := toServerMap(base)

	mergedMap := make(map[string]models.MCPServer)
	allIDs := make(map[string]bool)
	for id := range localMap {
		allIDs[id] = true
	}
	for id := range remoteMap {
		allIDs[id] = true
	}
	for id := range baseMap {
		allIDs[id] = true
	}

	for id := range allIDs {
		l, hasLocal := localMap[id]
		r, hasRemote := remoteMap[id]
		b, hasBase := baseMap[id]

		// Check for explicit decision
		decisionKey := fmt.Sprintf("%s:%s", agentID, id)
		decision, hasDecision := decisions[decisionKey]

		if hasDecision {
			if decision == "local" && hasLocal {
				mergedMap[id] = l
				continue
			} else if decision == "remote" && hasRemote {
				mergedMap[id] = r
				continue
			} else if decision == "delete" {
				// Explict delete instruction (if UI supports it)
				continue
			}
		}

		// Fallback to Smart 3-Way Merge if no explicit decision
		if hasLocal && hasRemote {
			if !hasBase {
				mergedMap[id] = mergeTwoServers(l, r)
			} else {
				mergedMap[id] = mergeThreeWay(l, r, b)
			}
		} else if hasLocal && !hasRemote {
			// Local Only
			if hasBase && !isDeepEqual(l, b) {
				// Local modified, Remote deleted -> Keep Local (unless logic changes)
				mergedMap[id] = l
			} else if !hasBase {
				// New Local
				mergedMap[id] = l
			}
		} else if !hasLocal && hasRemote {
			// Remote Only
			if hasBase && !isDeepEqual(r, b) {
				// Remote modified, Local deleted -> Keep Remote
				mergedMap[id] = r
			} else if !hasBase {
				// New Remote
				mergedMap[id] = r
			}
		}
	}

	result := make([]models.MCPServer, 0, len(mergedMap))
	for _, s := range mergedMap {
		result = append(result, s)
	}
	return result
}

// GetConfigVersions retrieves the configuration version history
func (as *AppService) GetConfigVersions(limit int) ([]models.ConfigVersion, error) {
	return as.storage.ListConfigVersions(limit)
}

// updateGitignore adds .mcp-sync/ to .gitignore if not present
func (as *AppService) updateGitignore() error {
	homeDir := os.Getenv("USERPROFILE")
	if homeDir == "" {
		homeDir = os.Getenv("HOME")
	}
	if homeDir == "" {
		return fmt.Errorf("could not determine user home directory")
	}

	gitignorePath := filepath.Join(homeDir, ".gitignore")

	// Check if file exists
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new file
			return os.WriteFile(gitignorePath, []byte(".mcp-sync/\n"), 0644)
		}
		return err
	}

	// Check if already ignored
	contentStr := string(content)
	if strings.Contains(contentStr, ".mcp-sync/") {
		return nil
	}

	// Append to file
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if !strings.HasSuffix(contentStr, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	if _, err := f.WriteString(".mcp-sync/\n"); err != nil {
		return err
	}

	return nil
}
