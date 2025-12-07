import React, { useState } from "react"
import { Button } from "./ui/Button"
import { AlertTriangle, Check } from "lucide-react"
import { ConflictDetails, ServerConflict } from "../types/models"

interface ConflictResolverProps {
    details: ConflictDetails
    onResolve: (decisions: Record<string, string>) => void
    onCancel: () => void
}

export function ConflictResolver({ details, onResolve, onCancel }: ConflictResolverProps) {
    const [decisions, setDecisions] = useState<Record<string, string>>({})

    // Compute stats (with null safety)
    const agents = details.agents || {}
    const totalConflicts = Object.values(agents).reduce((sum, agent) => sum + (agent.conflicts?.length || 0), 0)
    const resolvedCount = Object.keys(decisions).length
    const progress = totalConflicts > 0 ? (resolvedCount / totalConflicts) * 100 : 100

    const handleDecision = (agentId: string, serverId: string, choice: "local" | "remote") => {
        setDecisions(prev => ({
            ...prev,
            [`${agentId}:${serverId}`]: choice
        }))
    }

    const handleSubmit = () => {
        onResolve(decisions)
    }

    return (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div className="bg-card w-full max-w-5xl max-h-[90vh] flex flex-col rounded-xl border border-border shadow-2xl">

                {/* Header */}
                <div className="p-6 border-b border-border flex items-center justify-between">
                    <div className="flex items-center gap-3">
                        <div className="p-3 bg-yellow-100 rounded-full text-yellow-600">
                            <AlertTriangle className="w-6 h-6" />
                        </div>
                        <div>
                            <h2 className="text-xl font-bold">Sync Conflicts Detected</h2>
                            <p className="text-sm text-muted-foreground">Some configurations differ between local and cloud.</p>
                        </div>
                    </div>
                    <div className="text-right">
                        <span className="text-sm font-medium text-muted-foreground block mb-1">
                            Resolution Progress: {resolvedCount} / {totalConflicts}
                        </span>
                        <div className="w-32 h-2 bg-secondary rounded-full overflow-hidden">
                            <div className="h-full bg-primary transition-all duration-300" style={{ width: `${progress}%` }} />
                        </div>
                    </div>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-auto p-6 space-y-8">
                    {Object.values(agents).map((agent) => (

                        ((agent.conflicts?.length || 0) > 0 || (agent.local_only?.length || 0) > 0 || (agent.remote_only?.length || 0) > 0) && (
                            <div key={agent.agent_id} className="space-y-4">
                                <h3 className="text-lg font-semibold flex items-center gap-2">
                                    <span className="w-2 h-8 bg-primary rounded-full" />
                                    Agent: {agent.agent_id}
                                </h3>

                                {/* Conflict List */}
                                <div className="space-y-4">
                                    {(agent.conflicts ?? []).map((conflict) => (
                                        <ConflictItem
                                            key={conflict.server_id}
                                            agentId={agent.agent_id}
                                            conflict={conflict}
                                            decision={decisions[`${agent.agent_id}:${conflict.server_id}`]}
                                            onDecide={handleDecision}
                                        />
                                    ))}

                                    {/* Local Only Items (Informational) */}
                                    {(agent.local_only ?? []).length > 0 && (
                                        <div className="bg-secondary/30 p-4 rounded-lg border border-border/50">
                                            <h4 className="text-sm font-medium mb-2 flex items-center gap-2 text-green-600">
                                                <Check className="w-4 h-4" />
                                                New Local Items (Will be added to Cloud)
                                            </h4>
                                            <div className="flex flex-wrap gap-2">
                                                {(agent.local_only ?? []).map(s => (
                                                    <span key={s.id} className="px-2 py-1 bg-background rounded border border-border text-xs font-mono">
                                                        {s.name}
                                                    </span>
                                                ))}
                                            </div>
                                        </div>
                                    )}

                                    {/* Remote Only Items (Informational) */}
                                    {(agent.remote_only ?? []).length > 0 && (
                                        <div className="bg-secondary/30 p-4 rounded-lg border border-border/50">
                                            <h4 className="text-sm font-medium mb-2 flex items-center gap-2 text-blue-600">
                                                <Check className="w-4 h-4" />
                                                New Cloud Items (Will be added to Local)
                                            </h4>
                                            <div className="flex flex-wrap gap-2">
                                                {(agent.remote_only ?? []).map(s => (
                                                    <span key={s.id} className="px-2 py-1 bg-background rounded border border-border text-xs font-mono">
                                                        {s.name}
                                                    </span>
                                                ))}
                                            </div>
                                        </div>
                                    )}
                                </div>
                            </div>
                        )
                    ))}
                </div>

                {/* Footer */}
                <div className="p-6 border-t border-border bg-secondary/10 flex justify-end gap-3">
                    <Button variant="outline" onClick={onCancel}>Cancel Sync</Button>
                    <Button onClick={handleSubmit} disabled={resolvedCount < totalConflicts}>
                        Apply Resolutions & Sync
                    </Button>
                </div>

            </div>
        </div>
    )
}

function ConflictItem({ agentId, conflict, decision, onDecide }: {
    agentId: string
    conflict: ServerConflict
    decision?: string
    onDecide: (agentId: string, serverId: string, choice: "local" | "remote") => void
}) {
    const isDiffCommand = conflict.local.command !== conflict.remote.command
    const isDiffArgs = JSON.stringify(conflict.local.args) !== JSON.stringify(conflict.remote.args)
    const isDiffEnv = JSON.stringify(conflict.local.env) !== JSON.stringify(conflict.remote.env)

    return (
        <div className={`border rounded-lg overflow-hidden transition-all ${decision ? 'border-primary/50 ring-1 ring-primary/20' : 'border-border'}`}>

            {/* Item Header */}
            <div className="bg-secondary/40 p-3 flex items-center justify-between border-b border-border">
                <div className="font-mono font-medium">{conflict.server_id}</div>
                <div className="text-xs text-muted-foreground uppercase tracking-wider">Modified in both</div>
            </div>

            <div className="grid grid-cols-2 divide-x divide-border">

                {/* Local Option */}
                <div
                    onClick={() => onDecide(agentId, conflict.server_id, "local")}
                    className={`p-4 cursor-pointer hover:bg-secondary/20 transition-colors relative group ${decision === 'local' ? 'bg-primary/5' : ''}`}
                >
                    <div className="flex items-center justify-between mb-3">
                        <span className="text-xs font-bold uppercase text-muted-foreground">Local Version</span>
                        {decision === 'local' && <Check className="w-5 h-5 text-primary" />}
                    </div>

                    <div className="space-y-2 text-sm font-mono text-xs">
                        {isDiffCommand && (
                            <div className="break-all">
                                <span className="text-muted-foreground block mb-0.5">Command:</span>
                                <span className="bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 px-1 rounded">{conflict.local.command}</span>
                            </div>
                        )}
                        {isDiffArgs && (
                            <div className="break-all">
                                <span className="text-muted-foreground block mb-0.5">Args:</span>
                                <span className="bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 px-1 rounded">
                                    {JSON.stringify(conflict.local.args)}
                                </span>
                            </div>
                        )}
                        {isDiffEnv && (
                            <div className="break-all">
                                <span className="text-muted-foreground block mb-0.5">Env:</span>
                                <div className="bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 p-1 rounded whitespace-pre-wrap">
                                    {JSON.stringify(conflict.local.env, null, 2)}
                                </div>
                            </div>
                        )}
                    </div>
                </div>

                {/* Remote Option */}
                <div
                    onClick={() => onDecide(agentId, conflict.server_id, "remote")}
                    className={`p-4 cursor-pointer hover:bg-secondary/20 transition-colors relative group ${decision === 'remote' ? 'bg-primary/5' : ''}`}
                >
                    <div className="flex items-center justify-between mb-3">
                        <span className="text-xs font-bold uppercase text-muted-foreground">Remote Version</span>
                        {decision === 'remote' && <Check className="w-5 h-5 text-primary" />}
                    </div>

                    <div className="space-y-2 text-sm font-mono text-xs">
                        {isDiffCommand && (
                            <div className="break-all">
                                <span className="text-muted-foreground block mb-0.5">Command:</span>
                                <span className="bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 px-1 rounded">{conflict.remote.command}</span>
                            </div>
                        )}
                        {isDiffArgs && (
                            <div className="break-all">
                                <span className="text-muted-foreground block mb-0.5">Args:</span>
                                <span className="bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 px-1 rounded">
                                    {JSON.stringify(conflict.remote.args)}
                                </span>
                            </div>
                        )}
                        {isDiffEnv && (
                            <div className="break-all">
                                <span className="text-muted-foreground block mb-0.5">Env:</span>
                                <div className="bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 p-1 rounded whitespace-pre-wrap">
                                    {JSON.stringify(conflict.remote.env, null, 2)}
                                </div>
                            </div>
                        )}
                    </div>
                </div>

            </div>
        </div>
    )
}
