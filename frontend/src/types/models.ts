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
