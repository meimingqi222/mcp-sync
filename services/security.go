package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

type SecurityManager struct {
	encryptionKey string
}

func NewSecurityManager(key string) *SecurityManager {
	return &SecurityManager{
		encryptionKey: padKey(key),
	}
}

// 敏感字段的键名模式
var sensitivePatterns = []string{
	"api_key",
	"apikey",
	"token",
	"secret",
	"password",
	"passwd",
	"key",
	"auth",
}

// IsSensitiveField 检查字段是否包含敏感信息
func IsSensitiveField(fieldName string) bool {
	lowerName := strings.ToLower(fieldName)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerName, pattern) {
			return true
		}
	}
	return false
}

// MaskSensitiveValue 掩码敏感值
func MaskSensitiveValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

// FilterSensitiveData 从配置中移除或掩码敏感数据
func FilterSensitiveData(servers []interface{}) []interface{} {
	result := make([]interface{}, 0)

	for _, server := range servers {
		if serverMap, ok := server.(map[string]interface{}); ok {
			filtered := make(map[string]interface{})

			for key, value := range serverMap {
				if IsSensitiveField(key) {
					// 对敏感字段进行掩码处理
					if strVal, ok := value.(string); ok {
						filtered[key] = MaskSensitiveValue(strVal)
					} else {
						filtered[key] = "****"
					}
				} else if env, ok := value.(map[string]interface{}); ok {
					// 处理环境变量字段
					filteredEnv := make(map[string]interface{})
					for envKey, envVal := range env {
						if IsSensitiveField(envKey) {
							if strVal, ok := envVal.(string); ok {
								filteredEnv[envKey] = MaskSensitiveValue(strVal)
							} else {
								filteredEnv[envKey] = "****"
							}
						} else {
							filteredEnv[envKey] = envVal
						}
					}
					filtered[key] = filteredEnv
				} else {
					filtered[key] = value
				}
			}
			result = append(result, filtered)
		}
	}

	return result
}

// Encrypt 加密字符串
func (sm *SecurityManager) Encrypt(plaintext string) (string, error) {
	key := []byte(sm.encryptionKey)
	plainBytes := []byte(plaintext)

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

	ciphertext := gcm.Seal(nonce, nonce, plainBytes, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密字符串
func (sm *SecurityManager) Decrypt(ciphertext string) (string, error) {
	key := []byte(sm.encryptionKey)

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

// padKey 将密钥补充到 32 字节（AES-256）
func padKey(key string) string {
	if len(key) > 32 {
		return key[:32]
	}
	for len(key) < 32 {
		key += key
	}
	return key[:32]
}

// ValidateGistSecurity 验证 Gist 安全性
type GistSecurityWarning struct {
	Level       string // "warning", "error"
	Title       string
	Description string
	Suggestion  string
}

func GetGistSecurityWarnings() []GistSecurityWarning {
	return []GistSecurityWarning{
		{
			Level: "warning",
			Title: "Gist Visibility",
			Description: "Secret Gists (not listed) can still be accessed by anyone with the URL. " +
				"They are not completely private unless using GitHub's private repositories.",
			Suggestion: "Consider storing sensitive credentials outside of Gist or use GitHub's encrypted secrets.",
		},
		{
			Level: "warning",
			Title: "Token Security",
			Description: "Your GitHub token has access to your Gist. If the token is compromised, " +
				"someone could modify your configurations.",
			Suggestion: "Use GitHub Personal Access Tokens with limited scopes. Rotate your token regularly.",
		},
		{
			Level: "warning",
			Title: "Sensitive Data",
			Description: "MCP configurations may contain API keys, tokens, or other sensitive information " +
				"that should not be stored in plain text.",
			Suggestion: "Never commit credentials to Gist. Use environment variables or encrypted storage instead.",
		},
	}
}

// SanitizeConfig 清理配置中的敏感信息
func SanitizeConfig(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		if IsSensitiveField(key) {
			// 对顶级敏感字段进行掩码
			if strVal, ok := value.(string); ok {
				result[key] = MaskSensitiveValue(strVal)
			} else {
				result[key] = "****"
			}
		} else if envMap, ok := value.(map[string]interface{}); ok {
			// 递归处理嵌套的对象（如 env）
			result[key] = SanitizeConfig(envMap)
		} else if arrayVal, ok := value.([]interface{}); ok {
			// 处理数组
			result[key] = arrayVal
		} else {
			result[key] = value
		}
	}

	return result
}
