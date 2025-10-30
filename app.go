package main

import (
	"context"
	"fmt"
	"mcp-sync/models"
	"mcp-sync/services"
)

// App struct
type App struct {
	ctx        context.Context
	appService *services.AppService
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	
	// Initialize app service
	appService, err := services.NewAppService()
	if err != nil {
		println("Error initializing app service:", err.Error())
		return
	}
	a.appService = appService
}

// DetectAgents detects installed agents on the system
func (a *App) DetectAgents() ([]models.Agent, error) {
	return a.appService.DetectAgents()
}

// InitializeGistSync sets up GitHub Gist synchronization
// Returns the Gist ID (either provided or auto-created)
func (a *App) InitializeGistSync(token, gistID string) (string, error) {
	// Validate token
	if err := a.appService.ValidateGitHubToken(token); err != nil {
		return "", err
	}
	
	// Initialize sync and save config
	return a.appService.InitializeGistSync(token, gistID)
}

// GetSyncConfig retrieves the current sync configuration
func (a *App) GetSyncConfig() (models.SyncConfig, error) {
	return a.appService.GetSyncConfig()
}

// SaveSyncConfig saves the sync configuration
func (a *App) SaveSyncConfig(config models.SyncConfig) error {
	return a.appService.SaveSyncConfig(config)
}

// PushAllAgentsToGist pushes all agents' configurations to GitHub Gist
func (a *App) PushAllAgentsToGist() error {
	return a.appService.PushAllAgentsToGist()
}

// PushToGist pushes configuration to GitHub Gist
func (a *App) PushToGist(servers []models.MCPServer) error {
	return a.appService.PushToGist(servers)
}

// PullFromGist pulls configuration from GitHub Gist
func (a *App) PullFromGist() ([]models.MCPServer, error) {
	return a.appService.PullFromGist()
}

// ApplyConfigToAgent applies MCP configuration to a specific agent
func (a *App) ApplyConfigToAgent(agentID string, servers []models.MCPServer) error {
	return a.appService.ApplyConfigToAgents(agentID, servers)
}

// ApplyConfigToAllAgents applies MCP configuration to all detected agents
func (a *App) ApplyConfigToAllAgents(servers []models.MCPServer) error {
	return a.appService.ApplyConfigToAllAgents(servers)
}

// GetConfigVersions retrieves the configuration version history
func (a *App) GetConfigVersions(limit int) ([]models.ConfigVersion, error) {
	return a.appService.GetConfigVersions(limit)
}

// GetSyncLogs retrieves the sync operation logs
func (a *App) GetSyncLogs(limit int) ([]models.SyncLog, error) {
	return a.appService.GetSyncLogs(limit)
}

// GetAgentMCPConfig reads the MCP configuration from a specific agent's config file
func (a *App) GetAgentMCPConfig(agentID string) (map[string]interface{}, error) {
	return a.appService.GetAgentMCPConfig(agentID)
}

// SaveAgentMCPConfig saves MCP configuration to a specific agent's config file
func (a *App) SaveAgentMCPConfig(agentID string, configJson map[string]interface{}) error {
	return a.appService.SaveAgentMCPConfig(agentID, configJson)
}

// SyncConfigBetweenAgents syncs configuration from source agent to target agent with automatic format conversion
func (a *App) SyncConfigBetweenAgents(sourceAgentID, targetAgentID string) error {
	return a.appService.SyncConfigBetweenAgents(sourceAgentID, targetAgentID)
}

// GetGistSecurityWarnings returns security warnings for Gist synchronization
func (a *App) GetGistSecurityWarnings() []map[string]string {
	return a.appService.GetGistSecurityWarnings()
}

// SetupGistEncryption setup encryption for Gist sync
func (a *App) SetupGistEncryption(enabled bool, password string) error {
	return a.appService.SetupGistEncryption(enabled, password)
}

// DetectPushConflict detects conflicts before pushing to Gist
func (a *App) DetectPushConflict() (*models.SyncConflict, error) {
	return a.appService.DetectPushConflict()
}

// DetectPullConflict detects conflicts before pulling from Gist
func (a *App) DetectPullConflict() (*models.SyncConflict, error) {
	return a.appService.DetectPullConflict()
}

// ResolveConflict resolves a detected conflict with the specified strategy
// resolution: "keep_local", "use_remote", "merge"
func (a *App) ResolveConflict(conflictType string, resolution string) error {
	return a.appService.ResolveConflict(conflictType, resolution)
}

// ConvertAgentConfig converts MCP config from one agent format to another
func (a *App) ConvertAgentConfig(sourceAgentID, targetAgentID string, sourceConfig map[string]interface{}) (*services.ConversionResult, error) {
	return a.appService.ConvertAgentConfig(sourceAgentID, targetAgentID, sourceConfig)
}

// ConvertToCodex converts any agent's config to Codex format
func (a *App) ConvertToCodex(sourceAgentID string, sourceConfig map[string]interface{}) (*services.ConversionResult, error) {
	return a.appService.ConvertToCodex(sourceAgentID, sourceConfig)
}

// ConvertFromCodex converts Codex config to any agent format
func (a *App) ConvertFromCodex(targetAgentID string, codexConfig map[string]interface{}) (*services.ConversionResult, error) {
	return a.appService.ConvertFromCodex(targetAgentID, codexConfig)
}

// BatchConvertConfig converts config to multiple target formats
func (a *App) BatchConvertConfig(sourceAgentID string, sourceConfig map[string]interface{}, targetAgentIDs []string) ([]*services.ConversionResult, error) {
	return a.appService.BatchConvertConfig(sourceAgentID, sourceConfig, targetAgentIDs)
}

// ValidateConfigFormat validates if a config matches expected format
func (a *App) ValidateConfigFormat(agentID string, config map[string]interface{}) (bool, []string) {
	return a.appService.ValidateConfigFormat(agentID, config)
}

// ExportConversionAsJSON exports a conversion result as JSON string
func (a *App) ExportConversionAsJSON(result *services.ConversionResult) (string, error) {
	return a.appService.ExportConversionAsJSON(result)
}

// Greet returns a greeting for the given name (kept for compatibility)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
