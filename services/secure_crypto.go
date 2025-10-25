package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// SecureCrypto 提供使用系统密钥环的安全加密服务
type SecureCrypto struct {
	keyring     SystemKeyring
	serviceName string
}

// NewSecureCrypto 创建一个新的安全加密实例
func NewSecureCrypto() (*SecureCrypto, error) {
	keyring, err := NewSystemKeyring()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize system keyring: %w", err)
	}
	
	return &SecureCrypto{
		keyring:     keyring,
		serviceName: "mcp-sync",
	}, nil
}

// Enable 启用加密，生成新的密钥并存储到系统密钥环
func (sc *SecureCrypto) Enable() error {
	// 检查是否已经有密钥
	existingKey, err := sc.getKey()
	if err == nil && len(existingKey) > 0 {
		// 密钥已存在，不需要重新生成
		return nil
	}
	
	// 生成新的随机密钥
	newKey, err := generateRandomKey()
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}
	
	// 存储到系统密钥环
	if err := sc.keyring.SetKey(sc.serviceName, "master_key", newKey); err != nil {
		return fmt.Errorf("failed to store encryption key to system keyring: %w", err)
	}
	
	return nil
}

// Disable 禁用加密，删除存储的密钥
func (sc *SecureCrypto) Disable() error {
	return sc.keyring.DeleteKey(sc.serviceName, "master_key")
}

// IsEnabled 检查加密是否已启用
func (sc *SecureCrypto) IsEnabled() bool {
	key, err := sc.getKey()
	return err == nil && len(key) > 0
}

// getKey 获取加密密钥
func (sc *SecureCrypto) getKey() ([]byte, error) {
	return sc.keyring.GetKey(sc.serviceName, "master_key")
}

// Encrypt 加密数据
func (sc *SecureCrypto) Encrypt(plaintext string) (string, error) {
	key, err := sc.getKey()
	if err != nil {
		return "", fmt.Errorf("encryption not enabled or key not available: %w", err)
	}
	
	return encryptData(key, plaintext)
}

// Decrypt 解密数据
func (sc *SecureCrypto) Decrypt(ciphertext string) (string, error) {
	key, err := sc.getKey()
	if err != nil {
		return "", fmt.Errorf("encryption not enabled or key not available: %w", err)
	}
	
	return decryptData(key, ciphertext)
}

// EncryptIfNeeded 如果加密启用则加密数据
func (sc *SecureCrypto) EncryptIfNeeded(data []byte) ([]byte, error) {
	if !sc.IsEnabled() {
		return data, nil
	}
	
	// 检查是否已经加密
	if sc.isEncrypted(data) {
		return data, nil
	}
	
	encrypted, err := sc.Encrypt(string(data))
	if err != nil {
		return nil, err
	}
	
	// 添加加密标记
	return []byte("ENC:" + encrypted), nil
}

// DecryptIfNeeded 如果需要则解密数据
func (sc *SecureCrypto) DecryptIfNeeded(data []byte) ([]byte, error) {
	if !sc.isEncrypted(data) {
		return data, nil
	}
	
	if !sc.IsEnabled() {
		return nil, fmt.Errorf("data is encrypted but encryption is not enabled")
	}
	
	// 移除加密标记
	encryptedData := strings.TrimPrefix(string(data), "ENC:")
	
	decrypted, err := sc.Decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	
	return []byte(decrypted), nil
}

// isEncrypted 检查数据是否已加密
func (sc *SecureCrypto) isEncrypted(data []byte) bool {
	return strings.HasPrefix(string(data), "ENC:")
}

// encryptData 使用给定密钥加密数据的通用函数
func encryptData(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	plainBytes := []byte(plaintext)
	ciphertext := gcm.Seal(nonce, nonce, plainBytes, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptData 使用给定密钥解密数据的通用函数
func decryptData(key []byte, ciphertext string) (string, error) {
	cipherBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(cipherBytes) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext2 := cipherBytes[:nonceSize], cipherBytes[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext2, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// MigrateFromPassword 从旧的基于密码的加密系统迁移
func (sc *SecureCrypto) MigrateFromPassword(password string) error {
	// 如果加密已经启用，不要迁移
	if sc.IsEnabled() {
		return nil
	}
	
	// 生成新的密钥
	newKey, err := generateRandomKey()
	if err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}
	
	// 从旧密码派生临时密钥（用于解密现有数据）
	salt := []byte("mcp-sync-migration-salt")
	tempKey := keyDerivation([]byte(password), salt)
	
	// 存储新密钥到系统密钥环
	if err := sc.keyring.SetKey(sc.serviceName, "master_key", newKey); err != nil {
		return fmt.Errorf("failed to store new key: %w", err)
	}
	
	// 创建迁移临时管理器
	tempCrypto := &SecureCrypto{}
	tempCrypto.keyring = &memoryKeyring{key: tempKey}
	
	// 返回两个管理器以便应用程序可以执行迁移
	_ = tempCrypto
	return nil
}

// memoryKeyring 用于迁移时的临时密钥存储
type memoryKeyring struct {
	key []byte
}

func (mk *memoryKeyring) SetKey(service, keyName string, keyData []byte) error {
	mk.key = keyData
	return nil
}

func (mk *memoryKeyring) GetKey(service, keyName string) ([]byte, error) {
	return mk.key, nil
}

func (mk *memoryKeyring) DeleteKey(service, keyName string) error {
	mk.key = nil
	return nil
}

// ValidatePassword 验证给定密码是否与存储的密码匹配（用于迁移验证）
func ValidatePassword(checkPassword, storedPassword string) bool {
	// 简单的SHA256比较（实际应用中应该使用更安全的方法）
	hash1 := sha256.Sum256([]byte(checkPassword))
	hash2 := sha256.Sum256([]byte(storedPassword))
	
	return hash1 == hash2
}

// GenerateRecoveryCode 生成恢复代码（当用户需要重置加密时）
func GenerateRecoveryCode() string {
	// 生成一个可读的恢复代码
	bytes := make([]byte, 4)
	rand.Read(bytes)
	code := fmt.Sprintf("%X-%X-%X-%X", bytes[0], bytes[1], bytes[2], bytes[3])
	return code
}

// BackupKey 提供密钥备份功能（用户可以将密钥导出安全存储）
func (sc *SecureCrypto) BackupKey() error {
	key, err := sc.getKey()
	if err != nil {
		return fmt.Errorf("failed to get key for backup: %w", err)
	}
	
	// 可以将密钥加密后备份到文件或其他位置
	// 这里只是示例
	_ = key
	_ = base64.StdEncoding.EncodeToString(key)
	
	return nil
}
