# Codex MCP 配置转换指南

## 概述

MCP Sync 现已支持 **Codex AI** 的 MCP 配置,并提供强大的配置转换功能,可以在 Codex 与其他 AI 工具之间自由转换配置。

## Codex 支持

### 配置路径

| 平台 | 配置文件路径 |
|------|-------------|
| Windows | `~/.codex/settings.json` 或 `%APPDATA%/Codex/settings.json` |
| macOS | `~/.codex/settings.json` 或 `~/Library/Application Support/Codex/settings.json` |
| Linux | `~/.codex/settings.json` 或 `~/.config/codex/settings.json` |

### 配置格式

Codex 使用标准的 MCP 配置格式 (`standard`),配置键为 `mcpServers`:

```json
{
  "mcpServers": {
    "my-server": {
      "command": "npx",
      "args": ["-y", "mcp-server"],
      "env": {
        "API_KEY": "your-key"
      }
    }
  }
}
```

## 配置转换功能

### 主要特性

- ✅ **双向转换**: Codex ↔ 其他工具(Claude Code, Cursor, Windsurf, Zed 等)
- ✅ **格式验证**: 自动验证配置格式是否正确
- ✅ **批量转换**: 一次性将配置转换为多个目标格式
- ✅ **JSON 导出**: 导出转换结果为 JSON 格式
- ✅ **自动格式适配**: 智能处理不同工具间的格式差异

### API 方法

#### 1. 转换到 Codex

将任意工具的配置转换为 Codex 格式:

```javascript
// 前端调用示例
const result = await ConvertToCodex(sourceAgentID, sourceConfig);

// result 结构
{
  "source_format": "zed",
  "target_format": "standard",
  "source_agent": "zed",
  "target_agent": "codex",
  "original_config": { /* 原始配置 */ },
  "converted_config": { /* 转换后的配置 */ },
  "success": true,
  "message": "Successfully converted from zed to standard format"
}
```

#### 2. 从 Codex 转换

将 Codex 配置转换为其他工具格式:

```javascript
const result = await ConvertFromCodex(targetAgentID, codexConfig);
```

#### 3. 任意工具间转换

在任意两个工具之间转换配置:

```javascript
const result = await ConvertAgentConfig(sourceAgentID, targetAgentID, sourceConfig);
```

#### 4. 批量转换

一次性转换为多个目标格式:

```javascript
const results = await BatchConvertConfig(
  sourceAgentID, 
  sourceConfig, 
  ["codex", "cursor", "zed", "claude-code"]
);

// results 是一个数组,包含每个目标的转换结果
```

#### 5. 配置验证

验证配置格式是否正确:

```javascript
const [isValid, errors] = await ValidateConfigFormat(agentID, config);

if (!isValid) {
  console.log("配置错误:", errors);
}
```

#### 6. 导出为 JSON

导出转换结果为 JSON 字符串:

```javascript
const jsonString = await ExportConversionAsJSON(result);
console.log(jsonString);
```

## 转换示例

### 示例 1: Zed → Codex

**源配置** (Zed `context_servers` 格式):

```json
{
  "filesystem-server": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path"],
    "source": "custom",
    "enabled": true
  }
}
```

**转换后** (Codex `mcpServers` 格式):

```json
{
  "filesystem-server": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path"]
  }
}
```

转换规则:
- 移除 Zed 特有字段: `source`, `enabled`
- 保留通用字段: `command`, `args`, `env`

### 示例 2: Codex → Zed

**源配置** (Codex `mcpServers` 格式):

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

**转换后** (Zed `context_servers` 格式):

```json
{
  "brave-search": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-brave-search"],
    "env": {
      "BRAVE_API_KEY": "your-key"
    },
    "source": "custom",
    "enabled": true
  }
}
```

转换规则:
- 保留所有 Codex 字段: `command`, `args`, `env`
- 添加 Zed 特有字段: `source: "custom"`, `enabled: true`

### 示例 3: 批量转换

将 Codex 配置同时转换为多个格式:

```javascript
const codexConfig = {
  "github-server": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-github"],
    "env": {
      "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_xxx"
    }
  }
};

const results = await BatchConvertConfig(
  "codex",
  codexConfig,
  ["cursor", "claude-code", "zed", "windsurf"]
);

// 结果包含4个转换结果
results.forEach(result => {
  console.log(`${result.target_agent}: ${result.success ? '✓' : '✗'}`);
});
```

## 使用场景

### 场景 1: 迁移到 Codex

从其他工具迁移到 Codex:

1. 检测已安装的工具
2. 读取源工具的 MCP 配置
3. 使用 `ConvertToCodex` 转换配置
4. 保存到 Codex 配置文件

### 场景 2: 从 Codex 同步到其他工具

将 Codex 配置同步到其他工具:

1. 读取 Codex 的 MCP 配置
2. 使用 `ConvertFromCodex` 或 `BatchConvertConfig` 转换
3. 保存到目标工具配置文件

### 场景 3: 配置备份与共享

备份和共享配置:

1. 使用 `ConvertToCodex` 统一格式
2. 使用 `ExportConversionAsJSON` 导出
3. 保存或分享 JSON 文件
4. 使用 `ConvertFromCodex` 恢复到目标工具

## 转换规则配置

转换规则在 `services/agents.yaml` 中定义。如需自定义 Codex 转换规则:

```yaml
transforms:
  # Codex 到其他格式的转换(如果需要)
  standard_to_custom_format:
    add_fields:
      custom_field: value
    remove_fields:
      - unwanted_field
    keep_fields:
      - command
      - args
      - env
  
  custom_format_to_standard:
    remove_fields:
      - custom_field
    keep_fields:
      - command
      - args
      - env
```

## 注意事项

### 1. 配置键差异

不同工具使用不同的配置键:
- **Standard 格式** (Codex, Claude Code, Cursor等): `mcpServers`
- **Zed 格式**: `context_servers`

转换时会自动处理键名差异。

### 2. Windows npx 命令

在 Windows 上,MCP Sync 会自动处理 npx 命令:
- 检测 `npx` 命令
- 自动包装为 `cmd /c npx`
- 同步时自动应用

### 3. 环境变量

`env` 字段中的环境变量会原样保留,请注意:
- 不要在配置文件中硬编码敏感信息
- 建议使用环境变量引用
- 使用 Gist 同步时启用加密

### 4. 格式验证

使用 `ValidateConfigFormat` 验证转换后的配置:

```javascript
const [isValid, errors] = await ValidateConfigFormat("codex", convertedConfig);

if (!isValid) {
  errors.forEach(error => console.error(error));
}
```

## 完整工作流示例

### 从 Cursor 迁移到 Codex

```javascript
// 1. 检测已安装的工具
const agents = await DetectAgents();
const cursorAgent = agents.find(a => a.id === "cursor");

if (cursorAgent && cursorAgent.status === "detected") {
  // 2. 读取 Cursor 配置
  const cursorConfig = await GetAgentMCPConfig("cursor");
  
  // 3. 提取 MCP 服务器配置
  const servers = cursorConfig.mcpServers;
  
  // 4. 转换为 Codex 格式
  const result = await ConvertToCodex("cursor", servers);
  
  if (result.success) {
    // 5. 保存到 Codex
    await SaveAgentMCPConfig("codex", {
      mcpServers: result.converted_config
    });
    
    console.log("✓ 成功迁移配置到 Codex");
  }
}
```

### 批量同步 Codex 配置

```javascript
// 1. 读取 Codex 配置
const codexConfig = await GetAgentMCPConfig("codex");
const servers = codexConfig.mcpServers;

// 2. 批量转换到多个工具
const targetAgents = ["cursor", "claude-code", "zed"];
const results = await BatchConvertConfig("codex", servers, targetAgents);

// 3. 保存到各个工具
for (const result of results) {
  if (result.success) {
    const targetKey = result.target_format === "zed" 
      ? "context_servers" 
      : "mcpServers";
    
    await SaveAgentMCPConfig(result.target_agent, {
      [targetKey]: result.converted_config
    });
    
    console.log(`✓ ${result.target_agent} 同步成功`);
  }
}
```

## 故障排除

### 问题 1: 转换失败

**错误**: `No transform rule found`

**解决方案**: 
- 检查 `services/agents.yaml` 是否定义了转换规则
- 确保源格式和目标格式的转换规则存在
- 使用 standard 格式作为中间格式

### 问题 2: 配置验证失败

**错误**: `Server xxx: missing 'command' field`

**解决方案**:
- 检查配置是否包含必需字段 `command`
- 确保配置格式正确(JSON 格式)
- 使用 `ValidateConfigFormat` 查看详细错误

### 问题 3: Windows npx 命令问题

**错误**: npx 命令无法执行

**解决方案**:
- 系统会自动处理 Windows npx 命令
- 确保在 Windows 上运行
- 检查 `cmd /c` 包装是否正确应用

## 相关文档

- [快速开始](QUICKSTART.md)
- [项目概述](PROJECT_SUMMARY.md)
- [安全指南](SECURITY.md)
- [主 README](../README.md)
