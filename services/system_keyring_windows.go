//go:build windows

package services

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/billgraziano/dpapi"
)

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
