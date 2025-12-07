export interface Agent {
  id: string
  name: string
  platform: string
  status: "detected" | "not_installed"
  configPaths: string[]
  enabled: boolean
}

export interface MCPServer {
  id: string
  name: string
  command: string
  args: string[]
  env: Record<string, string>
  enabled: boolean
  description: string
  supportedAgents: string[]
  createdAt?: string
}

export interface SyncConfig {
  id: string
  servers: MCPServer[]
  lastSyncTime: string
  lastSyncStatus: string
  gistID: string
  githubToken: string
  autoSync: boolean
  autoSyncInterval: number
}

export interface ConfigVersion {
  id: string
  timestamp: string
  content: string
  source: "local" | "gist"
  note: string
}

export interface SyncLog {
  id: string
  timestamp: string
  action: "push" | "pull" | "conflict"
  status: "success" | "failed"
  message: string
  details?: string
}

export interface SyncConflict {
  has_conflict: boolean
  conflict_type: "push_conflict" | "pull_conflict"
  local_version: ConfigVersion
  remote_version: ConfigVersion
  message: string
}

export interface ServerConflict {
  server_id: string
  local: MCPServer
  remote: MCPServer
}

export interface AgentDiff {
  agent_id: string
  local_only: MCPServer[]
  remote_only: MCPServer[]
  conflicts: ServerConflict[]
}

export interface ConflictDetails {
  agents: Record<string, AgentDiff>
}
