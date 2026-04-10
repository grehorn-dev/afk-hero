import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Language, Theme } from '../types/settings';
import { THEMES } from '../types/settings';

interface Props {
  language: string;
  theme: Theme;
  onLanguageChange: (code: string) => void;
  onThemeChange: (theme: Theme) => void;
}

export function AppearanceSelect({ language, theme, onLanguageChange, onThemeChange }: Props) {
  const { t } = useTranslation();
  const [languages, setLanguages] = useState<Language[]>([]);

  useEffect(() => {
    window.go.app.App.GetSupportedLanguages().then(setLanguages).catch(console.error);
  }, []);

  return (
    <div className="form-row">
      <label className="form-label">{t('settings.appearance')}</label>
      <div className="appearance-controls">
        <select
          className="form-select appearance-language-select"
          value={language}
          onChange={(e) => onLanguageChange(e.target.value)}
        >
          {languages.map((lang) => (
            <option key={lang.code} value={lang.code}>
              {lang.name}
            </option>
          ))}
        </select>

        <select
          className="form-select appearance-theme-select"
          value={theme}
          onChange={(e) => onThemeChange(e.target.value as Theme)}
        >
          {THEMES.map((option) => (
            <option key={option} value={option}>
              {t(`theme.${option}`)}
            </option>
          ))}
        </select>
      </div>
    </div>
  );
}
