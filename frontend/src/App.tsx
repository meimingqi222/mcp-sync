import React, { useState } from "react"
import "./globals.css"
import { Dashboard } from "./components/Dashboard"
import { AgentsPage } from "./components/AgentsPage"
import { SettingsPage } from "./components/SettingsPage"
import { Menu, X, Cloud } from "lucide-react"
import { useI18n } from "./i18n/useI18n"

export default function App() {
  const [currentPage, setCurrentPage] = useState("dashboard")
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const { t } = useI18n()

  const menuItems = [
    { id: "dashboard", label: t("menu.dashboard"), icon: "📊" },
    { id: "agents", label: t("menu.mcp_config"), icon: "🔗" },
    { id: "settings", label: t("menu.settings"), icon: "🔧" },
  ]

  const renderPage = () => {
    switch (currentPage) {
      case "agents":
        return <AgentsPage />
      case "settings":
        return <SettingsPage />
      default:
        return <Dashboard />
    }
  }

  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar */}
      <div className={`${sidebarOpen ? "w-64" : "w-20"} bg-card border-r border-border transition-all duration-300 flex flex-col`}>
        {/* Logo */}
        <div className="p-4 border-b border-border">
          <div className="flex items-center gap-3">
            <Cloud className="w-6 h-6 text-primary" />
            {sidebarOpen && <span className="font-bold text-lg">{t("app.title")}</span>}
          </div>
        </div>

        {/* Menu Items */}
        <nav className="flex-1 p-4 space-y-2">
          {menuItems.map((item) => (
            <button
              key={item.id}
              onClick={() => setCurrentPage(item.id)}
              className={`w-full flex items-center gap-3 px-4 py-2 rounded-md transition-colors ${
                currentPage === item.id
                  ? "bg-primary text-primary-foreground"
                  : "hover:bg-accent text-foreground"
              }`}
              title={item.label}
            >
              <span className="text-xl">{item.icon}</span>
              {sidebarOpen && <span className="text-sm">{item.label}</span>}
            </button>
          ))}
        </nav>

        {/* Toggle Button */}
        <div className="p-4 border-t border-border">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="w-full p-2 hover:bg-accent rounded-md transition-colors"
            title={sidebarOpen ? "Collapse" : "Expand"}
          >
            {sidebarOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <div className="h-16 bg-card border-b border-border flex items-center px-6">
          <h1 className="text-lg font-semibold">
            {menuItems.find(m => m.id === currentPage)?.label}
          </h1>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-6">
          <div className="max-w-6xl mx-auto">
            {renderPage()}
          </div>
        </div>
      </div>
    </div>
  )
}
