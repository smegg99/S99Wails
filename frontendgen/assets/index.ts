// index.ts
export { isWails, useWailsEnvironment } from './environment'
export { useDesktopLocaleSync, type DesktopLocaleSyncOptions } from './localeSync'
export { useSystemTheme, THEME_CHANGED_EVENT, type SystemThemeInfo, type SystemThemeOptions } from './systemTheme'
export { installFrontendLogBridge, type FrontendLogSink } from './logBridge'
