# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Common Development Commands

### Development
```bash
# Start development server with hot reload (frontend + backend)
wails dev

# Alternative: Use PowerShell script (Windows)
.\run-dev.ps1
```

### Building
```bash
# Build production executable (Windows)
wails build -clean

# Build for specific platform
wails build -platform windows/amd64
wails build -platform darwin/universal
wails build -platform linux/amd64

# Quick build script (Windows)
.\build.bat
```

### Testing
```bash
# Run Go tests
go test ./...

# Run specific service tests
go test ./services/...

# Test with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Dependencies
```bash
# Update Go dependencies
go mod tidy
go mod download

# Update frontend dependencies (use pnpm, not npm)
cd frontend
pnpm install
pnpm update
```

## Project Architecture

### Core Components

**MCP Sync** is a cross-platform desktop application built with **Wails v2** (Go + React) that synchronizes Model Context Protocol (MCP) configurations across multiple AI coding tools.

### Configuration-Driven Agent System

The entire agent detection and configuration system is **YAML-driven** via `agents.yaml` (also embedded via `services/agents.yaml`):

- **Agent Definitions**: Each AI tool (Claude, Cursor, Zed, etc.) is defined in YAML with platform-specific config paths
- **Format Transforms**: Conversion rules between different config formats (e.g., `standard` ↔ `zed`) are defined in YAML
- **No Hard-Coding**: To add a new tool, edit `agents.yaml` - no Go code changes needed

**Key files:**
- `agents.yaml` / `services/agents.yaml`: Single source of truth for all agent definitions and format transforms
- `services/config_loader.go`: Loads and parses YAML, expands paths (`~`, `$APPDATA`, etc.)
- `services/detector.go`: Uses ConfigLoader to detect installed agents dynamically

### Format Conversion System

Different tools use different MCP config formats. The system handles this via:

**Format Types:**
- `standard`: Uses `mcpServers` key (Claude, Cursor, Windsurf, Qwen, Cline, etc.)
- `zed`: Uses `context_servers` key with additional fields (`source`, `enabled`)

**Transform Rules** (in `agents.yaml`):
```yaml
transforms:
  standard_to_zed:
    add_fields:
      source: "custom"
      enabled: true
    keep_fields: [command, args, env]
  
  zed_to_standard:
    remove_fields: [source, enabled]
    keep_fields: [command, args, env]
```

**Implementation:**
- `services/config_loader.go`: `GetTransformRule()`, `ApplyTransformRule()`
- `services/config_manager.go`: Applies transforms during sync operations

### Windows NPX Handling

Windows requires `npx` commands to be wrapped as `cmd /c npx`. The system handles this automatically:

- **Detection**: `services/windows_service.go` checks if command starts with `npx`
- **Wrapping**: `WrapNpxCommand()` converts `npx @modelcontextprotocol/...` → `["cmd", "/c", "npx", "@modelcontextprotocol/..."]`
- **Unwrapping**: Reverse operation for cross-platform compatibility

### Service Layer Architecture

**App Service Flow** (`services/app_service.go`):
```
AppService (orchestration)
  ├─> AgentDetector (scan installed tools)
  ├─> ConfigManager (read/write config files)
  ├─> ConfigLoader (parse agents.yaml)
  ├─> GistSyncService (GitHub Gist sync)
  ├─> StorageService (local persistence)
  ├─> SecurityManager (encryption)
  └─> WindowsService (platform-specific)
```

**Key Service Methods:**
- `DetectAgents()`: Scans system for installed AI tools
- `GetAgentMCPConfig(agentID)`: Reads config from specific tool
- `SaveAgentMCPConfig(agentID, config)`: Writes config to tool
- `SyncConfigBetweenAgents(sourceID, targetID)`: Syncs with format conversion
- `PushAllAgentsToGist()`: Backup all configs to cloud
- `PullFromGist()`: Restore configs from cloud

### Data Models

**Core types** (`models/types.go`):
- `Agent`: Represents an AI tool (ID, name, config paths, status)
- `MCPServer`: MCP service definition (command, args, env)
- `SyncConfig`: App settings (GitHub token, Gist ID, encryption)
- `ConfigVersion`: Version history entry
- `SyncLog`: Sync operation log

### Encryption System

Two-layer encryption approach:

**Local Storage** (system keyring):
- Uses OS keyring: Windows Credential Manager / macOS Keychain / Linux Secret Service
- `services/system_keyring.go`: Cross-platform keyring interface
- `services/system_keyring_windows.go`: Windows DPAPI implementation
- `services/secure_crypto.go`: AES-256-GCM encryption

**Gist Sync** (user password):
- Optional password-based encryption for cloud backups
- User sets password in settings
- `services/gist_sync.go`: Encrypts before push, decrypts after pull

### Frontend-Backend Communication

Wails binds Go methods to JavaScript:

**Exposed Methods** (`app.go`):
```go
func (a *App) DetectAgents() ([]models.Agent, error)
func (a *App) GetAgentMCPConfig(agentID string) (map[string]interface{}, error)
func (a *App) SyncConfigBetweenAgents(sourceAgentID, targetAgentID string) error
func (a *App) PushAllAgentsToGist() error
func (a *App) PullFromGist() ([]models.MCPServer, error)
// ... more in app.go
```

**Frontend Usage** (`frontend/src/types/wails.ts`):
```typescript
import { DetectAgents, GetAgentMCPConfig, SyncConfigBetweenAgents } from '../wailsjs/go/main/App'
```

### Frontend Structure

**Pages** (`frontend/src/components/`):
- `Dashboard.tsx`: Overview and quick actions
- `AgentsPage.tsx`: View/manage detected AI tools, sync between tools
- `SettingsPage.tsx`: GitHub token, Gist ID, encryption settings

**i18n** (`frontend/src/i18n/`):
- Chinese/English translations
- `useI18n()` hook for language switching

## Important Development Notes

### Adding a New AI Tool

1. **Edit `agents.yaml`** only:
   ```yaml
   - id: new-tool
     name: New Tool
     description: New AI coding tool
     platforms:
       windows:
         config_paths:
           - ~/.new-tool/settings.json
       darwin:
         config_paths:
           - ~/.new-tool/settings.json
       linux:
         config_paths:
           - ~/.config/new-tool/settings.json
     config_key: mcpServers  # or custom key
     format: standard        # or define new format + transforms
   ```

2. **If new format needed**, add transform rules:
   ```yaml
   transforms:
     standard_to_new_format:
       add_fields:
         custom_field: value
       remove_fields: [unwanted]
   ```

3. **No code changes required** - restart app to detect

### Path Expansion

All config paths support variables:
- `~`: User home directory
- `$APPDATA`: Windows AppData (e.g., `C:\Users\<user>\AppData\Roaming`)
- `$ProgramData`: Windows ProgramData (e.g., `C:\ProgramData`)

**Implementation:** `services/config_loader.go: ExpandPath()`

### Config File Structure

Each AI tool's config file contains MCP servers under a specific key:

**Standard format** (most tools):
```json
{
  "mcpServers": {
    "server-name": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {"GITHUB_TOKEN": "xxx"}
    }
  }
}
```

**Zed format**:
```json
{
  "context_servers": {
    "server-name": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {"GITHUB_TOKEN": "xxx"},
      "source": "custom",
      "enabled": true
    }
  }
}
```

### Local Data Storage

All app data stored in: `~/.mcp-sync/`
```
~/.mcp-sync/
  ├── sync_config.json       # Main config (may be encrypted)
  ├── versions/              # Config version history
  └── logs/                  # Sync operation logs
```

**Storage Service** (`services/storage.go`):
- JSON-based persistence
- Optional AES-256 encryption
- Version history tracking

### GitHub Gist Integration

**Setup:**
1. User creates GitHub Personal Access Token with `gist` scope
2. App creates or uses existing private Gist
3. Gist stores complete config from all detected tools

**Security:**
- Private Gists are NOT truly private (URL-accessible)
- App warns users about sensitive data
- Optional encryption before upload

**Implementation:** `services/gist_sync.go`

### Build System

**Wails CLI** handles build pipeline:
- Frontend: Vite builds React → static assets
- Backend: Go compiles to native executable
- Packaging: Embeds assets into single executable

**Frontend config:**
- `wails.json`: Uses `pnpm` (not `npm`)
- `frontend/vite.config.ts`: Build configuration
- `frontend/tailwind.config.js`: Tailwind CSS setup

## Troubleshooting Development Issues

### Wails dev fails to start
- Check Go version: `go version` (need 1.23+)
- Check Node version: `node --version` (need 18+)
- Check pnpm: `pnpm --version`
- Reinstall frontend deps: `cd frontend && pnpm install`

### Agent not detected
- Check `agents.yaml` has correct paths for current OS
- Verify path expansion: add debug print in `config_loader.go: ExpandPath()`
- Check file exists: `os.Stat(path)` in `detector.go: fileExists()`

### Format conversion not working
- Verify transform rule exists in `agents.yaml: transforms`
- Rule naming: must be `{source_format}_to_{target_format}`
- Check `config_loader.go: GetTransformRule()` returns non-nil

### Windows npx commands fail
- Ensure `windows_service.go: WrapNpxCommand()` is called
- Check if command starts with "npx"
- Verify wrapping: `["cmd", "/c", "npx", ...args]`

### Encryption errors
- Local: Check OS keyring access permissions
- Gist: Verify password matches between encrypt/decrypt
- Storage: Check `~/.mcp-sync/` directory permissions
