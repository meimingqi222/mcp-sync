# Codex TOML 支持实现

## 概述

MCP Sync 现已完整支持 Codex AI 的 TOML 格式配置,可以无缝读取、编辑和转换 Codex 的 MCP 配置。

## 实现的功能

### ✅ 已实现

1. **TOML 格式解析**
   - 读取 Codex 的 `config.toml` 文件
   - 解析 `[mcp_servers]` 部分
   - 保留其他 Codex 配置(model_provider, model 等)

2. **格式转换**
   - Codex TOML → 标准 JSON (用于展示和编辑)
   - 标准 JSON → Codex TOML (用于保存)
   - 自动处理格式差异

3. **配置同步**
   - 从其他工具同步到 Codex
   - 从 Codex 同步到其他工具
   - 批量转换支持

## 技术实现

### 新增文件

#### `services/toml_adapter.go`
TOML 格式适配器,核心功能:

```go
// 数据结构
type CodexConfig struct {
    ModelProvider  string                    `toml:"model_provider,omitempty"`
    Model          string                    `toml:"model,omitempty"`
    MCPServers     map[string]CodexMCPServer `toml:"mcp_servers,omitempty"`
    // ...
}

type CodexMCPServer struct {
    Command string            `toml:"command"`
    Args    []string          `toml:"args,omitempty"`
    Env     map[string]string `toml:"env,omitempty"`
    CWD     string            `toml:"cwd,omitempty"`
}

// 主要方法
func (ta *TOMLAdapter) ReadCodexConfig(filePath string) (*CodexConfig, error)
func (ta *TOMLAdapter) WriteCodexConfig(filePath string, config *CodexConfig) error
func (ta *TOMLAdapter) CodexToStandard(codexServers map[string]CodexMCPServer) map[string]interface{}
func (ta *TOMLAdapter) StandardToCodex(standardServers map[string]interface{}) map[string]CodexMCPServer
```

### 修改的文件

#### `services/agents.yaml`
```yaml
- id: codex
  name: Codex AI
  description: OpenAI Codex CLI
  platforms:
    windows:
      config_paths:
        - ~/.codex/config.toml
  config_key: mcp_servers
  format: codex_toml
```

#### `services/app_service.go`
- 添加 `tomlAdapter *TOMLAdapter` 字段
- 在 `GetAgentMCPConfig` 中检测 TOML 格式并使用适配器
- 在 `SaveAgentMCPConfig` 中检测 TOML 格式并使用适配器

#### `go.mod`
```go
require github.com/BurntSushi/toml v1.5.0
```

## 使用示例

### 读取 Codex 配置

Codex 的 `config.toml`:
```toml
model_provider = "paid"
model = "gpt-5"

[mcp_servers.filesystem]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/path"]

[mcp_servers.playwright]
command = "npx"
args = ["-y", "@playwright/mcp@latest"]
```

转换为标准格式(前端显示):
```json
{
  "mcp_servers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path"]
    },
    "playwright": {
      "command": "npx",
      "args": ["-y", "@playwright/mcp@latest"]
    }
  }
}
```

### 从其他工具同步到 Codex

1. 用户在 MCP Sync 中选择源工具(如 Claude Code)
2. 点击"同步到 Codex"
3. 系统自动:
   - 读取 Claude Code 的 JSON 配置
   - 转换为 Codex TOML 格式
   - 写入 `~/.codex/config.toml`
   - 保留 Codex 的其他配置项

### 从 Codex 同步到其他工具

1. 用户在 MCP Sync 中选择 Codex
2. 点击"同步到 Cursor"
3. 系统自动:
   - 读取 Codex 的 TOML 配置
   - 转换为标准 JSON 格式
   - 写入 Cursor 的配置文件

## 格式转换示例

### Codex TOML → 标准 JSON

**输入 (Codex TOML)**:
```toml
[mcp_servers.github]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-github"]
env = { "GITHUB_TOKEN" = "ghp_xxx" }
```

**输出 (标准 JSON)**:
```json
{
  "github": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-github"],
    "env": {
      "GITHUB_TOKEN": "ghp_xxx"
    }
  }
}
```

### 标准 JSON → Codex TOML

**输入 (标准 JSON)**:
```json
{
  "brave-search": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-brave-search"],
    "env": {
      "BRAVE_API_KEY": "your-key"
    }
  }
}
```

**输出 (Codex TOML)**:
```toml
[mcp_servers.brave-search]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-brave-search"]

[mcp_servers.brave-search.env]
BRAVE_API_KEY = "your-key"
```

## 特殊处理

### 1. 保留 Codex 原有配置
写入 MCP 配置时,保留 Codex 的其他配置项:
- `model_provider`
- `model`
- `model_reasoning_effort`
- `disable_response_storage`
- `model_providers`

### 2. 环境变量处理
TOML 环境变量使用内联表格格式:
```toml
env = { "KEY1" = "value1", "KEY2" = "value2" }
```

### 3. 可选字段
- `args`: 如果为空,不写入
- `env`: 如果为空,不写入
- `cwd`: 如果为空,不写入

## 测试

### 手动测试步骤

1. **启动应用**:
   ```bash
   wails dev
   ```

2. **检测 Codex**:
   - 应该在工具列表中看到 "Codex AI"
   - 状态应为"已检测"

3. **查看配置**:
   - 选择 Codex
   - 应该能看到当前的 MCP 配置(JSON 格式)

4. **编辑配置**:
   - 点击"编辑配置"
   - 修改 JSON
   - 保存
   - 验证 `~/.codex/config.toml` 文件更新

5. **同步配置**:
   - 从 Claude Code 同步到 Codex
   - 验证配置正确转换
   - 反向同步测试

### 验证文件

检查 Codex 配置文件:
```bash
cat ~/.codex/config.toml
```

应该看到:
- MCP 服务器配置在 `[mcp_servers.xxx]` 部分
- 其他 Codex 配置保持不变
- TOML 格式正确

## 故障排除

### 问题 1: Codex 未被检测

**原因**: 配置文件不存在

**解决**:
```bash
mkdir -p ~/.codex
touch ~/.codex/config.toml
```

### 问题 2: TOML 解析错误

**原因**: TOML 语法错误

**解决**:
- 使用在线 TOML 验证器检查语法
- 确保 `[mcp_servers.xxx]` 格式正确
- 检查引号和逗号

### 问题 3: 配置丢失

**原因**: 写入时覆盖了其他配置

**解决**:
- 检查 `toml_adapter.go` 的 `ReadCodexConfig` 方法
- 确保读取完整配置后再更新 MCP servers

### 问题 4: 环境变量格式错误

**原因**: 环境变量格式不符合 TOML 规范

**解决**:
```toml
# ❌ 错误
env = { KEY = value }

# ✅ 正确
env = { "KEY" = "value" }
```

## 限制和注意事项

### 当前限制

1. **仅支持 STDIO 传输**
   - Codex 目前只支持本地 MCP 服务器
   - 不支持远程 HTTP/SSE 服务器

2. **配置文件位置固定**
   - 只支持 `~/.codex/config.toml`
   - 不支持自定义位置

3. **TOML 格式要求**
   - 必须使用 `[mcp_servers.xxx]` 格式
   - 环境变量必须使用内联表格

### 最佳实践

1. **备份配置**
   ```bash
   cp ~/.codex/config.toml ~/.codex/config.toml.bak
   ```

2. **验证 TOML 语法**
   ```bash
   # 使用 Python 验证
   python -c "import tomllib; open('~/.codex/config.toml').read()"
   ```

3. **逐步测试**
   - 先添加一个简单的 MCP 服务器
   - 验证工作后再添加复杂配置

## 相关文档

- [Codex MCP 配置转换指南](CODEX_CONVERTER.md)
- [Codex 功能更新说明](../CODEX_UPDATE.md)
- [主 README](../README.md)
