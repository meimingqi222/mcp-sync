package models

import "time"

type Agent struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Platform        string   `json:"platform"`      // windows, macos, linux
	Status          string   `json:"status"`        // detected, not_installed
	ConfigPaths     []string `json:"config_paths"`
	ExistingPaths   []string `json:"existing_paths"` // paths that actually exist
	Enabled         bool     `json:"enabled"`
}

type MCPServer struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	Enabled     bool              `json:"enabled"`
	Description string            `json:"description"`
	SupportedAgents []string      `json:"supported_agents"`
	CreatedAt   time.Time         `json:"created_at"`
}

type SyncConfig struct {
	ID                 string         `json:"id"`
	Servers            []MCPServer    `json:"servers"`
	LastSyncTime       time.Time      `json:"last_sync_time"`
	LastSyncStatus     string         `json:"last_sync_status"`
	LastUpdateTime     time.Time      `json:"last_update_time"`
	GistID             string         `json:"gist_id"`
	GitHubToken        string         `json:"github_token"`
	AutoSync           bool           `json:"auto_sync"`
	AutoSyncInterval   int           `json:"auto_sync_interval"`
	EnableEncryption   bool          `json:"enable_encryption"`
	// 保留EncryptionPassword字段以兼容旧版本，但不再使用
	EncryptionPassword string       `json:"encryption_password,omitempty"`
	// 新增字段表示加密系统版本
	EncryptionVersion  string         `json:"encryption_version,omitempty"`
}

type SyncLog struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"` // push, pull, conflict
	Status    string    `json:"status"` // success, failed
	Message   string    `json:"message"`
	Details   string    `json:"details"`
}

type ConfigVersion struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
	Source    string    `json:"source"` // local, gist
	Note      string    `json:"note"`
	Hash      string    `json:"hash"`   // SHA256 hash for comparison
}

type SyncConflict struct {
	HasConflict   bool             `json:"has_conflict"`
	ConflictType  string           `json:"conflict_type"` // push_conflict, pull_conflict
	LocalVersion  *ConfigVersion   `json:"local_version"`
	RemoteVersion *ConfigVersion   `json:"remote_version"`
	Message       string           `json:"message"`
}
