package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mcp-sync/models"
	"net/http"
	"time"
)

type GistSyncService struct {
	githubToken       string
	gistID            string
	client            *http.Client
	encryptionEnabled bool
	encryptionKey     string
	securityMgr       *SecurityManager
}

func NewGistSyncService(githubToken, gistID string) *GistSyncService {
	return &GistSyncService{
		githubToken:       githubToken,
		gistID:            gistID,
		client:            &http.Client{Timeout: 10 * time.Second},
		encryptionEnabled: false,
	}
}

// SetEncryption 设置加密参数
func (gs *GistSyncService) SetEncryption(enabled bool, password string) error {
	gs.encryptionEnabled = enabled
	if enabled {
		if password == "" {
			return fmt.Errorf("encryption password cannot be empty")
		}
		gs.encryptionKey = password
		gs.securityMgr = NewSecurityManager(password)
	}
	return nil
}

type GistFile struct {
	Content string `json:"content"`
}

type GistResponse struct {
	ID      string             `json:"id"`
	Files   map[string]GistFile `json:"files"`
	Updated string             `json:"updated_at"`
}

type GistUpdateRequest struct {
	Files map[string]map[string]string `json:"files"`
}

func (gs *GistSyncService) PushToGist(servers []models.MCPServer) error {
	if gs.gistID == "" || gs.githubToken == "" {
		return fmt.Errorf("gist ID or GitHub token not configured")
	}

	// Encryption is mandatory for Gist sync
	if !gs.encryptionEnabled || gs.securityMgr == nil {
		return fmt.Errorf("encryption is required for Gist synchronization. Please set an encryption password")
	}

	// Prepare content
	data := map[string]interface{}{
		"servers": servers,
		"timestamp": time.Now().Format(time.RFC3339),
		"encrypted": true,
	}

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Encrypt configuration
	contentStr := string(content)
	encrypted, err := gs.securityMgr.Encrypt(contentStr)
	if err != nil {
		return fmt.Errorf("failed to encrypt configuration: %w", err)
	}
	contentStr = encrypted
	println("Configuration encrypted before pushing to Gist")

	// Create update request
	updateReq := GistUpdateRequest{
		Files: map[string]map[string]string{
			"mcp-config.json": {
				"content": contentStr,
			},
		},
	}

	reqBody, err := json.Marshal(updateReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.github.com/gists/%s", gs.gistID)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gs.githubToken))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := gs.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("gist update failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func (gs *GistSyncService) PullFromGist() ([]models.MCPServer, error) {
	if gs.gistID == "" || gs.githubToken == "" {
		return nil, fmt.Errorf("gist ID or GitHub token not configured")
	}

	url := fmt.Sprintf("https://api.github.com/gists/%s", gs.gistID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gs.githubToken))
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := gs.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("gist fetch failed: %d - %s", resp.StatusCode, string(body))
	}

	var gistResp GistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gistResp); err != nil {
		return nil, err
	}

	// Parse mcp-config.json
	configFile, exists := gistResp.Files["mcp-config.json"]
	if !exists {
		return nil, fmt.Errorf("mcp-config.json not found in gist")
	}

	contentStr := configFile.Content

	// Try to decrypt if encryption is detected
	var dataMap map[string]interface{}
	err = json.Unmarshal([]byte(contentStr), &dataMap)
	if err != nil && gs.encryptionEnabled && gs.securityMgr != nil {
		// Content is likely encrypted, try to decrypt
		decrypted, err := gs.securityMgr.Decrypt(contentStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt configuration: %w (check encryption password)", err)
		}
		contentStr = decrypted
		println("Configuration decrypted after pulling from Gist")
	}

	var data struct {
		Servers   []models.MCPServer `json:"servers"`
		Encrypted bool               `json:"encrypted"`
	}

	if err := json.Unmarshal([]byte(contentStr), &data); err != nil {
		return nil, err
	}

	return data.Servers, nil
}

func (gs *GistSyncService) CreateGist(servers []models.MCPServer, description string) (string, error) {
	if gs.githubToken == "" {
		return "", fmt.Errorf("GitHub token not configured")
	}

	content, err := json.MarshalIndent(map[string]interface{}{
		"servers": servers,
		"timestamp": time.Now().Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return "", err
	}

	createReq := map[string]interface{}{
		"description": description,
		"public":      false,
		"files": map[string]map[string]string{
			"mcp-config.json": {
				"content": string(content),
			},
		},
	}

	reqBody, err := json.Marshal(createReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.github.com/gists", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gs.githubToken))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := gs.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("gist creation failed: %d - %s", resp.StatusCode, string(body))
	}

	var gistResp GistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gistResp); err != nil {
		return "", err
	}

	return gistResp.ID, nil
}

func (gs *GistSyncService) ValidateToken() error {
	if gs.githubToken == "" {
		return fmt.Errorf("GitHub token not configured")
	}

	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gs.githubToken))
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := gs.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid GitHub token")
	}

	return nil
}

// PushAgentConfigsToGist 推送完整的 agent 配置到 Gist（保留完整信息）
func (gs *GistSyncService) PushAgentConfigsToGist(agentConfigs map[string]interface{}) error {
	if gs.gistID == "" || gs.githubToken == "" {
		return fmt.Errorf("gist ID or GitHub token not configured")
	}

	// Encryption is mandatory for Gist sync
	if !gs.encryptionEnabled || gs.securityMgr == nil {
		return fmt.Errorf("encryption is required for Gist synchronization. Please set an encryption password")
	}

	// Prepare content
	data := map[string]interface{}{
		"agents":    agentConfigs,
		"timestamp": time.Now().Format(time.RFC3339),
		"encrypted": true,
	}

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Encrypt configuration
	contentStr := string(content)
	encrypted, err := gs.securityMgr.Encrypt(contentStr)
	if err != nil {
		return fmt.Errorf("failed to encrypt configuration: %w", err)
	}
	contentStr = encrypted
	println("Complete agent configurations encrypted before pushing to Gist")

	// Create update request
	updateReq := GistUpdateRequest{
		Files: map[string]map[string]string{
			"mcp-config.json": {
				"content": contentStr,
			},
		},
	}

	reqBody, err := json.Marshal(updateReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.github.com/gists/%s", gs.gistID)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gs.githubToken))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := gs.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("gist update failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// PullAgentConfigsFromGist 从 Gist 拉取完整的 agent 配置（保留完整信息）
func (gs *GistSyncService) PullAgentConfigsFromGist() (map[string]interface{}, error) {
	if gs.gistID == "" || gs.githubToken == "" {
		return nil, fmt.Errorf("gist ID or GitHub token not configured")
	}

	url := fmt.Sprintf("https://api.github.com/gists/%s", gs.gistID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gs.githubToken))
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := gs.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("gist fetch failed: %d - %s", resp.StatusCode, string(body))
	}

	var gistResp GistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gistResp); err != nil {
		return nil, err
	}

	// Parse mcp-config.json
	configFile, exists := gistResp.Files["mcp-config.json"]
	if !exists {
		return nil, fmt.Errorf("mcp-config.json not found in gist")
	}

	contentStr := configFile.Content

	// Try to decrypt if encryption is detected
	var dataMap map[string]interface{}
	err = json.Unmarshal([]byte(contentStr), &dataMap)
	if err != nil && gs.encryptionEnabled && gs.securityMgr != nil {
		// Content is likely encrypted, try to decrypt
		decrypted, err := gs.securityMgr.Decrypt(contentStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt configuration: %w (check encryption password)", err)
		}
		contentStr = decrypted
		println("Complete agent configurations decrypted after pulling from Gist")
	}

	var data struct {
		Agents    map[string]interface{} `json:"agents"`
		Encrypted bool                   `json:"encrypted"`
	}

	if err := json.Unmarshal([]byte(contentStr), &data); err != nil {
		return nil, err
	}

	if data.Agents == nil {
		return make(map[string]interface{}), nil
	}

	return data.Agents, nil
}

// GetLatestVersion 从 Gist 获取最新的配置版本
func (gs *GistSyncService) GetLatestVersion() (*models.ConfigVersion, error) {
	if gs.gistID == "" || gs.githubToken == "" {
		return nil, fmt.Errorf("gist ID or GitHub token not configured")
	}

	url := fmt.Sprintf("https://api.github.com/gists/%s", gs.gistID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gs.githubToken))
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := gs.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("gist not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("gist fetch failed: %d - %s", resp.StatusCode, string(body))
	}

	var gistResp GistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gistResp); err != nil {
		return nil, err
	}

	// Parse mcp-config.json
	configFile, exists := gistResp.Files["mcp-config.json"]
	if !exists {
		return nil, fmt.Errorf("mcp-config.json not found in gist")
	}

	contentStr := configFile.Content

	// Try to decrypt if encryption is enabled
	var dataMap map[string]interface{}
	err = json.Unmarshal([]byte(contentStr), &dataMap)
	if err != nil && gs.encryptionEnabled && gs.securityMgr != nil {
		// Content is likely encrypted, try to decrypt
		decrypted, err := gs.securityMgr.Decrypt(contentStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt configuration: %w", err)
		}
		contentStr = decrypted
	}

	// Parse timestamp from content
	var data struct {
		Timestamp string `json:"timestamp"`
	}
	json.Unmarshal([]byte(contentStr), &data)

	timestamp := time.Now()
	if data.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, data.Timestamp); err == nil {
			timestamp = t
		}
	}

	// Calculate hash of content
	hash := sha256.Sum256([]byte(contentStr))
	hashStr := hex.EncodeToString(hash[:])

	return &models.ConfigVersion{
		ID:        gistResp.ID,
		Timestamp: timestamp,
		Content:   contentStr,
		Source:    "gist",
		Note:      "Latest version from Gist",
		Hash:      hashStr,
	}, nil
}
