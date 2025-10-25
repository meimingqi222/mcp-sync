package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"runtime"
)

// SystemKeyring 提供跨平台的系统密钥存储接口
type SystemKeyring interface {
	// SetKey 存储加密密钥到系统密钥环
	SetKey(service, keyName string, keyData []byte) error
	// GetKey 从系统密钥环获取加密密钥
	GetKey(service, keyName string) ([]byte, error)
	// DeleteKey 从系统密钥环删除加密密钥
	DeleteKey(service, keyName string) error
}

// NewSystemKeyring 创建适合当前平台的系统密钥环实例
func NewSystemKeyring() (SystemKeyring, error) {
	switch runtime.GOOS {
	case "windows":
		return &WindowsKeyring{}, nil
	case "darwin":
		return &MacOSKeyring{}, nil
	case "linux":
		return &LinuxKeyring{}, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// generateRandomKey 生成一个随机的加密密钥
func generateRandomKey() ([]byte, error) {
	key := make([]byte, 32) // 256-bit key for AES-256
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	return key, nil
}

// keyDerivation 从用户密码派生密钥（用于迁移）
func keyDerivation(password, salt []byte) []byte {
	// 使用SHA256作为简单的KDF（实际应用中应使用PBKDF2或Argon2）
	hash := sha256.New()
	hash.Write(password)
	hash.Write(salt)
	return hash.Sum(nil)
}

// WindowsKeyring 使用Windows DPAPI存储密钥
type WindowsKeyring struct{}

func (wk *WindowsKeyring) SetKey(service, keyName string, keyData []byte) error {
	// 在实际的Windows实现中，这将调用DPAPI
	// 这里使用文件存储作为fallback，并建议在实际生产环境中使用DPAPI
	
	dir := os.Getenv("APPDATA")
	if dir == "" {
		return fmt.Errorf("APPDATA environment variable not set")
	}
	
	keyringDir := fmt.Sprintf("%s\\mcp-sync\\keyring", dir)
	if err := os.MkdirAll(keyringDir, 0700); err != nil {
		return fmt.Errorf("failed to create keyring directory: %w", err)
	}
	
	keyFile := fmt.Sprintf("%s\\%s_%s.key", keyringDir, service, keyName)
	
	// 简单地存储密钥（实际应该使用DPAPI加密）
	return os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(keyData)), 0600)
}

func (wk *WindowsKeyring) GetKey(service, keyName string) ([]byte, error) {
	dir := os.Getenv("APPDATA")
	if dir == "" {
		return nil, fmt.Errorf("APPDATA environment variable not set")
	}
	
	keyFile := fmt.Sprintf("%s\\mcp-sync\\keyring\\%s_%s.key", dir, service, keyName)
	
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	
	return base64.StdEncoding.DecodeString(string(data))
}

func (wk *WindowsKeyring) DeleteKey(service, keyName string) error {
	dir := os.Getenv("APPDATA")
	if dir == "" {
		return fmt.Errorf("APPDATA environment variable not set")
	}
	
	keyFile := fmt.Sprintf("%s\\mcp-sync\\keyring\\%s_%s.key", dir, service, keyName)
	
	if err := os.Remove(keyFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete key file: %w", err)
	}
	
	return nil
}

// MacOSKeyring 使用macOS Keychain存储密钥
type MacOSKeyring struct{}

func (mk *MacOSKeyring) SetKey(service, keyName string, keyData []byte) error {
	// 在实际的macOS实现中，这将调用Keychain API
	// 这里使用文件存储作为fallback
	
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	keyringDir := fmt.Sprintf("%s/.local/share/mcp-sync/keyring", home)
	if err := os.MkdirAll(keyringDir, 0700); err != nil {
		return fmt.Errorf("failed to create keyring directory: %w", err)
	}
	
	keyFile := fmt.Sprintf("%s/%s_%s.key", keyringDir, service, keyName)
	
	return os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(keyData)), 0600)
}

func (mk *MacOSKeyring) GetKey(service, keyName string) ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	keyFile := fmt.Sprintf("%s/.local/share/mcp-sync/keyring/%s_%s.key", home, service, keyName)
	
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	
	return base64.StdEncoding.DecodeString(string(data))
}

func (mk *MacOSKeyring) DeleteKey(service, keyName string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	keyFile := fmt.Sprintf("%s/.local/share/mcp-sync/keyring/%s_%s.key", home, service, keyName)
	
	if err := os.Remove(keyFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete key file: %w", err)
	}
	
	return nil
}

// LinuxKeyring 使用Linux密钥环存储密钥
type LinuxKeyring struct{}

func (lk *LinuxKeyring) SetKey(service, keyName string, keyData []byte) error {
	// 在实际的Linux实现中，这将使用libsecret或其他密钥环服务
	// 这里使用文件存储作为fallback
	
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	keyringDir := fmt.Sprintf("%s/.local/share/mcp-sync/keyring", home)
	if err := os.MkdirAll(keyringDir, 0700); err != nil {
		return fmt.Errorf("failed to create keyring directory: %w", err)
	}
	
	keyFile := fmt.Sprintf("%s/%s_%s.key", keyringDir, service, keyName)
	
	return os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(keyData)), 0600)
}

func (lk *LinuxKeyring) GetKey(service, keyName string) ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	keyFile := fmt.Sprintf("%s/.local/share/mcp-sync/keyring/%s_%s.key", home, service, keyName)
	
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	
	return base64.StdEncoding.DecodeString(string(data))
}

func (lk *LinuxKeyring) DeleteKey(service, keyName string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	keyFile := fmt.Sprintf("%s/.local/share/mcp-sync/keyring/%s_%s.key", home, service, keyName)
	
	if err := os.Remove(keyFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete key file: %w", err)
	}
	
	return nil
}
