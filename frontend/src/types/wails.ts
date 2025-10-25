import { Agent, MCPServer, SyncConfig, ConfigVersion, SyncLog } from "./models"

interface AppRuntime {
  DetectAgents(): Promise<Agent[]>
  InitializeGistSync(token: string, gistID: string): Promise<void>
  GetSyncConfig(): Promise<SyncConfig>
  SaveSyncConfig(config: SyncConfig): Promise<void>
  PushToGist(servers: MCPServer[]): Promise<void>
  PullFromGist(): Promise<MCPServer[]>
  ApplyConfigToAgent(agentID: string, servers: MCPServer[]): Promise<void>
  ApplyConfigToAllAgents(servers: MCPServer[]): Promise<void>
  GetConfigVersions(limit: number): Promise<ConfigVersion[]>
  GetSyncLogs(limit: number): Promise<SyncLog[]>
  Greet(name: string): Promise<string>
}

interface EventEmitter {
  EventsOn(event: string, callback: (data: any) => void): void
  EventsEmit(event: string, ...args: any[]): void
  EventsOff(event: string): void
}

declare global {
  interface Window {
    app: AppRuntime
    runtime: EventEmitter
  }
}

export {}
