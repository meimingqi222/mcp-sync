# MCP Sync - Codex 支持更新

## 新增功能

### ✅ Codex AI 支持

MCP Sync 现已完整支持 **Codex AI** MCP 配置:

| 平台 | 配置文件路径 |
|------|-------------|
| Windows | `~/.codex/settings.json` 或 `%APPDATA%/Codex/settings.json` |
| macOS | `~/.codex/settings.json` 或 `~/Library/Application Support/Codex/settings.json` |
| Linux | `~/.codex/settings.json` 或 `~/.config/codex/settings.json` |

配置格式: **Standard** (`mcpServers`)

### ✅ 配置转换功能

新增强大的配置转换系统,支持:

#### 核心功能
- 🔄 **双向转换**: Codex ↔ 其他工具(Claude, Cursor, Zed等)
- ✅ **格式验证**: 自动验证配置格式正确性
- 📦 **批量转换**: 一次性转换为多个目标格式
- 💾 **JSON 导出**: 导出转换结果为 JSON

#### API 方法

```javascript
// 1. 转换到 Codex
ConvertToCodex(sourceAgentID, sourceConfig)

// 2. 从 Codex 转换
ConvertFromCodex(targetAgentID, codexConfig)

// 3. 任意工具间转换
ConvertAgentConfig(sourceAgentID, targetAgentID, config)

// 4. 批量转换
BatchConvertConfig(sourceAgentID, config, targetAgentIDs)

// 5. 配置验证
ValidateConfigFormat(agentID, config)

// 6. 导出 JSON
ExportConversionAsJSON(result)
```

## 使用示例

### 从 Cursor 迁移到 Codex

```javascript
const cursorConfig = await GetAgentMCPConfig("cursor");
const result = await ConvertToCodex("cursor", cursorConfig.mcpServers);

if (result.success) {
  await SaveAgentMCPConfig("codex", {
    mcpServers: result.converted_config
  });
}
```

### 批量同步 Codex 配置

```javascript
const codexConfig = await GetAgentMCPConfig("codex");
const results = await BatchConvertConfig(
  "codex", 
  codexConfig.mcpServers,
  ["cursor", "claude-code", "zed"]
);
```

## 技术实现

### 新增文件

1. **services/converter.go** - 配置转换核心引擎
   - `ConfigConverter` 结构
   - 格式转换逻辑
   - 批量处理支持
   - 配置验证功能

2. **docs/CODEX_CONVERTER.md** - 完整使用文档
   - API 参考
   - 转换示例
   - 使用场景
   - 故障排除

### 修改文件

1. **services/agents.yaml** - 添加 Codex 配置定义
2. **services/app_service.go** - 集成转换器
3. **app.go** - 导出前端 API

## 更新的工具支持列表

| 工具 | 配置格式 | 状态 |
|------|---------|------|
| Claude Code | standard | ✅ |
| Cursor | standard | ✅ |
| Windsurf | standard | ✅ |
| Qwen CLI | standard | ✅ |
| Zed | zed | ✅ |
| Cline | standard | ✅ |
| Gemini CLI | standard | ✅ |
| Droid CLI | standard | ✅ |
| iFlow CLI | standard | ✅ |
| **Codex AI** | **standard** | ✅ **新增** |

## 转换支持矩阵

| 源 \ 目标 | Codex | Cursor | Claude | Zed | Windsurf |
|----------|-------|--------|--------|-----|----------|
| Codex | - | ✅ | ✅ | ✅ | ✅ |
| Cursor | ✅ | - | ✅ | ✅ | ✅ |
| Claude | ✅ | ✅ | - | ✅ | ✅ |
| Zed | ✅ | ✅ | ✅ | - | ✅ |
| Windsurf | ✅ | ✅ | ✅ | ✅ | - |

**说明**: 
- ✅ = 支持直接转换
- 所有 standard ↔ standard 格式无需转换
- standard ↔ zed 格式自动转换

## 配置转换规则

### Standard → Zed

```
添加字段:
  - source: "custom"
  - enabled: true

保留字段:
  - command
  - args
  - env
```

### Zed → Standard

```
移除字段:
  - source
  - enabled

保留字段:
  - command
  - args
  - env
```

## 兼容性

- ✅ Windows 10/11
- ✅ macOS 10.15+
- ✅ Linux (各主流发行版)
- ✅ 自动处理 Windows npx 命令包装
- ✅ 跨平台路径处理

## 文档

详细文档请参阅:
- **[Codex 配置转换指南](docs/CODEX_CONVERTER.md)** - 完整的 API 文档和使用示例
- **[主 README](README.md)** - 项目概述和快速开始
- **[快速开始](docs/QUICKSTART.md)** - 安装和使用指南

## 下一步计划

- [ ] 前端 UI 集成配置转换功能
- [ ] 添加转换历史记录
- [ ] 支持更多自定义格式
- [ ] 配置差异对比功能
- [ ] 转换预览功能

## 变更日志

**v1.1.0** - 2025-10-30
- ✨ 新增 Codex AI 支持
- ✨ 新增配置转换系统
- ✨ 新增批量转换功能
- ✨ 新增配置验证功能
- 📝 添加完整的转换文档

---

有问题或建议? 欢迎提交 Issue 或 PR!
