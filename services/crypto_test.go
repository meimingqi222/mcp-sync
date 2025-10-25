package services

import (
	"testing"
)

func TestSecureCryptoFixed(t *testing.T) {
	// 清理之前的测试状态
	keyring, _ := NewSystemKeyring()
	if keyring != nil {
		_ = keyring.DeleteKey("mcp-sync", "master_key")
	}

	tests := []struct {
		name    string
		plain   string
		wantErr bool
	}{
		{
			name:    "basic encryption test",
			plain:   "Hello, World!",
			wantErr: false,
		},
		{
			name:    "empty string",
			plain:   "",
			wantErr: false,
		},
		{
			name:    "json data",
			plain:   `{"name":"test","servers":[{"id":"server1","command":"echo","args":["hello"]}]}`,
			wantErr: false,
		},
	}

	// 创建安全加密实例
	crypto, err := NewSecureCrypto()
	if err != nil {
		t.Fatalf("Failed to create secure crypto: %v", err)
	}

	// 确保加密被禁用开始时
	if crypto.IsEnabled() {
		t.Errorf("Expected encryption to be disabled initially")
	}

	// 启用加密
	err = crypto.Enable()
	if err != nil {
		t.Fatalf("Failed to enable encryption: %v", err)
	}

	// 检查加密是否启用
	if !crypto.IsEnabled() {
		t.Errorf("Expected encryption to be enabled after calling Enable()")
	}

	// 测试加密和解密
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 加密
			encrypted, err := crypto.Encrypt(tt.plain)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && encrypted == "" {
				t.Errorf("Encrypt() returned empty string")
				return
			}

			// 确保加密后的文本与原文不同
			if !tt.wantErr && encrypted == tt.plain {
				t.Errorf("Encrypted data should be different from plaintext")
			}

			// 解密
			decrypted, err := crypto.Decrypt(encrypted)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && decrypted != tt.plain {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plain)
			}
		})
	}

	// 清理测试数据
	crypto.Disable()
}

func TestSystemKeyringFixed(t *testing.T) {
	keyring, err := NewSystemKeyring()
	if err != nil {
		t.Skipf("System keyring not available for testing: %v", err)
		return
	}

	service := "test-mcp-sync-fixed"
	keyName := "test-key-fixed"
	keyData := []byte("test-data-12345")

	// 清理之前的测试数据
	_ = keyring.DeleteKey(service, keyName)

	// 测试存储密钥
	err = keyring.SetKey(service, keyName, keyData)
	if err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}

	// 测试获取密钥
	retrievedKey, err := keyring.GetKey(service, keyName)
	if err != nil {
		t.Fatalf("Failed to retrieve key: %v", err)
	}

	if string(retrievedKey) != string(keyData) {
		t.Errorf("Retrieved key doesn't match stored key")
	}

	// 测试删除密钥
	err = keyring.DeleteKey(service, keyName)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	// 验证密钥已删除
	_, err = keyring.GetKey(service, keyName)
	if err == nil {
		t.Error("Expected error when retrieving deleted key")
	}
}

func BenchmarkSecureCryptoEncryptDecrypt(b *testing.B) {
	crypto, err := NewSecureCrypto()
	if err != nil {
		b.Fatalf("Failed to create secure crypto: %v", err)
	}

	err = crypto.Enable()
	if err != nil {
		b.Fatalf("Failed to enable encryption: %v", err)
	}

	plainText := "This is a benchmark test for secure encryption and decryption operations."

	b.ResetTimer()
	
	// 加密基准测试
	b.Run("Encrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := crypto.Encrypt(plainText)
			if err != nil {
				b.Fatalf("Encryption failed: %v", err)
			}
		}
	})

	// 解密基准测试
	encrypted, err := crypto.Encrypt(plainText)
	if err != nil {
		b.Fatalf("Failed to encrypt for decryption benchmark: %v", err)
	}

	b.Run("Decrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := crypto.Decrypt(encrypted)
			if err != nil {
				b.Fatalf("Decryption failed: %v", err)
			}
		}
	})
}
