# MCP 同步器 - 快速开始指南

> 一个强大的 MCP 配置管理工具，支持在多个 AI 编程工具之间自动同步和转换配置。

## ✨ 功能特性

- 🔗 **多工具支持**: Claude Code、Cursor、Windsurf、Qwen CLI、Zed、Cline 等
- 🔄 **智能同步**: 自动识别并处理不同工具间的配置差异
- 🎨 **格式转换**: 自动转换不同的 MCP 配置格式（Standard ↔ Zed）
- ⚙️ **配置化扩展**: 无需修改代码，通过 YAML 配置添加新工具
- 🌍 **跨平台**: 支持 Windows、macOS、Linux
- 💾 **Gist 同步**: 支持通过 GitHub Gist 备份和分享配置
- 🔒 **本地加密**: 使用 AES-256 加密保护本地配置文件

## 📋 系统要求

**开发环境**:
- Go 1.23+
- Node.js 18+
- npm 10+
- Wails CLI v2.10+

**运行环境**:
- Windows 10+ / macOS 10.15+ / Ubuntu 20.04+
- 100MB 磁盘空间
- 512MB RAM

## 🚀 安装和运行

### 方法一：开发模式运行

```bash
# 1. 克隆项目
git clone <repo-url>
cd mcp-sync

# 2. 安装依赖
go mod download
npm install --prefix frontend

# 3. 启动开发服务器
wails dev
```

### 方法二：构建生产版本

```bash
# 构建可执行文件
wails build -clean

# 可执行文件位置
build/bin/mcp-sync.exe  # Windows
build/bin/mcp-sync      # macOS/Linux

# 运行
./build/bin/mcp-sync
```

### 方法三：一键安装（Windows）

双击 `build.bat` 文件，自动完成依赖安装和构建。

## 📖 使用指南

### 首次使用

1. **启动应用**
   ```bash
   wails dev
   ```

2. **检测编程工具**
   - 打开应用 → "编程工具" 页面
   - 系统自动检测已安装的工具（Claude、Cursor 等）
   - 显示工具安装状态和配置文件路径

3. **配置 GitHub Token**
   - 访问：https://github.com/settings/tokens/new
   - 输入 Token 名称（如：mcp-sync）
   - 勾选 `gist` 权限
   - 生成并复制 Token
   - 在应用 "设置" 页面粘贴 Token

4. **添加 MCP 服务**
   - 打开 "MCP 服务" 页面
   - 从预设库选择服务（GitHub、Perplexity、Sequential Thinking）
   - 或手动添加自定义服务

5. **同步配置**
   - 点击 "立即同步" 推送到 Gist
   - 或设置自动同步

### 多设备同步

1. 在新设备上运行应用
2. 配置相同的 GitHub Token 和 Gist ID
3. 点击 "立即同步" → 从 Gist 拉取配置
4. 配置自动应用到所有已安装的工具

## 🔧 常用操作

### 同步配置

```bash
# 推送所有代理配置到云端
wails exec "PushAllAgentsToGist"

# 从云端拉取配置
wails exec "PullFromGist"
```

### 支持的编程工具

| 工具 | 配置文件 | 配置键 | 格式 |
|------|---------|--------|------|
| Claude Code | `~/.claude.json` | `mcpServers` | standard |
| Cursor | `~/.cursor/mcp.json` | `mcpServers` | standard |
| Windsurf | `~/.codeium/windsurf/mcp_config.json` | `mcpServers` | standard |
| Qwen CLI | `~/.qwen/settings.json` | `mcpServers` | standard |
| Zed | `~/.config/zed/settings.json` | `context_servers` | zed |
| Cline | `~/.config/Code/User/settings.json` | `mcpServers` | standard |

## 💾 数据存储

### 本地数据位置

```
Windows: C:\Users\<用户名>\.mcp-sync\
macOS:   ~/.mcp-sync/
Linux:   ~/.mcp-sync/

目录结构:
├── sync_config.json    # 主配置（可能加密）
├── versions/           # 配置版本历史（可能加密）
└── logs/               # 同步操作日志（可能加密）
```

### 云端存储

- **GitHub Gist** (私密存储)
- **文件**: `mcp-config.json`
- **加密**: 配置可使用 AES-256 加密后存储

## 🔒 加密功能

启用本地加密保护敏感信息：

1. 打开 "设置" 页面
2. 勾选 "启用加密"
3. 输入加密密码（建议 8 位以上，包含大小写、数字和特殊字符）
4. 点击 "保存设置"
5. 所有本地文件将自动加密

⚠️ **重要**: 加密密码一旦丢失，配置无法恢复！

## 🛠️ 开发

### 项目结构

```
mcp-sync/
├── app.go                  # 主应用逻辑
├── main.go                 # Wails 入口
├── models/
│   └── types.go           # 数据类型定义
├── services/              # 业务逻辑服务
│   ├── detector.go        # Agent 检测
│   ├── config_manager.go  # 配置管理
│   ├── gist_sync.go       # Gist 同步
│   ├── storage.go         # 本地存储（含加密）
│   └── app_service.go     # 核心服务
├── frontend/              # React 前端
│   ├── src/
│   │   ├── components/    # React 组件
│   │   ├── types/         # TypeScript 类型
│   │   └── lib/           # 工具函数
│   └── package.json       # 依赖配置
└── wails.json             # Wails 配置
```

### 开发命令

```bash
# 开发模式（热重载）
wails dev

# 构建
wails build -clean

# 查看支持平台
wails build -list

# 打包特定平台
wails build -platform windows/amd64
wails build -platform darwin/universal
wails build -platform linux/amd64
```

## ❓ 故障排除

### 检测不到编程工具

**解决方案**:
- 确认工具已正确安装并启动过
- 检查配置文件路径是否正确
- 重启应用

### 无法同步到 Gist

**解决方案**:
- 验证 GitHub Token 是否有 `gist` 权限
- Token 是否已过期
- 网络连接是否正常
- 在设置中重新输入 Token

### 配置文件路径错误

**解决方案**:
- 确保工具已启动一次（自动生成配置文件）
- 查看应用日志文件：`~/.mcp-sync/logs/`
- 检查工具是否在受支持列表中

### 编译错误

```bash
# 清理并重新安装
go clean -cache
go mod tidy
rm -rf frontend/node_modules
npm install --prefix frontend
```

### 获取帮助

- 查看完整文档：`docs/` 目录
- 检查 `docs/SECURITY.md` 了解安全最佳实践
- 检查 `docs/ENCRYPTION.md` 了解加密功能

## 📝 更新日志

### v2.0.0
- ✅ 新增本地文件加密功能（AES-256-GCM）
- ✅ 修复 Windows 环境变量问题
- ✅ 自动加密敏感配置文件
- ✅ 完善错误处理和日志记录

### v1.0.0
- ✅ 支持 9 种编程工具
- ✅ GitHub Gist 同步
- ✅ 配置格式自动转换
- ✅ 跨平台支持

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

---

**祝你使用愉快！** 🚀

更多详情请查看：
- [项目概述](PROJECT_SUMMARY.md)
- [安全指南](SECURITY.md)
- [加密存储](ENCRYPTION.md)
