import React, { useState, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/Card"
import { Button } from "./ui/Button"
import { Lock, Save, Globe } from "lucide-react"
import { useI18n } from "../i18n/useI18n"
import { validatePasswordStrength, getPasswordStrengthColor } from "../utils/passwordValidator"

interface Settings {
  githubToken: string
  gistID: string
  autoSync: boolean
  autoSyncInterval: number
  encryptionPassword: string
}

function PasswordStrengthIndicator({ password, t }: { password: string; t: any }) {
  const strength = validatePasswordStrength(password)
  const colors = {
    weak: "bg-red-100 text-red-700",
    fair: "bg-orange-100 text-orange-700",
    good: "bg-yellow-100 text-yellow-700",
    strong: "bg-lime-100 text-lime-700",
    very_strong: "bg-green-100 text-green-700",
  }

  return (
    <div className={`mt-2 p-3 rounded-md ${colors[strength.level]}`}>
      <div className="flex items-center gap-2 mb-2">
        <div className="flex gap-1">
          {[...Array(5)].map((_, i) => (
            <div
              key={i}
              className={`h-1.5 w-8 rounded ${
                i < strength.score + 1
                  ? "bg-current"
                  : "bg-gray-300"
              }`}
            />
          ))}
        </div>
        <span className="text-xs font-medium capitalize">{strength.level}</span>
      </div>
      {strength.feedback.length > 0 && (
        <ul className="text-xs space-y-1">
          {strength.feedback.map((msg, i) => (
            <li key={i}>• {msg}</li>
          ))}
        </ul>
      )}
      {!strength.isValid && (
        <p className="text-xs font-semibold mt-1">{t("settings.encryption_password_weak")}</p>
      )}
    </div>
  )
}

export function SettingsPage() {
  const { language, changeLanguage, availableLanguages, t } = useI18n()
  
  const [settings, setSettings] = useState<Settings>({
    githubToken: "",
    gistID: "",
    autoSync: false,
    autoSyncInterval: 3600,
    encryptionPassword: "",
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState("")

  useEffect(() => {
    loadSettings()
  }, [])

  const loadSettings = async () => {
    try {
      const config = await (window as any).go.main.App.GetSyncConfig()
      if (config) {
        console.log("Loaded config:", config)
        setSettings({
          githubToken: config.github_token || "",
          gistID: config.gist_id || "",
          autoSync: config.auto_sync === true, // explicitly check for true
          autoSyncInterval: config.auto_sync_interval > 0 ? config.auto_sync_interval : 3600,
          encryptionPassword: config.encryption_password || "",
        })
      }
    } catch (error) {
      console.error("Failed to load settings:", error)
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async () => {
    setSaving(true)
    setMessage("")
    
    // Validate encryption password (required)
    if (!settings.encryptionPassword) {
      setMessage(t("settings.error_password_required"))
      setSaving(false)
      return
    }
    
    // Check password strength
    const strength = validatePasswordStrength(settings.encryptionPassword)
    if (!strength.isValid) {
      setMessage(`${t("settings.error_password_weak")}: ${strength.feedback[0] || t("settings.error_password_too_weak")}`)
      setSaving(false)
      return
    }
    
    try {
      console.log("Saving settings...")
      // Get current config and update all fields
      const currentConfig = await (window as any).go.main.App.GetSyncConfig()
      console.log("Current config:", currentConfig)
      
      const updatedConfig = {
        ...currentConfig,
        github_token: settings.githubToken,
        gist_id: settings.gistID,
        auto_sync: settings.autoSync,
        auto_sync_interval: settings.autoSyncInterval,
        encryption_password: settings.encryptionPassword,
        enable_encryption: true,
      }
      console.log("Updated config:", updatedConfig)
      
      // Initialize Gist sync first (auto-creates Gist if ID is empty)
      console.log("Initializing Gist sync...")
      const createdGistID = await (window as any).go.main.App.InitializeGistSync(settings.githubToken, settings.gistID)
      console.log("Gist sync initialized, Gist ID:", createdGistID)
      
      // If a new Gist was created, update the settings with the returned ID
      if (createdGistID && !settings.gistID) {
        console.log("New Gist created, updating settings with ID:", createdGistID)
        setSettings(prev => ({ ...prev, gistID: createdGistID }))
      }
      
      // Setup encryption (now mandatory)
      console.log("Setting up encryption...")
      await (window as any).go.main.App.SetupGistEncryption(true, settings.encryptionPassword)
      console.log("Encryption setup complete")
      
      // Save all settings including autoSync
      // Update config with the actual Gist ID (in case it was auto-created)
      const finalConfig = {
        ...updatedConfig,
        gist_id: createdGistID,
      }
      console.log("Final config before saving:", finalConfig)
      console.log("Saving config to storage...")
      await (window as any).go.main.App.SaveSyncConfig(finalConfig)
      console.log("Config saved")
      
      // Reload settings to verify they were saved
      console.log("Reloading settings...")
      await loadSettings()
      console.log("Settings reloaded")
      
      setMessage(t("settings.settings_saved"))
      setTimeout(() => setMessage(""), 3000)
    } catch (error) {
      console.error("Save settings error:", error)
      let errorMsg = t("common.error")
      if (error instanceof Error) {
        errorMsg = error.message
      } else if (typeof error === "string") {
        errorMsg = error
      }
      console.log("Final error:", errorMsg)
      setMessage(`${t("common.error")}: ${errorMsg}`)
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <div className="text-center py-8">{t("common.loading")}</div>
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">{t("settings.title")}</h2>
        <p className="text-muted-foreground mt-1">{t("settings.subtitle")}</p>
      </div>

      {/* Language Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="w-5 h-5" />
            {t("settings.language")}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.select_language")}
            </label>
            <select
              value={language}
              onChange={(e) => changeLanguage(e.target.value as any)}
              className="w-full px-3 py-2 border border-input bg-background rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            >
              {availableLanguages.map((lang) => (
                <option key={lang.code} value={lang.code}>
                  {lang.name}
                </option>
              ))}
            </select>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Lock className="w-5 h-5" />
            {t("settings.github_integration")}
          </CardTitle>
          <CardDescription>
            {t("settings.github_token_description")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* GitHub Token */}
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.github_token")}
            </label>
            <input
              type="password"
              value={settings.githubToken}
              onChange={(e) => setSettings({ ...settings, githubToken: e.target.value })}
              placeholder={t("settings.github_token_placeholder")}
              className="w-full px-3 py-2 border border-input bg-background rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            />
            <p className="text-xs text-muted-foreground mt-1">
              {t("settings.github_token_help")}
            </p>
          </div>

          {/* Gist ID */}
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.gist_id")}
            </label>
            <input
              type="text"
              value={settings.gistID}
              onChange={(e) => setSettings({ ...settings, gistID: e.target.value })}
              placeholder={t("settings.gist_id_placeholder")}
              className="w-full px-3 py-2 border border-input bg-background rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>

          {/* Encryption Settings - REQUIRED */}
          <div className="space-y-3 border-t pt-4 bg-amber-50 p-4 rounded-lg">
            <div className="flex items-start gap-2">
              <Lock className="w-4 h-4 text-amber-600 mt-0.5" />
              <div className="flex-1">
                <p className="text-sm font-semibold text-amber-900">{t("settings.encryption_required")}</p>
                <p className="text-xs text-amber-700 mt-1">
                  {t("settings.encryption_required_description")}
                </p>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">
                {t("settings.encryption_password")} <span className="text-red-500">{t("settings.encryption_password_required")}</span>
              </label>
              <input
                type="password"
                value={settings.encryptionPassword}
                onChange={(e) => setSettings({ ...settings, encryptionPassword: e.target.value })}
                placeholder={t("settings.encryption_password_placeholder")}
                className="w-full px-3 py-2 border border-input bg-background rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-ring"
              />
              {settings.encryptionPassword && (
                <PasswordStrengthIndicator password={settings.encryptionPassword} t={t} />
              )}
              <p className="text-xs text-muted-foreground mt-2">
                {t("settings.encryption_password_warning")}
              </p>
            </div>
          </div>

          {/* Auto Sync */}
          <div className="space-y-3">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={settings.autoSync}
                onChange={(e) => setSettings({ ...settings, autoSync: e.target.checked })}
                className="w-4 h-4"
              />
              <span className="text-sm font-medium">{t("settings.auto_sync_label")}</span>
            </label>
            {settings.autoSync && (
              <div>
                <label className="block text-sm font-medium mb-2">
                  {t("settings.auto_sync_interval_label")}
                </label>
                <input
                  type="number"
                  value={settings.autoSyncInterval}
                  onChange={(e) => setSettings({ ...settings, autoSyncInterval: parseInt(e.target.value) || 3600 })}
                  min="60"
                  className="w-full px-3 py-2 border border-input bg-background rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-ring"
                />
              </div>
            )}
          </div>

          {/* Status Message */}
          {message && (
            <div className={`p-3 rounded-md text-sm ${message.startsWith("Error") ? "bg-destructive/10 text-destructive" : "bg-green-100 text-green-700"}`}>
              {message}
            </div>
          )}

          {/* Save Button */}
          <Button 
            onClick={handleSave}
            disabled={saving || !settings.githubToken || !settings.encryptionPassword || !validatePasswordStrength(settings.encryptionPassword).isValid}
            className="w-full gap-2"
          >
            <Save className="w-4 h-4" />
            {saving ? t("settings.save_button_saving") : t("settings.save_button")}
          </Button>
        </CardContent>
      </Card>

      {/* About Section */}
      <Card>
        <CardHeader>
          <CardTitle>{t("settings.about_title")}</CardTitle>
        </CardHeader>
        <CardContent className="text-sm space-y-2 text-muted-foreground">
          <p>{t("settings.about_description")}</p>
          <p>{t("settings.about_supported")}</p>
          <p>{t("settings.about_secure")}</p>
        </CardContent>
      </Card>
    </div>
  )
}
