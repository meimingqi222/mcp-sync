import React, { useEffect, useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/Card"
import { Button } from "./ui/Button"
import { RefreshCw, Check, AlertCircle } from "lucide-react"
import { useI18n } from "../i18n/useI18n"
import { ConflictDetails } from "../types/models"
import { ConflictResolver } from "./ConflictResolver"

interface SyncStatus {
  lastSyncTime: string
  status: "success" | "failed" | "pending"
  message?: string
}

interface AgentStats {
  detectedAgents: number
  totalServers: number
}

export function Dashboard() {
  const { t, tReplace } = useI18n()
  const [syncStatus, setSyncStatus] = useState<SyncStatus>({
    lastSyncTime: "Never",
    status: "pending",
  })
  const [agentStats, setAgentStats] = useState<AgentStats>({
    detectedAgents: 0,
    totalServers: 0,
  })
  const [isLoading, setIsLoading] = useState(false)
  const [showConflictResolver, setShowConflictResolver] = useState(false)
  const [conflictDetails, setConflictDetails] = useState<ConflictDetails | null>(null)

  useEffect(() => {
    loadSyncStatus()
    loadAgentStats()

    // Listen for sync-status-update events
    if (window.runtime?.EventsOn) {
      window.runtime.EventsOn('sync-status-update', (data: any) => {
        setSyncStatus(data)
      })
    }
  }, [])

  const loadSyncStatus = async () => {
    try {
      // Use GetSyncReadyStatus for complete configuration check (including encryption)
      if ((window as any).go?.main?.App?.GetSyncReadyStatus) {
        const readyStatus = await (window as any).go.main.App.GetSyncReadyStatus()

        if (readyStatus.ready) {
          // Fully configured and ready - get last sync time from config
          const config = await (window as any).go.main.App.GetSyncConfig()
          const lastSyncDate = config?.last_sync_time ? new Date(config.last_sync_time) : null
          setSyncStatus({
            lastSyncTime: lastSyncDate ? lastSyncDate.toLocaleString() : "Never",
            status: "success",
          })
        } else if (readyStatus.has_token || readyStatus.has_gist_id) {
          // Partially configured - show warning with missing items
          setSyncStatus({
            lastSyncTime: "Never",
            status: "failed", // Use failed to show warning color
            message: readyStatus.message || "Configuration incomplete",
          })
        } else {
          // Not configured at all
          setSyncStatus({
            lastSyncTime: "Never",
            status: "pending",
          })
        }
      } else if ((window as any).go?.main?.App?.GetSyncConfig) {
        // Fallback to old behavior if new API not available
        const config = await (window as any).go.main.App.GetSyncConfig()
        const isConfigured = config && config.github_token && config.gist_id && config.enable_encryption

        if (isConfigured) {
          const lastSyncDate = config.last_sync_time ? new Date(config.last_sync_time) : null
          setSyncStatus({
            lastSyncTime: lastSyncDate ? lastSyncDate.toLocaleString() : "Never",
            status: "success",
          })
        } else if (config && (config.github_token || config.gist_id)) {
          // Partially configured
          setSyncStatus({
            lastSyncTime: "Never",
            status: "failed",
            message: "Encryption not configured - go to Settings to complete setup",
          })
        } else {
          setSyncStatus({
            lastSyncTime: "Never",
            status: "pending",
          })
        }
      }
    } catch (error) {
      console.error("Failed to load sync status:", error)
      setSyncStatus({
        lastSyncTime: "Never",
        status: "pending",
      })
    }
  }

  const loadAgentStats = async () => {
    try {
      if (!(window as any).go?.main?.App?.DetectAgents) return

      const agents = await (window as any).go.main.App.DetectAgents()
      let totalServers = 0

      if (agents && agents.length > 0) {
        for (const agent of agents) {
          try {
            const agentConfig = await (window as any).go.main.App.GetAgentMCPConfig(agent.id)
            if (agentConfig) {
              for (const key in agentConfig) {
                const serverMap = agentConfig[key]
                if (serverMap && typeof serverMap === 'object') {
                  totalServers += Object.keys(serverMap).length
                }
              }
            }
          } catch (e) {
            // Ignore
          }
        }
      }

      setAgentStats({
        detectedAgents: agents?.length || 0,
        totalServers: totalServers,
      })
    } catch (error) {
      console.error("Failed to load agent stats:", error)
    }
  }

  // Smart Sync: Check for conflicts, if any show resolver, otherwise merge
  const handleSync = async () => {
    setIsLoading(true)
    try {
      // Check for any conflicts (both push and pull)
      const pushConflict = await (window as any).go.main.App.DetectPushConflict()
      const pullConflict = await (window as any).go.main.App.DetectPullConflict()

      const hasConflict = (pushConflict && pushConflict.has_conflict) || (pullConflict && pullConflict.has_conflict)

      if (hasConflict) {
        // Fetch detailed conflict info and show resolver
        const details = await (window as any).go.main.App.GetConflictDetails()
        setConflictDetails(details)
        setShowConflictResolver(true)
        setIsLoading(false)
        return
      }

      // No conflict -> Perform merge (sync both ways)
      await (window as any).go.main.App.ResolveConflict("", "merge")

      setSyncStatus({
        lastSyncTime: new Date().toLocaleString(),
        status: "success",
        message: t("dashboard.sync_success") || "Synced successfully",
      })

      // Reload stats
      loadAgentStats()

      setTimeout(() => {
        setSyncStatus(prev => ({ ...prev, message: undefined }))
        loadSyncStatus()
      }, 3000)
    } catch (error) {
      console.error("Sync error:", error)
      setSyncStatus({
        lastSyncTime: new Date().toLocaleString(),
        status: "failed",
        message: `Sync failed: ${String(error)}`,
      })
    } finally {
      setIsLoading(false)
    }
  }

  const handleResolveConflict = async (decisions: Record<string, string>) => {
    setIsLoading(true)
    setShowConflictResolver(false)
    try {
      await (window as any).go.main.App.ResolveConflictSelective(decisions)

      setSyncStatus({
        lastSyncTime: new Date().toLocaleString(),
        status: "success",
        message: "Conflicts resolved and synced successfully",
      })
      setTimeout(() => {
        setSyncStatus(prev => ({ ...prev, message: undefined }))
        loadSyncStatus()
      }, 3000)
    } catch (error) {
      console.error("Resolution error:", error)
      setSyncStatus({
        lastSyncTime: new Date().toLocaleString(),
        status: "failed",
        message: `Resolution failed: ${String(error)}`,
      })
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <>
      <div className="space-y-8">
        {/* Header */}
        <div>
          <h1 className="text-3xl font-bold">{t("app.title")}</h1>
          <p className="text-muted-foreground mt-1">{t("dashboard.subtitle")}</p>
        </div>

        {/* Quick Sync Card */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <RefreshCw className="w-5 h-5" />
              {t("dashboard.sync_status")}
            </CardTitle>
            <CardDescription>{t("dashboard.last_sync")}: {syncStatus.lastSyncTime}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                {syncStatus.status === "success" && (
                  <div className="flex items-center gap-2 text-green-600">
                    <Check className="w-5 h-5" />
                    <span>{t("common.success")}</span>
                  </div>
                )}
                {syncStatus.status === "failed" && (
                  <div className="flex items-center gap-2 text-yellow-600">
                    <AlertCircle className="w-5 h-5" />
                    <span>{t("dashboard.settings_status_not_configured") || "Setup Required"}</span>
                  </div>
                )}
                {syncStatus.status === "pending" && (
                  <span className="text-muted-foreground">Not configured</span>
                )}
              </div>
              <Button
                onClick={handleSync}
                disabled={isLoading || syncStatus.status !== "success"}
                className="gap-2"
                title={syncStatus.status !== "success" ? "Complete sync configuration in Settings first" : ""}
              >
                <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
                {t("common.sync") || "Sync"}
              </Button>
            </div>
            {syncStatus.message && (
              <div className={`text-sm p-2 rounded ${syncStatus.status === "failed"
                  ? "bg-yellow-50 text-yellow-800 border border-yellow-200"
                  : "text-muted-foreground"
                }`}>
                {syncStatus.message}
                {syncStatus.status === "failed" && (
                  <span
                    className="ml-2 underline cursor-pointer hover:text-yellow-900"
                    onClick={() => window.location.hash = '#settings'}
                  >
                    Go to Settings →
                  </span>
                )}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Stats & Quick Links */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card className="cursor-pointer hover:shadow-md transition-shadow" onClick={() => window.location.hash = '#agents'}>
            <CardHeader>
              <CardTitle className="text-lg flex items-center justify-between">
                {t("dashboard.detected_tools")}
                <span className="text-2xl font-bold text-blue-600">{agentStats.detectedAgents}</span>
              </CardTitle>
              <CardDescription>{t("menu.mcp_config")} - {t("dashboard.click_to_manage")}</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                {agentStats.detectedAgents === 0
                  ? t("dashboard.tools_found_zero")
                  : agentStats.detectedAgents === 1
                    ? t("dashboard.tools_found_one")
                    : tReplace("dashboard.tools_found_many", { count: agentStats.detectedAgents.toString() })}
              </p>
            </CardContent>
          </Card>

          <Card className="cursor-pointer hover:shadow-md transition-shadow" onClick={() => window.location.hash = '#agents'}>
            <CardHeader>
              <CardTitle className="text-lg flex items-center justify-between">
                {t("dashboard.total_servers")}
                <span className="text-2xl font-bold text-green-600">{agentStats.totalServers}</span>
              </CardTitle>
              <CardDescription>{t("dashboard.mcp_servers_across_tools")}</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                {agentStats.totalServers === 0
                  ? t("dashboard.no_servers_configured")
                  : agentStats.totalServers === 1
                    ? t("dashboard.servers_configured_one")
                    : tReplace("dashboard.servers_configured_many", { count: agentStats.totalServers.toString() })}
              </p>
            </CardContent>
          </Card>

          <Card className="cursor-pointer hover:shadow-md transition-shadow" onClick={() => window.location.hash = '#settings'}>
            <CardHeader>
              <CardTitle className="text-lg flex items-center justify-between">
                {t("menu.settings")}
                <span className={`text-sm px-2 py-1 rounded ${syncStatus.status === 'success' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>
                  {syncStatus.status === 'success' ? t("dashboard.settings_status_configured") : t("dashboard.settings_status_not_configured")}
                </span>
              </CardTitle>
              <CardDescription>{t("dashboard.settings_description")}</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                {syncStatus.status === 'success'
                  ? tReplace("dashboard.last_sync_info", { time: syncStatus.lastSyncTime })
                  : t("dashboard.setup_sync_description")}
              </p>
            </CardContent>
          </Card>
        </div>
      </div>

      {showConflictResolver && conflictDetails && (
        <ConflictResolver
          details={conflictDetails}
          onResolve={handleResolveConflict}
          onCancel={() => {
            setShowConflictResolver(false)
            setIsLoading(false)
          }}
        />
      )}
    </>
  )
}
