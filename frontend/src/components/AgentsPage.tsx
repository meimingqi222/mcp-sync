import React, { useEffect, useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/Card"
import { Button } from "./ui/Button"
import { CheckCircle2, AlertCircle, Copy, Save, Search } from "lucide-react"
import { useI18n } from "../i18n/useI18n"

interface Agent {
  id: string
  name: string
  platform: string
  status: "detected" | "not_installed"
  configPaths: string[]
  existing_paths: string[]
  enabled: boolean
}

export function AgentsPage() {
  const { t, tReplace } = useI18n()
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null)
  const [configJson, setConfigJson] = useState("")
  const [editMode, setEditMode] = useState(false)
  const [saveMessage, setSaveMessage] = useState("")
  const [searchTerm, setSearchTerm] = useState("")

  const filteredAgents = agents.filter(agent =>
    agent.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    agent.platform.toLowerCase().includes(searchTerm.toLowerCase())
  )

  useEffect(() => {
    loadAgents()
  }, [])

  // Load config when selected agent changes
  useEffect(() => {
    if (selectedAgent) {
      loadAgentConfig(selectedAgent)
    }
  }, [selectedAgent])

  const loadAgents = async () => {
    try {
      setLoading(true)
      const result = await (window as any).go.main.App.DetectAgents()
      console.log("检测到的工具:", result)
      setAgents(result || [])
      if (result && result.length > 0) {
        setSelectedAgent(result[0].id)
      }
    } catch (error) {
      console.error("检测工具失败:", error)
      setAgents([])
    } finally {
      setLoading(false)
    }
  }

  const handleAgentSelect = (agentId: string) => {
    setSelectedAgent(agentId)
    // Config will be loaded automatically by useEffect
    setEditMode(false)
    setSaveMessage("")
  }

  const loadAgentConfig = async (agentId: string) => {
    try {
      const config = await (window as any).go.main.App.GetAgentMCPConfig(agentId)
      if (config) {
        setConfigJson(JSON.stringify(config, null, 2))
      } else {
        setConfigJson(JSON.stringify({ mcpServers: {} }, null, 2))
      }
    } catch (error) {
      console.error("加载配置失败:", error)
      setConfigJson(JSON.stringify({ mcpServers: {} }, null, 2))
    }
  }

  const handleSaveConfig = async () => {
    try {
      const config = JSON.parse(configJson)
      if (selectedAgent) {
        await (window as any).go.main.App.SaveAgentMCPConfig(selectedAgent, config)
        setSaveMessage("配置已保存!")
        setTimeout(() => setSaveMessage(""), 3000)
        setEditMode(false)
      }
    } catch (error) {
      setSaveMessage("JSON格式错误或保存失败!")
    }
  }

  const handleCopyToAgent = async (targetAgentId: string) => {
    try {
      if (!selectedAgent) {
        setSaveMessage("未选择源工具!")
        return
      }
      await (window as any).go.main.App.SyncConfigBetweenAgents(selectedAgent, targetAgentId)
      setSaveMessage(`已同步到 ${targetAgentId}（自动处理格式差异）`)
      setTimeout(() => setSaveMessage(""), 3000)
    } catch (error) {
      setSaveMessage("同步失败: " + (error as any).message)
    }
  }

  const currentAgent = agents.find(a => a.id === selectedAgent)

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">{t("mcp_config.title")}</h2>
        <p className="text-muted-foreground mt-1">
          {t("mcp_config.subtitle")}
        </p>
      </div>

      {loading ? (
        <div className="text-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">{t("mcp_config.detecting")}</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          {/* Left Sidebar: Tool Selection */}
          <div className="md:col-span-1 space-y-4">
            <div className="relative">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder={t("common.search") || "Search tools..."}
                className="w-full pl-9 pr-4 py-2 text-sm border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary/20"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>

            <div className="space-y-1 max-h-[600px] overflow-y-auto pr-1">
              {filteredAgents.length === 0 ? (
                <div className="text-center py-8 px-4 text-sm text-muted-foreground bg-muted/30 rounded-lg border border-dashed">
                  {agents.length === 0 ? t("mcp_config.no_tools_detected") : "No matching tools found"}
                </div>
              ) : (
                filteredAgents.map(agent => (
                  <button
                    key={agent.id}
                    onClick={() => handleAgentSelect(agent.id)}
                    className={`w-full flex items-center justify-between p-3 rounded-lg text-sm transition-all border ${selectedAgent === agent.id
                      ? "bg-primary/10 border-primary text-primary font-medium shadow-sm"
                      : "bg-card border-transparent hover:bg-accent hover:text-accent-foreground text-muted-foreground"
                      }`}
                  >
                    <div className="flex items-center gap-3">
                      <div className={`w-2 h-2 rounded-full ${agent.status === 'detected' ? 'bg-green-500' : 'bg-gray-300'}`} />
                      <span>{agent.name}</span>
                    </div>
                    {agent.status === "detected" && (
                      <CheckCircle2 className={`w-3.5 h-3.5 ${selectedAgent === agent.id ? 'opacity-100' : 'opacity-0'} transition-opacity`} />
                    )}
                  </button>
                ))
              )}
            </div>
          </div>

          {/* Right Content: Configuration */}
          <div className="md:col-span-3">
            {currentAgent ? (
              <div className="space-y-6 animate-in fade-in duration-300 slide-in-from-bottom-2">
                {/* Tool Info Card */}
                <Card>
                  <CardHeader className="pb-4">
                    <div className="flex items-start justify-between">
                      <div className="space-y-1">
                        <CardTitle className="text-xl flex items-center gap-2">
                          {currentAgent.name}
                          <span className={`text-xs px-2 py-0.5 rounded-full border ${currentAgent.status === "detected" ? "bg-green-50/50 text-green-700 border-green-200" : "bg-gray-100 text-gray-600 border-gray-200"}`}>
                            {currentAgent.status === "detected" ? t("mcp_config.status_detected") : t("mcp_config.status_not_installed")}
                          </span>
                        </CardTitle>
                        <p className="text-sm text-muted-foreground">
                          {tReplace("mcp_config.platform_config", { platform: currentAgent.platform })}
                        </p>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="space-y-2">
                      <div className="text-sm font-medium">{t("mcp_config.config_paths")}</div>
                      <div className="bg-muted/50 rounded-lg p-3 space-y-2 text-xs font-mono">
                        {currentAgent.existing_paths && currentAgent.existing_paths.length > 0 ? (
                          currentAgent.existing_paths.map((path, idx) => {
                            const isWsl = path.toLowerCase().includes("wsl.localhost") || path.toLowerCase().includes("wsl$")
                            return (
                              <div key={idx} className="flex items-start gap-2 text-green-700">
                                <CheckCircle2 className="w-3.5 h-3.5 mt-0.5 shrink-0" />
                                <div className="flex flex-col gap-1 w-full">
                                  <span className="break-all">{path}</span>
                                  {isWsl && (
                                    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-100 text-blue-800 w-fit">
                                      WSL Environment
                                    </span>
                                  )}
                                </div>
                              </div>
                            )
                          })
                        ) : (
                          <div className="text-muted-foreground italic px-1">No configuration file found</div>
                        )}

                        {currentAgent.configPaths && currentAgent.configPaths
                          .filter(p => !currentAgent.existing_paths?.includes(p))
                          .map((path, idx) => {
                            const isWsl = path.toLowerCase().includes("wsl.localhost") || path.toLowerCase().includes("wsl$")
                            return (
                              <div key={`missing-${idx}`} className="flex items-start gap-2 text-muted-foreground opacity-60">
                                <div className="w-3.5 h-3.5 mt-0.5 shrink-0 border rounded-full" />
                                <div className="flex flex-col gap-1 w-full">
                                  <span className="break-all">{path}</span>
                                  {isWsl && (
                                    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-muted text-muted-foreground w-fit border">
                                      WSL
                                    </span>
                                  )}
                                </div>
                              </div>
                            )
                          })
                        }
                      </div>
                    </div>
                  </CardContent>
                </Card>

                {/* JSON Editor */}
                <Card className="overflow-hidden border-t-4 border-t-primary/20">
                  <CardHeader className="bg-muted/10 border-b">
                    <div className="flex items-center justify-between">
                      <div>
                        <CardTitle className="text-base">{t("mcp_config.mcp_config_title")}</CardTitle>
                        <CardDescription className="text-xs">{t("mcp_config.mcp_config_subtitle")}</CardDescription>
                      </div>
                      <div className="flex gap-2">
                        <Button
                          variant={editMode ? "ghost" : "outline"}
                          size="sm"
                          onClick={() => setEditMode(!editMode)}
                          className="h-8"
                        >
                          {editMode ? t("mcp_config.cancel_edit") : t("mcp_config.edit_config")}
                        </Button>
                        {editMode && (
                          <Button onClick={handleSaveConfig} className="h-8 gap-1.5" size="sm">
                            <Save className="w-3.5 h-3.5" />
                            {t("mcp_config.save_config")}
                          </Button>
                        )}
                      </div>
                    </div>
                  </CardHeader>
                  <div className="relative">
                    <textarea
                      value={configJson}
                      onChange={(e) => setConfigJson(e.target.value)}
                      disabled={!editMode}
                      spellCheck={false}
                      className={`w-full h-80 p-4 font-mono text-xs leading-relaxed resize-y focus:outline-none transition-colors ${editMode
                        ? "bg-background text-foreground"
                        : "bg-muted/20 text-muted-foreground cursor-default"
                        }`}
                    />
                    {saveMessage && (
                      <div className={`absolute bottom-4 right-4 text-xs px-3 py-1.5 rounded-full shadow-sm animate-in fade-in slide-in-from-bottom-2 ${saveMessage.includes("error") || saveMessage.includes("Error") ? "bg-red-100 text-red-700 border border-red-200" : "bg-green-100 text-green-700 border border-green-200"}`}>
                        {saveMessage}
                      </div>
                    )}
                  </div>

                  {/* Sync Footer */}
                  <div className="bg-muted/10 border-t p-4">
                    <div className="flex items-center gap-2 mb-2">
                      <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t("mcp_config.sync_to_other")}</span>
                      <div className="h-px bg-border flex-1" />
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {agents
                        .filter(a => a.id !== selectedAgent && a.status === "detected")
                        .map(agent => (
                          <Button
                            key={agent.id}
                            variant="secondary"
                            size="sm"
                            onClick={() => handleCopyToAgent(agent.id)}
                            className="h-7 text-xs gap-1.5 hover:bg-primary hover:text-primary-foreground transition-colors"
                          >
                            <Copy className="w-3 h-3" />
                            {agent.name}
                          </Button>
                        ))}
                      {agents.filter(a => a.id !== selectedAgent && a.status === "detected").length === 0 && (
                        <span className="text-xs text-muted-foreground italic">{t("mcp_config.no_other_tools")}</span>
                      )}
                    </div>
                  </div>
                </Card>
              </div>
            ) : (
              <div className="h-full min-h-[400px] flex flex-col items-center justify-center text-center p-8 border-2 border-dashed rounded-xl bg-muted/10">
                <div className="bg-muted rounded-full p-4 mb-4">
                  <span className="text-4xl">👈</span>
                </div>
                <h3 className="text-lg font-medium text-foreground">{t("mcp_config.select_tool")}</h3>
                <p className="text-sm text-muted-foreground max-w-xs mt-2">
                  Select an agent from the list to view and edit its MCP configuration.
                </p>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
