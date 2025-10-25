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
	dataDir string
	crypto  *SecureCrypto
	// 保留旧的securityMgr以兼容现有代码（将在下个版本移除）
	securityMgr *SecurityManager
	oldEnabled  bool
}

func NewStorageService(dataDir string) (*StorageService, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	// 初始化新的安全加密服务
	crypto, err := NewSecureCrypto()
	if err != nil {
		// 如果系统密钥环不可用，仍然返回服务但加密功能将被禁用
		fmt.Printf("Warning: failed to initialize secure crypto: %v\n", err)
	}

	return &StorageService{
		dataDir: dataDir,
		crypto:  crypto,
	}, nil
}

// EnableEncryption enables encryption for the storage service
// 注意：新版本不再需要密码参数，使用系统密钥环
func (s *StorageService) EnableEncryption(password string) {
	// 如果提供了密码，说明是从旧版本迁移
	if password != "" && s.crypto != nil {
		// 尝试从密码迁移到新系统
		if err := s.crypto.MigrateFromPassword(password); err != nil {
			fmt.Printf("Warning: failed to migrate from password encryption: %v\n", err)
			// 作为fallback，仍然使用旧方式
			s.securityMgr = NewSecurityManager(password)
			s.oldEnabled = true
		} else {
			// 迁移成功，启用新系统
			if err := s.crypto.Enable(); err != nil {
				fmt.Printf("Warning: failed to enable secure crypto: %v\n", err)
			}
		}
	} else if s.crypto != nil {
		// 直接启用新系统
		if err := s.crypto.Enable(); err != nil {
			fmt.Printf("Warning: failed to enable secure crypto: %v\n", err)
		}
	}
}

// DisableEncryption disables encryption for the storage service
func (s *StorageService) DisableEncryption() error {
	if s.crypto != nil && s.crypto.IsEnabled() {
		if err := s.crypto.Disable(); err != nil {
			return fmt.Errorf("failed to disable encryption: %w", err)
		}
	}

	// 清理旧的加密组件
	if s.securityMgr != nil {
		s.securityMgr = nil
	}
	s.oldEnabled = false

	return nil
}

// IsEncryptionEnabled checks if encryption is currently enabled
func (s *StorageService) IsEncryptionEnabled() bool {
	if s.crypto != nil && s.crypto.IsEnabled() {
		return true
	}
	return s.oldEnabled
}

// isEncrypted checks if a file is already encrypted (starts with ENC: marker)
func (s *StorageService) isEncrypted(data []byte) bool {
	return strings.HasPrefix(string(data), "ENC:")
}

// encryptIfNeeded encrypts data if encryption is enabled
func (s *StorageService) encryptIfNeeded(data []byte) ([]byte, error) {
	// 首先尝试使用新的安全加密系统
	if s.crypto != nil && s.crypto.IsEnabled() {
		return s.crypto.EncryptIfNeeded(data)
	}

	// 兼容旧的加密系统
	if s.securityMgr != nil && s.oldEnabled {
		return s.encryptIfNeededOld(data)
	}

	// 如果没有启用加密，直接返回原数据
	return data, nil
}

// encryptIfNeededOld 兼容旧的加密方法
func (s *StorageService) encryptIfNeededOld(data []byte) ([]byte, error) {
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
	// 首先尝试使用新的安全加密系统
	if s.crypto != nil && s.crypto.IsEnabled() {
		result, err := s.crypto.DecryptIfNeeded(data)
		if err == nil {
			return result, nil
		}
		// 如果新系统解密失败，可能是旧版本的数据，继续尝试旧系统
	}

	// 兼容旧的加密系统
	if s.securityMgr != nil && s.oldEnabled {
		return s.decryptIfNeededOld(data)
	}

	// 如果数据未加密，直接返回
	if !s.isEncrypted(data) {
		return data, nil
	}

	// 如果数据已加密但没有可用的解密系统
	return nil, fmt.Errorf("file is encrypted but no decryption key available")
}

// decryptIfNeededOld 兼容旧的解密方法
func (s *StorageService) decryptIfNeededOld(data []byte) ([]byte, error) {
	if !s.isEncrypted(data) {
		return data, nil
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
	if config.EnableEncryption && !s.IsEncryptionEnabled() {
		println("Auto-enabling local storage encryption")
		s.EnableEncryption("") // 新版本不需要密码

		// Re-encrypt the file if it's not already encrypted
		data, _ := json.MarshalIndent(config, "", "  ")
		data, _ = s.encryptIfNeeded(data)
		ioutil.WriteFile(path, data, 0644)
	}

	// 处理密码迁移逻辑
	if config.EncryptionPassword != "" && config.GistEncryptionPassword == "" {
		// 如果有旧密码字段但没有新字段，说明需要迁移
		config.GistEncryptionPassword = config.EncryptionPassword

		// 标记已迁移，但保留旧字段以防回滚需要
		config.EncryptionVersion = "2.0"

		// 保存更新后的配置（包含新的密码字段）
		configData, _ := json.MarshalIndent(config, "", "  ")
		configData, _ = s.encryptIfNeeded(configData)
		ioutil.WriteFile(path, configData, 0644)
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
