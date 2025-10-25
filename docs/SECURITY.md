# MCP Sync 安全指南

## GitHub Gist 可见性问题

### 默认情况下，GitHub Gist 的可见性如何？

**重要：** 默认创建的 Secret Gist **不是完全私密的**！

- **Public Gist**: 任何人都可以通过搜索找到并查看
- **Secret Gist**: 
  - 不会出现在 GitHub 搜索结果中
  - 不会被搜索引擎索引
  - **但是**，任何知道 Gist URL 的人都可以访问！
  - 可以通过 GitHub API 列举（如果有 GitHub 账户）

### 示例

```
你的 Secret Gist URL:
https://gist.githubusercontent.com/username/abc123xyz/raw/mcp-config.json

任何知道这个 URL 的人都可以访问！
```

## MCP 配置中的敏感信息

MCP 配置文件可能包含：

```json
{
  "mcpServers": {
    "my-server": {
      "command": "python",
      "args": ["/path/to/server.py"],
      "env": {
        "API_KEY": "sk-xxxxxxxxxxxx",
        "DATABASE_URL": "postgresql://user:password@host/db",
        "SECRET_TOKEN": "token_value"
      }
    }
  }
}
```

**风险项**：
- API 密钥和令牌
- 数据库连接字符串
- 用户名和密码
- 文件系统路径和位置信息

## MCP Sync 中的安全机制

### 1. 敏感字段检测

系统会自动识别以下类型的敏感字段：

- `*_api_key`, `apikey`
- `token`, `*_token`
- `secret`, `*_secret`
- `password`, `passwd`
- `key`, `*_key`
- `auth`, `*_auth`

### 2. 掩码显示

在 UI 中，敏感值会自动被掩码处理：

```
原始值: "sk-1234567890abcdef"
显示值: "sk-**************ef"

原始值: "super_secret_password"
显示值: "su********************rd"
```

### 3. 安全警告

系统会在以下情况显示安全警告：

- 推送配置到 Gist 时
- 访问 GitHub 集成设置时

### 4. 加密存储

敏感的 GitHub Token 可以加密存储（可选）。

## 推荐做法

### ✅ 安全的做法

1. **使用环境变量**
   ```bash
   export MCP_API_KEY="your-secret-key"
   # 然后在配置中引用
   ```

2. **使用密钥管理服务**
   - GitHub Secrets（用于 CI/CD）
   - AWS Secrets Manager
   - HashiCorp Vault

3. **定期轮换令牌**
   - GitHub Personal Access Token
   - API 密钥

4. **限制令牌作用域**
   - 仅授予必要的权限
   - 不要使用通用令牌

5. **使用 GitHub Private Repository**
   - 比 Secret Gist 更安全
   - 需要 GitHub Pro 账户

### ❌ 不安全的做法

1. **在 Gist 中存储凭证**
   ```json
   ❌ 不要这样做：
   {
     "API_KEY": "sk-1234567890",
     "PASSWORD": "mypassword"
   }
   ```

2. **使用通用的 Personal Access Token**
   - 应该创建仅限 Gist 访问的令牌

3. **在公开仓库中共享 Gist 链接**
   ```
   ❌ 不要在 README 中写：
   My config: https://gist.github.com/.../abc123
   ```

4. **不定期轮换令牌**
   - 定期（每 3-6 个月）更新令牌

## GitHub Token 安全配置

### 创建安全的 Personal Access Token

1. 访问 GitHub Settings → Developer settings → Personal access tokens
2. 创建新 Token，选择最小必要权限
3. 仅选择 `gist` 作用域（不要选择 `repo` 或其他）
4. 设置过期时间（推荐 90 天）
5. 复制并立即保存到安全位置

### Token 示例

```
❌ 不安全（包含过多权限）：
ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx (full_repo scope)

✅ 安全（仅限 Gist）：
github_pat_xxxxxxxxxxx (gist scope only, expires in 90 days)
```

## 备份替代方案

### 选项 1: 本地备份（最安全）
```bash
# 定期手动备份
cp ~/.claude.json ~/.claude.json.backup
cp ~/.cursor/mcp.json ~/.cursor/mcp.json.backup
```

### 选项 2: 私密的 Git 仓库
```bash
# 在私密的 Git 仓库中版本控制
git init mcp-configs
echo "api_keys*" >> .gitignore
git add mcp-config.json
git commit -m "MCP configuration backup"
```

### 选项 3: 加密存储
```bash
# 使用 GPG 加密敏感配置
gpg --encrypt mcp-config.json
# 发送到 Gist 或云存储
```

### 选项 4: Secret Gist（适度风险）
- 可用于非敏感的配置
- **不要** 存储 API 密钥或凭证

## 数据泄露应急响应

如果不小心将敏感信息推送到 Gist：

1. **立即轮换凭证**
   - 重置 GitHub Token
   - 重置 API 密钥
   - 更改密码

2. **删除 Gist**
   ```
   GitHub 设置 → Gist 页面 → 删除 Gist
   ```

3. **检查访问日志**
   - GitHub 已记录谁访问了 Gist
   - 检查 GitHub 账户的活动日志

4. **联系 GitHub 支持**
   - 如果需要完全删除历史

## 安全检查清单

在使用 Gist 同步前：

- [ ] 已创建专用的 GitHub Personal Access Token
- [ ] Token 只有 `gist` 作用域
- [ ] Token 设置了过期时间（推荐 90 天）
- [ ] 已从配置中移除敏感数据
- [ ] 了解 Secret Gist 的安全限制
- [ ] 考虑使用本地备份代替
- [ ] 已告知团队成员安全政策
- [ ] 定期轮换凭证

## 监控和审计

### 启用安全警告

MCP Sync 会在以下情况提醒：

1. 首次推送配置到 Gist
2. 在敏感字段中检测到可能的凭证
3. Token 将要过期

### 定期审查

- 每月检查 GitHub Token 的使用情况
- 审查 Gist 的访问日志
- 确保没有无意中暴露敏感信息

## 联系和反馈

如果发现 MCP Sync 中的安全问题：

1. **不要** 在公开 Issue 中讨论
2. 通过私密渠道报告
3. 提供详细的重现步骤

## 相关资源

- [GitHub Gist 文档](https://docs.github.com/en/get-started/writing-on-github/editing-and-sharing-content-with-gists)
- [GitHub Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
- [OWASP 密钥管理](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html)
- [国密算法标准](https://en.wikipedia.org/wiki/GB/T_32905-2016)
