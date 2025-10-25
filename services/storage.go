package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mcp-sync/models"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type StorageService struct {
	dataDir        string
	securityMgr    *SecurityManager
	encryptionKey  string // Derive from encryption password
}

func NewStorageService(dataDir string) (*StorageService, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	return &StorageService{
		dataDir: dataDir,
	}, nil
}

// EnableEncryption enables encryption for the storage service
func (s *StorageService) EnableEncryption(password string) {
	s.encryptionKey = password
	if password != "" {
		s.securityMgr = NewSecurityManager(password)
	}
}

// isEncrypted checks if a file is already encrypted (starts with ENC: marker)
func (s *StorageService) isEncrypted(data []byte) bool {
	return strings.HasPrefix(string(data), "ENC:")
}

// encryptIfNeeded encrypts data if encryption is enabled
func (s *StorageService) encryptIfNeeded(data []byte) ([]byte, error) {
	if s.securityMgr == nil || s.encryptionKey == "" {
		return data, nil
	}

	// Check if already encrypted
	if s.isEncrypted(data) {
		return data, nil
	}

	// Encrypt the data
	encrypted, err := s.securityMgr.Encrypt(string(data))
	if err != nil {
		return nil, err
	}

	// Add encryption marker
	return []byte("ENC:" + encrypted), nil
}

// decryptIfNeeded decrypts data if it's encrypted
func (s *StorageService) decryptIfNeeded(data []byte) ([]byte, error) {
	if !s.isEncrypted(data) {
		return data, nil
	}

	if s.securityMgr == nil || s.encryptionKey == "" {
		return nil, fmt.Errorf("file is encrypted but no encryption key provided")
	}

	// Remove encryption marker
	encryptedData := strings.TrimPrefix(string(data), "ENC:")

	// Decrypt
	decrypted, err := s.securityMgr.Decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt file (check encryption password): %w", err)
	}

	return []byte(decrypted), nil
}

func (s *StorageService) SaveSyncConfig(config models.SyncConfig) error {
	path := filepath.Join(s.dataDir, "sync_config.json")

	// Ensure directory exists before saving
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Encrypt if encryption is enabled
	data, err = s.encryptIfNeeded(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt configuration: %w", err)
	}

	return ioutil.WriteFile(path, data, 0644)
}

func (s *StorageService) LoadSyncConfig() (models.SyncConfig, error) {
	path := filepath.Join(s.dataDir, "sync_config.json")

	var config models.SyncConfig

	if !fileExists(path) {
		// Return default config
		config.ID = "default"
		config.Servers = []models.MCPServer{}
		config.LastSyncTime = time.Now()
		config.AutoSync = false
		config.AutoSyncInterval = 3600 // 1 hour
		return config, nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return config, err
	}

	// Decrypt if needed
	data, err = s.decryptIfNeeded(data)
	if err != nil {
		return config, fmt.Errorf("failed to load config: %w", err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}

	// Auto-enable encryption if configured but not yet enabled
	if config.EnableEncryption && config.EncryptionPassword != "" && s.encryptionKey == "" {
		println("Auto-enabling local storage encryption")
		s.EnableEncryption(config.EncryptionPassword)

		// Re-encrypt the file if it's not already encrypted
		data, _ := json.MarshalIndent(config, "", "  ")
		data, _ = s.encryptIfNeeded(data)
		ioutil.WriteFile(path, data, 0644)
	}

	return config, nil
}

func (s *StorageService) SaveConfigVersion(version models.ConfigVersion) error {
	dir := filepath.Join(s.dataDir, "versions")

	// Ensure directory exists before saving
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	filename := fmt.Sprintf("version_%d.json", time.Now().Unix())
	path := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		return err
	}

	// Encrypt if encryption is enabled
	data, err = s.encryptIfNeeded(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt version: %w", err)
	}

	return ioutil.WriteFile(path, data, 0644)
}

func (s *StorageService) ListConfigVersions(limit int) ([]models.ConfigVersion, error) {
	dir := filepath.Join(s.dataDir, "versions")

	if !fileExists(dir) {
		return []models.ConfigVersion{}, nil
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var versions []models.ConfigVersion

	// Read files in reverse order (newest first)
	for i := len(files) - 1; i >= 0 && len(versions) < limit; i-- {
		if files[i].IsDir() {
			continue
		}

		path := filepath.Join(dir, files[i].Name())
		data, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}

		// Decrypt if needed
		data, err = s.decryptIfNeeded(data)
		if err != nil {
			// Skip files that can't be decrypted
			continue
		}

		var version models.ConfigVersion
		if err := json.Unmarshal(data, &version); err != nil {
			continue
		}

		versions = append(versions, version)
	}

	return versions, nil
}

func (s *StorageService) SaveSyncLog(log models.SyncLog) error {
	dir := filepath.Join(s.dataDir, "logs")

	// Ensure directory exists before saving
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	filename := fmt.Sprintf("sync_%d.json", time.Now().Unix())
	path := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return err
	}

	// Encrypt if encryption is enabled
	data, err = s.encryptIfNeeded(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt log: %w", err)
	}

	return ioutil.WriteFile(path, data, 0644)
}

func (s *StorageService) GetSyncLogs(limit int) ([]models.SyncLog, error) {
	dir := filepath.Join(s.dataDir, "logs")

	if !fileExists(dir) {
		return []models.SyncLog{}, nil
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var logs []models.SyncLog

	// Read files in reverse order (newest first)
	for i := len(files) - 1; i >= 0 && len(logs) < limit; i-- {
		if files[i].IsDir() {
			continue
		}

		path := filepath.Join(dir, files[i].Name())
		data, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}

		// Decrypt if needed
		data, err = s.decryptIfNeeded(data)
		if err != nil {
			// Skip files that can't be decrypted
			continue
		}

		var log models.SyncLog
		if err := json.Unmarshal(data, &log); err != nil {
			continue
		}

		logs = append(logs, log)
	}

	return logs, nil
}

func (s *StorageService) GetDataDir() string {
	return s.dataDir
}
