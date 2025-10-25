package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"runtime"

	"github.com/billgraziano/dpapi"
	"github.com/zalando/go-keyring"
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
	// 使用DPAPI加密密钥数据
	encrypted, err := dpapi.EncryptBytes(keyData)
	if err != nil {
		return fmt.Errorf("failed to encrypt key data with DPAPI: %w", err)
	}

	dir := os.Getenv("APPDATA")
	if dir == "" {
		return fmt.Errorf("APPDATA environment variable not set")
	}

	keyringDir := fmt.Sprintf("%s\\mcp-sync\\keyring", dir)
	if err := os.MkdirAll(keyringDir, 0700); err != nil {
		return fmt.Errorf("failed to create keyring directory: %w", err)
	}

	keyFile := fmt.Sprintf("%s\\%s_%s.key", keyringDir, service, keyName)

	// 存储DPAPI加密后的密钥
	return os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(encrypted)), 0600)
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

	// 解码base64数据
	encrypted, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 key data: %w", err)
	}

	// 使用DPAPI解密
	decrypted, err := dpapi.DecryptBytes(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key data with DPAPI: %w", err)
	}

	return decrypted, nil
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
	// 使用go-keyring包访问macOS Keychain
	// 将二进制数据base64编码为字符串存储
	encodedData := base64.StdEncoding.EncodeToString(keyData)
	keyID := fmt.Sprintf("%s_%s", service, keyName)

	err := keyring.Set(service, keyID, encodedData)
	if err != nil {
		return fmt.Errorf("failed to store key in macOS Keychain: %w", err)
	}

	return nil
}

func (mk *MacOSKeyring) GetKey(service, keyName string) ([]byte, error) {
	// 从macOS Keychain获取密钥
	keyID := fmt.Sprintf("%s_%s", service, keyName)

	encodedData, err := keyring.Get(service, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve key from macOS Keychain: %w", err)
	}

	// base64解码回二进制数据
	keyData, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key data: %w", err)
	}

	return keyData, nil
}

func (mk *MacOSKeyring) DeleteKey(service, keyName string) error {
	// 从macOS Keychain删除密钥
	keyID := fmt.Sprintf("%s_%s", service, keyName)

	err := keyring.Delete(service, keyID)
	if err != nil {
		return fmt.Errorf("failed to delete key from macOS Keychain: %w", err)
	}

	return nil
}

// LinuxKeyring 使用Linux密钥环存储密钥
type LinuxKeyring struct{}

func (lk *LinuxKeyring) SetKey(service, keyName string, keyData []byte) error {
	// 使用go-keyring包访问Linux系统密钥环（如GNOME Keyring、KDE Wallet等）
	// 将二进制数据base64编码为字符串存储
	encodedData := base64.StdEncoding.EncodeToString(keyData)
	keyID := fmt.Sprintf("%s_%s", service, keyName)

	err := keyring.Set(service, keyID, encodedData)
	if err != nil {
		return fmt.Errorf("failed to store key in Linux keyring: %w", err)
	}

	return nil
}

func (lk *LinuxKeyring) GetKey(service, keyName string) ([]byte, error) {
	// 从Linux系统密钥环获取密钥
	keyID := fmt.Sprintf("%s_%s", service, keyName)

	encodedData, err := keyring.Get(service, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve key from Linux keyring: %w", err)
	}

	// base64解码回二进制数据
	keyData, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key data: %w", err)
	}

	return keyData, nil
}

func (lk *LinuxKeyring) DeleteKey(service, keyName string) error {
	// 从Linux系统密钥环删除密钥
	keyID := fmt.Sprintf("%s_%s", service, keyName)

	err := keyring.Delete(service, keyID)
	if err != nil {
		return fmt.Errorf("failed to delete key from Linux keyring: %w", err)
	}

	return nil
}
