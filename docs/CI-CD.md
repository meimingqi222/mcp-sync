# CI/CD 自动构建与发布

## GitHub Actions 工作流

项目已配置 GitHub Actions 自动构建和发布流程，支持 Windows、macOS 和 Linux 三个平台。

## 🚀 自动触发

工作流会在以下情况自动运行：

1. **推送版本标签**：
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **手动触发**：
   - 访问 GitHub 仓库 → Actions → Build and Release → Run workflow

## 📦 构建内容

每次构建会生成：
- **Windows**: `mcp-sync.exe` (带图标的可执行文件)
- **macOS**: `mcp-sync.app` (应用包)
- **Linux**: `mcp-sync` (无扩展名可执行文件)

## 🔄 工作流程

### 1. 构建阶段 (Build)
- ✅ 检出代码
- ✅ 安装 Go 1.23
- ✅ 安装 Node.js 18
- ✅ 安装 Wails CLI
- ✅ 安装依赖 (Go modules + npm packages)
- ✅ 构建三个平台的应用
- ✅ 上传构建产物

### 2. 发布阶段 (Release)
- ✅ 仅在推送标签时触发
- ✅ 下载所有构建产物
- ✅ 创建 GitHub Release
- ✅ 上传文件到 Release
- ✅ 生成详细的发布说明

### 3. Docker 镜像 (可选)
- ✅ 构建 Docker 镜像
- ✅ 推送到 GitHub Container Registry
- ✅ 标签: `ghcr.io/meimingqi222/mcp-sync:latest`

## 📝 使用指南

### 方法一：创建发布版本

```bash
# 1. 更新版本号（可选）
# 编辑 wails.json 中的 version 字段

# 2. 创建并推送标签
git tag v1.0.0
git push origin v1.0.0

# 3. 访问 GitHub 查看 Actions
# https://github.com/meimingqi222/mcp-sync/actions
```

### 方法二：手动触发构建

1. 打开 GitHub 仓库
2. 点击 `Actions` 标签
3. 选择 `Build and Release` 工作流
4. 点击 `Run workflow` 按钮
5. 选择分支并点击运行

## 🎯 构建产物

构建完成后，产物会在以下位置：

### GitHub Release
- 下载地址：https://github.com/meimingqi222/mcp-sync/releases
- 包含所有平台的二进制文件
- 自动生成发布说明

### Actions Artifacts
- 临时存储（7 天）
- 下载地址：Actions → 具体任务 → Artifacts
- 仅用于测试，不发布

## 🛠️ 故障排除

### 构建失败

**常见原因**：
1. Go 版本不兼容
2. Node.js 依赖安装失败
3. Wails 构建错误

**解决方案**：
- 检查 Go 版本: `go version`
- 检查 Node 版本: `node --version`
- 本地测试: `wails build -clean`

### 发布失败

**检查项**：
1. 是否有 `release` 权限
2. 标签格式是否正确（v*.*.*）
3. GITHUB_TOKEN 是否可用

### macOS 构建问题

如果 macOS 构建失败，可能需要：
```bash
# 开发者签名（可选）
export APPLE_CERTIFICATE="${{ secrets.APPLE_CERTIFICATE }}"
export APPLE_CERTIFICATE_PASSWORD="${{ secrets.APPLE_CERTIFICATE_PASSWORD }}"
export APPLE_SIGNING_IDENTITY="${{ secrets.APPLE_SIGNING_IDENTITY }}"
```

## 🔐 环境变量

工作流会自动使用以下密钥：

| 密钥名称 | 用途 | 必需 |
|----------|------|------|
| `GITHUB_TOKEN` | 访问 GitHub API | ✅ 自动提供 |
| `APPLE_CERTIFICATE` | macOS 代码签名 | ❌ 可选 |
| `APPLE_CERTIFICATE_PASSWORD` | 证书密码 | ❌ 可选 |
| `APPLE_SIGNING_IDENTITY` | 签名身份 | ❌ 可选 |

## 📊 工作流状态徽章

在 README.md 中添加状态徽章：

```markdown
[![Build and Release](https://github.com/meimingqi222/mcp-sync/actions/workflows/build-release.yml/badge.svg)](https://github.com/meimingqi222/mcp-sync/actions/workflows/build-release.yml)
```

## 🎨 自定义

### 修改触发条件

编辑 `.github/workflows/build-release.yml`：

```yaml
on:
  push:
    tags:
      - 'v*.*.*'  # 只在推送 v*.*.* 标签时触发
  workflow_dispatch:  # 允许手动触发
```

### 添加平台

在 matrix 中添加新平台：

```yaml
strategy:
  matrix:
    include:
      - os: windows-latest
        platform: windows/amd64
      - os: macos-latest
        platform: darwin/universal
      - os: ubuntu-latest
        platform: linux/amd64
      # 添加新平台
      - os: macos-14
        platform: darwin/arm64  # Apple Silicon
```

## 📦 发布检查清单

发布前检查：

- [ ] 更新版本号到 `wails.json`
- [ ] 更新 `docs/CHANGELOG.md`（如果有）
- [ ] 测试本地构建: `wails build -clean`
- [ ] 创建发布标签: `git tag vX.X.X`
- [ ] 推送标签: `git push origin vX.X.X`
- [ ] 等待构建完成
- [ ] 验证 Release 页面
- [ ] 下载并测试二进制文件

## 🐛 报告问题

如果构建或发布失败：

1. 查看 Actions 日志
2. 检查错误信息
3. 在本地复现问题
4. 提交 Issue 描述问题

---

**提示**: 工作流文件位于 `.github/workflows/build-release.yml`

更多 Wails 构建选项：https://wails.io/docs/next/building
