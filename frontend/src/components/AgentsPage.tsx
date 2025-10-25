import React, { useEffect, useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/Card"
import { Button } from "./ui/Button"
import { CheckCircle2, AlertCircle, Copy, Save } from "lucide-react"
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
        <div className="text-center py-8">
          <p className="text-muted-foreground">{t("mcp_config.detecting")}</p>
        </div>
      ) : (
        <div className="space-y-6">
          {/* 工具切换 */}
          <div>
            <h3 className="text-sm font-medium mb-3">{t("mcp_config.select_tool")}</h3>
            <div className="flex flex-wrap gap-2">
              {agents.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t("mcp_config.no_tools_detected")}</p>
              ) : (
                agents.map(agent => (
                  <button
                    key={agent.id}
                    onClick={() => handleAgentSelect(agent.id)}
                    className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-all ${
                      selectedAgent === agent.id
                        ? "bg-primary text-white"
                        : "bg-secondary hover:bg-secondary/80 text-foreground"
                    }`}
                  >
                    {agent.status === "detected" ? (
                      <CheckCircle2 className="w-4 h-4" aria-label={t("mcp_config.status_detected")} />
                    ) : (
                      <AlertCircle className="w-4 h-4" aria-label={t("mcp_config.status_not_installed")} />
                    )}
                    <span className="font-medium">{agent.name}</span>
                  </button>
                ))
              )}
            </div>
          </div>

          {/* 配置编辑 */}
          <div>
            {currentAgent ? (
              <>
                {/* 工具信息 */}
                <Card className="mb-6">
                  <CardHeader className="pb-3">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <CardTitle>{currentAgent.name}</CardTitle>
                        <CardDescription className="mt-1">
                          {currentAgent.status === "detected" ? (
                            <span className="text-green-600 flex items-center gap-1">
                              <CheckCircle2 className="w-4 h-4" />{t("mcp_config.status_detected")}
                            </span>
                          ) : (
                            <span className="text-amber-600 flex items-center gap-1">
                              <AlertCircle className="w-4 h-4" />{t("mcp_config.status_not_installed")}
                            </span>
                          )}
                        </CardDescription>
                      </div>
                      <span className="text-xs bg-muted px-2 py-1 rounded whitespace-nowrap">{currentAgent.platform}</span>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4 pt-0">
                    <div>
                      <p className="text-sm font-medium mb-2">{t("mcp_config.config_paths")}</p>
                      {currentAgent.existing_paths && currentAgent.existing_paths.length > 0 ? (
                        <div className="space-y-2">
                          <div className="space-y-1">
                            <p className="text-xs text-green-600 font-medium">{t("mcp_config.detected_paths")}</p>
                            {currentAgent.existing_paths.map((path, idx) => (
                              <li key={idx} className="text-xs font-mono bg-green-50 border border-green-200 p-2 rounded break-all text-green-700 list-none">
                                ✓ {path}
                              </li>
                            ))}
                          </div>
                          {currentAgent.configPaths && currentAgent.configPaths.length > currentAgent.existing_paths.length && (
                            <div className="space-y-1 pt-2">
                              <p className="text-xs text-amber-600 font-medium">{t("mcp_config.other_possible_paths")}</p>
                              {currentAgent.configPaths.filter(p => !currentAgent.existing_paths.includes(p)).map((path, idx) => (
                                <li key={idx} className="text-xs font-mono bg-amber-50 border border-amber-200 p-2 rounded break-all text-amber-600 list-none">
                                  ○ {path}
                                </li>
                              ))}
                            </div>
                          )}
                        </div>
                      ) : (
                        <div className="space-y-1">
                          <p className="text-xs text-muted-foreground font-medium mb-2">{t("mcp_config.auto_detected_locations")}</p>
                          {currentAgent.configPaths && currentAgent.configPaths.length > 0 ? (
                            currentAgent.configPaths.map((path, idx) => (
                              <li key={idx} className="text-xs font-mono bg-muted p-2 rounded break-all text-muted-foreground list-none">
                                ○ {path}
                              </li>
                            ))
                          ) : (
                            <p className="text-xs text-muted-foreground italic">No configuration paths found</p>
                          )}
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>

                {/* JSON编辑器 */}
                <Card>
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <div>
                        <CardTitle>{t("mcp_config.mcp_config_title")}</CardTitle>
                        <CardDescription>{t("mcp_config.mcp_config_subtitle")}</CardDescription>
                      </div>
                      <Button
                        variant={editMode ? "default" : "outline"}
                        size="sm"
                        onClick={() => setEditMode(!editMode)}
                      >
                        {editMode ? t("mcp_config.cancel_edit") : t("mcp_config.edit_config")}
                      </Button>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <textarea
                      value={configJson}
                      onChange={(e) => setConfigJson(e.target.value)}
                      disabled={!editMode}
                      className={`w-full h-64 p-4 font-mono text-sm rounded-lg border ${
                        editMode
                          ? "bg-input text-foreground border-input"
                          : "bg-muted text-muted-foreground border-border cursor-not-allowed"
                      }`}
                    />

                    {editMode && (
                      <div className="flex gap-2">
                        <Button onClick={handleSaveConfig} className="gap-2" size="sm">
                          <Save className="w-4 h-4" />
                          {t("mcp_config.save_config")}
                        </Button>
                      </div>
                    )}

                    {saveMessage && (
                      <p className={`text-sm ${saveMessage.includes("error") || saveMessage.includes("Error") ? "text-red-600" : "text-green-600"}`}>
                        {saveMessage}
                      </p>
                    )}

                    {/* 同步到其他工具 */}
                    <div className="pt-4 border-t">
                      <p className="text-sm font-medium mb-3">{t("mcp_config.sync_to_other")}</p>
                      <div className="flex flex-wrap gap-2">
                        {agents
                          .filter(a => a.id !== selectedAgent && a.status === "detected")
                          .map(agent => (
                            <Button
                              key={agent.id}
                              variant="outline"
                              size="sm"
                              onClick={() => handleCopyToAgent(agent.id)}
                              className="gap-2"
                            >
                              <Copy className="w-4 h-4" />
                              {tReplace("mcp_config.sync_to_tool", { tool: agent.name })}
                            </Button>
                          ))}
                      </div>
                      {agents.filter(a => a.id !== selectedAgent && a.status === "detected").length === 0 && (
                        <p className="text-sm text-muted-foreground">{t("mcp_config.no_other_tools")}</p>
                      )}
                    </div>
                  </CardContent>
                </Card>
              </>
            ) : (
              <Card>
                <CardContent className="pt-6">
                  <p className="text-muted-foreground">Please select a tool</p>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
