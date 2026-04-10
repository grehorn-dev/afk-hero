import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

// RTL language codes
const RTL_LANGUAGES = new Set(['arb', 'heb', 'urd', 'fas', 'pus']);

export function isRTL(code: string): boolean {
  return RTL_LANGUAGES.has(code);
}

export function updateDirection(code: string): void {
  document.documentElement.dir = isRTL(code) ? 'rtl' : 'ltr';
}

// Initialize i18next with empty resources - they'll be loaded from Go backend
i18n.use(initReactI18next).init({
  resources: {},
  lng: 'eng',
  fallbackLng: 'eng',
  interpolation: {
    escapeValue: false,
  },
  react: {
    useSuspense: false,
  },
});

export async function loadLanguage(code: string, translations: Record<string, string>): Promise<void> {
  i18n.addResourceBundle(code, 'translation', translations, true, true);
}

export async function changeLanguage(code: string): Promise<void> {
  await i18n.changeLanguage(code);
  updateDirection(code);
}

export default i18n;
