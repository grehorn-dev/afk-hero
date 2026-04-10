import { useEffect } from 'react';
import { SettingsForm } from './components/SettingsForm';
import { useAppStore } from './store/appStore';
import { useSettingsStore } from './store/settingsStore';
import { loadLanguage, changeLanguage } from './i18n/i18n';
import { applyTheme } from './theme/theme';
import type { ActivationRuntimeState, RuntimeState } from './types/state';
import type { Settings } from './types/settings';

// Wails runtime bindings
declare global {
  interface Window {
    go: {
      app: {
        App: {
          GetSettings(): Promise<Settings>;
          ApplySettings(s: Settings): Promise<void>;
          SetEnabled(enabled: boolean): Promise<void>;
          GetTranslations(code: string): Promise<Record<string, string>>;
          GetSupportedLanguages(): Promise<Array<{ code: string; name: string }>>;
          GetWindowList(): Promise<Array<{ id: number; title: string; className: string; processName: string; autoCandidate: boolean }>>;
          GetState(): Promise<RuntimeState>;
          GetActivationState(): Promise<ActivationRuntimeState>;
          ShowWindow(): Promise<void>;
          SyncWindowLayout(advanced: boolean, activationEnabled: boolean): Promise<void>;
        };
      };
    };
    runtime: {
      EventsOn(event: string, callback: (...args: unknown[]) => void): () => void;
      WindowHide(): void;
    };
  }
}

function App() {
  const setRuntimeState = useAppStore((s) => s.setRuntimeState);
  const setActivationState = useAppStore((s) => s.setActivationState);
  const initFromApplied = useSettingsStore((s) => s.initFromApplied);
  const setApplied = useSettingsStore((s) => s.setApplied);

  useEffect(() => {
    async function init() {
      try {
        // Load settings from backend
        const settings = await window.go.app.App.GetSettings();
        initFromApplied(settings);

        // Load translations
        const translations = await window.go.app.App.GetTranslations(settings.language);
        await loadLanguage(settings.language, translations);

        // Also load English as fallback
        if (settings.language !== 'eng') {
          const engTranslations = await window.go.app.App.GetTranslations('eng');
          await loadLanguage('eng', engTranslations);
        }

        await changeLanguage(settings.language);
        applyTheme(settings.theme);

        // Get initial state
        const state = await window.go.app.App.GetState();
        setRuntimeState(state);

        const activationState = await window.go.app.App.GetActivationState();
        setActivationState(activationState);
      } catch (e) {
        console.error('Initialization failed:', e);
      }
    }

    init();

    // Listen for state changes from backend
    const unsubState = window.runtime?.EventsOn('state:changed', (...args: unknown[]) => {
      const data = args[0] as RuntimeState;
      if (data) {
        setRuntimeState(data);
      }
    });

    const unsubActivation = window.runtime?.EventsOn('activation:changed', (...args: unknown[]) => {
      const data = args[0] as ActivationRuntimeState;
      if (data) {
        setActivationState(data);
      }
    });

    // Listen for settings changes from backend (e.g., tray toggle)
    const unsubSettings = window.runtime?.EventsOn('settings:changed', (...args: unknown[]) => {
      const data = args[0] as Settings;
      if (data) {
        setApplied(data);
      }
    });

    return () => {
      unsubState?.();
      unsubActivation?.();
      unsubSettings?.();
    };
  }, [initFromApplied, setActivationState, setApplied, setRuntimeState]);

  return <SettingsForm />;
}

export default App;
