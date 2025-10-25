import { useState, useEffect, useCallback } from 'react'
import { i18n, Language } from './index'

export function useI18n() {
  const [language, setLanguage] = useState<Language>(i18n.getLanguage())

  useEffect(() => {
    // 监听语言变化事件
    const handleLanguageChange = (event: Event) => {
      const customEvent = event as CustomEvent<Language>
      setLanguage(customEvent.detail)
    }

    window.addEventListener('languageChange', handleLanguageChange)
    return () => {
      window.removeEventListener('languageChange', handleLanguageChange)
    }
  }, [])

  const t = useCallback((key: string, defaultValue?: string): string => {
    return i18n.t(key, defaultValue)
  }, [])

  const tReplace = useCallback(
    (key: string, vars: Record<string, string>, defaultValue?: string): string => {
      return i18n.tReplace(key, vars, defaultValue)
    },
    []
  )

  const changeLanguage = useCallback((lang: Language) => {
    i18n.setLanguage(lang)
    setLanguage(lang)
  }, [])

  return {
    language,
    t,
    tReplace,
    changeLanguage,
    availableLanguages: i18n.getAvailableLanguages(),
  }
}
