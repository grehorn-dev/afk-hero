import { WindowSetDarkTheme, WindowSetLightTheme } from '../wailsjs/runtime/runtime';
import type { Theme } from '../types/settings';

const themeAttributes: Record<Theme, 'dark' | 'light'> = {
  Dark: 'dark',
  Light: 'light',
};

export function applyTheme(theme: Theme): void {
  const attribute = themeAttributes[theme] ?? 'dark';

  document.documentElement.dataset.theme = attribute;
  document.documentElement.style.colorScheme = attribute;

  if (theme === 'Light') {
    WindowSetLightTheme();
    return;
  }

  WindowSetDarkTheme();
}
