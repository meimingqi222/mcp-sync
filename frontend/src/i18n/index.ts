import en from '../locales/en.json'
import zh from '../locales/zh.json'

export type Language = 'en' | 'zh'

const messages = {
  en,
  zh,
}

export class I18n {
  private currentLanguage: Language = 'en'

  constructor() {
    const savedLanguage = localStorage.getItem('language') as Language | null
    if (savedLanguage && this.isValidLanguage(savedLanguage)) {
      this.currentLanguage = savedLanguage
    } else {
      // 尝试使用浏览器语言
      this.setLanguageFromBrowser()
    }
  }

  private isValidLanguage(lang: string): lang is Language {
    return ['en', 'zh'].includes(lang)
  }

  private setLanguageFromBrowser() {
    const browserLang = navigator.language.split('-')[0]
    if (this.isValidLanguage(browserLang)) {
      this.currentLanguage = browserLang
    }
  }

  getLanguage(): Language {
    return this.currentLanguage
  }

  setLanguage(lang: Language): void {
    if (this.isValidLanguage(lang)) {
      this.currentLanguage = lang
      localStorage.setItem('language', lang)
      // 触发语言变化事件
      window.dispatchEvent(new CustomEvent('languageChange', { detail: lang }))
    }
  }

  t(key: string, defaultValue?: string): string {
    const keys = key.split('.')
    let value: any = messages[this.currentLanguage]

    for (const k of keys) {
      if (value && typeof value === 'object') {
        value = value[k]
      } else {
        return defaultValue || key
      }
    }

    return typeof value === 'string' ? value : defaultValue || key
  }

  // 支持模板变量替换
  tReplace(key: string, vars: Record<string, string>, defaultValue?: string): string {
    let text = this.t(key, defaultValue)
    for (const [varName, varValue] of Object.entries(vars)) {
      text = text.replace(`{${varName}}`, varValue)
    }
    return text
  }

  getAvailableLanguages(): { code: Language; name: string }[] {
    return [
      { code: 'en', name: 'English' },
      { code: 'zh', name: '中文' },
    ]
  }
}

// 创建全局实例
export const i18n = new I18n()
