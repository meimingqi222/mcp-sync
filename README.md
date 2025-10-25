# MCP 同步器 (MCP Sync)

> 一个强大的 MCP 配置管理工具，支持在多个 AI 编程工具之间自动同步和转换配置。

## 功能特性

- 🔗 **多工具支持**: 支持 Claude Code、Cursor、Windsurf、Qwen CLI、Zed、Cline、Gemini CLI、Droid CLI、iFlow CLI 等
- 🔄 **智能同步**: 自动识别并处理不同工具间的配置差异
- 🎨 **格式转换**: 自动转换不同的 MCP 配置格式（Standard ↔ Zed）
- ⚙️ **配置化扩展**: 无需修改代码，通过 YAML 配置添加新工具
- 🌍 **跨平台**: 支持 Windows、macOS、Linux
- 💾 **Gist 同步**: 支持通过 GitHub Gist 备份和分享配置

## 快速开始

### 安装

1. 克隆项目
```bash
git clone <repo-url>
cd mcp-sync
```

2. 安装依赖
```bash
go mod download
npm install --prefix frontend
```

3. 构建
```bash
wails build
```

### 运行

#### 开发模式
```bash
wails dev
```

#### 生产模式
```bash
wails build
```

## 支持的工具

| 工具 | 配置文件 | 配置键 | 格式 |
|------|---------|--------|------|
| Claude Code | `~/.claude.json` | `mcpServers` | standard |
| Cursor | `~/.cursor/mcp.json` | `mcpServers` | standard |
| Windsurf | `~/.codeium/windsurf/mcp_config.json` | `mcpServers` | standard |
| Qwen CLI | `~/.qwen/settings.json` | `mcpServers` | standard |
| Zed | `~/.config/zed/settings.json` | `context_servers` | zed |
| Cline | `~/.config/Code/User/settings.json` | `mcpServers` | standard |
| Gemini CLI | `~/.gemini/settings.json` | `mcpServers` | standard |
| Droid CLI | `~/.factory/mcp.json` | `mcpServers` | standard |
| iFlow CLI | `~/.iflow/settings.json` | `mcpServers` | standard |

## 配置系统

### 架构

所有配置定义在 `services/agents.yaml` 文件中，包含两部分：

1. **转换规则** (Transforms): 定义不同格式间的转换方式
2. **Agent 定义** (Agents): 定义各个工具的配置规则

### 添加新 Agent

编辑 `services/agents.yaml`，在 `agents` 部分添加新的 agent 定义：

```yaml
- id: my-tool
  name: My AI Tool
  description: My custom AI coding tool
  platforms:
    windows:
      config_paths:
        - ~/.my-tool/settings.json
        - $APPDATA/my-tool/config.json
    darwin:
      config_paths:
        - ~/.my-tool/settings.json
        - ~/Library/Application Support/MyTool/settings.json
    linux:
      config_paths:
        - ~/.my-tool/settings.json
        - ~/.config/my-tool/settings.json
  config_key: mcpServers          # MCP 配置在文件中的键名
  format: standard                # 配置格式类型
```

#### 配置字段说明

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `id` | string | ✓ | 工具的唯一标识符（驼峰命名） |
| `name` | string | ✓ | 工具的显示名称 |
| `description` | string | ✓ | 工具的简单描述 |
| `platforms` | object | ✓ | 各平台的配置（windows, darwin, linux） |
| `config_paths` | array | ✓ | 该平台上的配置文件路径列表 |
| `config_key` | string | ✓ | MCP 服务器配置在 JSON 中的键名 |
| `format` | string | ✓ | 配置格式类型（预定义或自定义） |

#### 路径变量

支持以下路径变量（会自动展开）：

- `~` - 用户主目录
- `$APPDATA` - Windows AppData 目录（仅 Windows）
- `$ProgramData` - Windows ProgramData 目录（仅 Windows）
- 常用路径如 `Library/Application Support` 等

#### 配置键说明

`config_key` 是 MCP 服务器配置在 JSON 文件中的键名：

- **Standard 格式**: 通常使用 `mcpServers`
  ```json
  {
    "mcpServers": {
      "server-name": {...}
    }
  }
  ```

- **Zed 格式**: 使用 `context_servers`
  ```json
  {
    "context_servers": {
      "server-name": {...}
    }
  }
  ```

### 配置转换规则

转换规则定义如何在不同格式间转换 MCP 配置。在 `services/agents.yaml` 的 `transforms` 部分定义：

```yaml
transforms:
  standard_to_zed:
    add_fields:
      source: "custom"
      enabled: true
    keep_fields:
      - command
      - args
      - env
  
  zed_to_standard:
    remove_fields:
      - source
      - enabled
    keep_fields:
      - command
      - args
      - env
  
  # 自定义转换规则示例
  my_format_to_other_format:
    add_fields:
      custom_field: value
      another_field: true
    remove_fields:
      - unwanted_field
    keep_fields:
      - command
      - args
```

#### 转换规则字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `add_fields` | object | 转换时要添加的新字段及其值 |
| `remove_fields` | array | 转换时要移除的字段名称列表 |
| `keep_fields` | array | 要保留的字段名称列表（如果为空则保留除 `remove_fields` 外的所有字段） |

#### 转换规则命名约定

转换规则的键名遵循 `{源格式}_to_{目标格式}` 的约定：

```
standard_to_zed      # 标准格式转换到 Zed 格式
zed_to_standard      # Zed 格式转换到标准格式
custom_to_standard   # 自定义格式转换到标准格式
```

### 完整示例

以下是添加一个名为 `nova-code` 的新工具的完整示例：

```yaml
# 配置转换规则（如果需要自定义格式）
transforms:
  nova_custom_to_standard:
    remove_fields:
      - nova_internal_id
    keep_fields:
      - command
      - args
      - env

agents:
  # ... 其他 agent 定义 ...
  
  - id: nova-code
    name: Nova Code
    description: Nova AI Code Editor
    platforms:
      windows:
        config_paths:
          - $APPDATA/NovaCode/settings.json
          - ~/.nova-code/config.json
      darwin:
        config_paths:
          - ~/Library/Application Support/NovaCode/settings.json
          - ~/.nova-code/config.json
      linux:
        config_paths:
          - ~/.config/nova-code/settings.json
          - ~/.nova-code/config.json
    config_key: mcpServers
    format: standard
```

## 同步机制

### 工作流程

1. **检测工具**: 系统自动扫描用户计算机上已安装的工具
2. **加载配置**: 从每个工具的配置文件读取 MCP 服务器配置
3. **智能同步**: 用户选择源工具和目标工具，系统自动：
   - 读取源工具的配置
   - 识别源和目标工具的格式差异
   - 根据转换规则自动转换格式
   - 处理配置键名称的差异
   - 写入目标工具的配置文件

### 自动转换内容

| 差异类型 | 说明 | 自动处理 |
|--------|------|--------|
| 配置键名称 | `mcpServers` vs `context_servers` | ✓ |
| 格式转换 | Standard vs Zed vs 其他 | ✓ |
| 字段映射 | 不同格式的字段差异 | ✓ |
| 平台路径 | Windows/macOS/Linux 路径差异 | ✓ |

### 同步示例

从 Claude Code (Standard) 同步到 Zed (Zed 格式)：

```
源工具:       Claude Code
源格式:       standard (mcpServers)
源配置 JSON:  {"mcpServers": {"my-server": {...}}}

目标工具:     Zed
目标格式:     zed (context_servers)

转换过程:
1. 读取源配置的 mcpServers
2. 应用 standard_to_zed 转换规则
3. 添加 source: "custom" 和 enabled: true 字段
4. 保留 command、args、env 字段
5. 写入目标配置为 context_servers

目标配置 JSON: {"context_servers": {"my-server": {...}}}
```

## 高级用法

### 编辑配置

1. 打开 MCP 配置标签页
2. 选择工具
3. 在下方编辑器中修改 JSON 配置
4. 点击"编辑配置"按钮进入编辑模式
5. 修改后点击"保存配置"

### 同步配置

1. 选择源工具（已检测到的工具）
2. 查看其 MCP 配置
3. 点击"同步到"按钮选择目标工具
4. 系统自动处理格式差异并同步

### 通过 GitHub Gist 备份

1. 在"设置"标签页配置 GitHub Token 和 Gist ID
2. 使用"推送到 Gist"备份当前配置
3. 使用"从 Gist 拉取"恢复配置

## 项目结构

```
mcp-sync/
├── services/
│   ├── agents.yaml              # 所有 Agent 和转换规则定义
│   ├── config_loader.go         # 配置文件加载器
│   ├── detector.go              # Agent 检测器
│   ├── config_manager.go        # 配置管理器
│   ├── app_service.go           # 应用服务（包含同步逻辑）
│   └── ...
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── AgentsPage.tsx   # MCP 配置页面
│   │   │   ├── Dashboard.tsx    # 仪表板
│   │   │   └── ...
│   │   └── ...
│   └── ...
├── models/
│   └── types.go                 # 数据结构定义
├── app.go                       # 应用主入口
├── main.go                      # 程序入口
└── ...
```

## 开发

### 技术栈

- **后端**: Go + Wails
- **前端**: React + TypeScript + Tailwind CSS
- **配置**: YAML

### 本地开发

```bash
# 安装依赖
go mod download
npm install --prefix frontend

# 开发模式（热重载）
wails dev

# 构建
wails build
```

### 添加新功能

1. 修改 `services/agents.yaml` 配置（如需要）
2. 在 Go 代码中实现功能
3. 在 React 组件中调用 Go 方法
4. 测试功能

## 安全

### Gist 同步安全

⚠️ **重要安全提示**

- **Secret Gist 不是完全私密的**：任何知道 URL 的人都可以访问
- **不要在 Gist 中存储 API 密钥、密码或其他敏感凭证**
- MCP 配置可能包含敏感信息

### 安全机制

MCP Sync 包含以下安全功能：

- ✓ 敏感字段自动检测和掩码
- ✓ Gist 同步前的安全警告
- ✓ Token 安全验证
- ✓ 配置加密存储（可选）

### 推荐做法

1. **使用专用 GitHub Token**
   - 仅授予 `gist` 作用域权限
   - 定期轮换（每 90 天）

2. **避免存储凭证**
   - 使用环境变量代替
   - 使用专业密钥管理服务

3. **考虑本地备份**
   - 不推送敏感配置到云端
   - 定期本地备份

详见 [SECURITY.md](docs/SECURITY.md) 了解更多安全指南。

## 文档

完整的项目文档位于 `docs/` 目录：

- **[文档索引](docs/README.md)** - 📚 快速导航和阅读建议
- **[快速开始](docs/QUICKSTART.md)** - 完整的安装、使用和故障排除指南
- **[项目概述](docs/PROJECT_SUMMARY.md)** - 完整的技术架构和实现说明
- **[安全指南](docs/SECURITY.md)** - 安全最佳实践
- **[加密存储](docs/ENCRYPTION.md)** - 本地加密存储功能详解
- **[CI/CD 自动构建](docs/CI-CD.md)** - GitHub Actions 自动构建与发布指南

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License
