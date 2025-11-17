# Codex MCP 配置限制说明

## 传输协议限制

### Codex 仅支持 stdio 传输

OpenAI Codex CLI 目前**仅支持 stdio（标准输入输出）传输方式**的 MCP 服务器，不支持：
- ❌ HTTP 传输
- ❌ SSE（Server-Sent Events）传输

相比之下，其他工具如 Claude Desktop、Cursor、Windsurf 等都支持多种传输方式。

## 同步行为

### 自动跳过不支持的服务器

当你将其他工具（如 Claude）的配置同步到 Codex 时，MCP Sync 会：

1. **自动识别** HTTP 和 SSE 类型的服务器
2. **跳过这些服务器**（不会同步到 Codex）
3. **记录警告日志**，显示跳过的服务器名称和原因
4. **仅同步** stdio 类型的服务器

### 示例

**源配置（Claude）：**
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem"]
    },
    "exa": {
      "type": "http",
      "url": "https://mcp.exa.ai/mcp"
    }
  }
}
```

**同步到 Codex 后：**
```toml
# 只有 stdio 服务器被同步
[mcp_servers.filesystem]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem"]

# exa 服务器被跳过，因为它是 HTTP 类型
```

**日志输出：**
```
[TOML] Skipping server 'exa': Codex does not support http transport (only stdio is supported)
[Warning] Skipped 1 HTTP/SSE servers (Codex only supports stdio)
Updated MCP servers count: 1 (original: 2, skipped: 1)
```

## 识别服务器类型

### stdio 类型（支持）
```json
{
  "server-name": {
    "command": "npx",
    "args": ["@some/mcp-server"]
  }
}
```
- ✅ 会被同步到 Codex

### HTTP 类型（不支持）
```json
{
  "server-name": {
    "type": "http",
    "url": "https://example.com/mcp"
  }
}
```
- ❌ 会被自动跳过

### SSE 类型（不支持）
```json
{
  "server-name": {
    "type": "sse",
    "url": "https://example.com/sse"
  }
}
```
- ❌ 会被自动跳过

## 查看同步结果

同步完成后，你可以：

1. **查看应用日志**：在 MCP Sync 界面中查看详细的同步日志
2. **检查 Codex 配置**：打开 `~/.codex/config.toml` 查看实际同步的服务器
3. **统计信息**：日志会显示原始服务器数、成功同步数和跳过数

## 解决方案

### 选项 1: 使用 stdio 版本（推荐）

许多 MCP 服务器同时提供 stdio 和 HTTP 版本。优先选择 stdio 版本：

```json
{
  "server-name": {
    "command": "npx",
    "args": ["-y", "@vendor/server-name"]
  }
}
```

### 选项 2: 等待 Codex 官方支持

Codex 团队正在考虑添加 HTTP/SSE 支持，详见：
- [GitHub Issue #2129: Native SSE transport support](https://github.com/openai/codex/issues/2129)

### 选项 3: 仅在其他工具中使用 HTTP 服务器

对于只有 HTTP/SSE 版本的 MCP 服务器，在支持的工具（Claude、Cursor 等）中使用它们。

## 常见问题

### Q: 为什么我的某些服务器没有同步到 Codex？

A: 检查这些服务器是否使用了 HTTP 或 SSE 传输。MCP Sync 会自动跳过这些服务器，并在日志中显示警告。

### Q: 我可以手动添加 HTTP 服务器到 Codex 吗？

A: 不可以。Codex 从架构层面不支持 HTTP/SSE 传输，手动添加也无法工作。

### Q: 跳过的服务器会影响其他工具吗？

A: 不会。跳过仅影响到 Codex 的同步，其他工具的配置保持不变。

### Q: 如何查看我有多少 HTTP/SSE 服务器？

A: 在同步到 Codex 时，查看日志中的 "skipped" 计数，或者检查源配置中带有 `"type": "http"` 或 `"type": "sse"` 的服务器。

## 参考资料

- [Codex MCP 文档](https://developers.openai.com/codex/mcp/)
- [MCP 规范](https://modelcontextprotocol.io/)
- [Codex SSE 支持请求](https://github.com/openai/codex/issues/2129)
