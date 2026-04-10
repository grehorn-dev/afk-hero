import { create } from 'zustand';
import type { Settings, NumericRange } from '../types/settings';
import { DEFAULT_SETTINGS } from '../types/settings';

interface SettingsStore {
  // Draft form state
  draft: Settings;
  // Applied state (last known from backend)
  applied: Settings;
  // Dirty tracking
  isDirty: boolean;

  // Actions
  setDraft: (update: Partial<Settings>) => void;
  setDraftRange: (field: keyof Pick<Settings, 'distance' | 'interval' | 'speed' | 'inactivity'>, update: Partial<NumericRange>) => void;
  initFromApplied: (settings: Settings) => void;
  setApplied: (settings: Settings) => void;
  markClean: () => void;
  resetToDefaults: () => void;
  cancelChanges: () => void;
}

function settingsEqual(a: Settings, b: Settings): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

export const useSettingsStore = create<SettingsStore>((set) => ({
  draft: { ...DEFAULT_SETTINGS },
  applied: { ...DEFAULT_SETTINGS },
  isDirty: false,

  setDraft: (update) => {
    set((s) => {
      const newDraft = { ...s.draft, ...update };
      return { draft: newDraft, isDirty: !settingsEqual(newDraft, s.applied) };
    });
  },

  setDraftRange: (field, update) => {
    set((s) => {
      const current = s.draft[field] as NumericRange;
      const newRange = { ...current, ...update };
      const newDraft = { ...s.draft, [field]: newRange };
      return { draft: newDraft, isDirty: !settingsEqual(newDraft, s.applied) };
    });
  },

  initFromApplied: (settings) => {
    set({ draft: { ...settings }, applied: { ...settings }, isDirty: false });
  },

  setApplied: (settings) => {
    set((s) => ({
      applied: { ...settings },
      isDirty: !settingsEqual(s.draft, settings),
    }));
  },

  markClean: () => {
    set((s) => ({ applied: { ...s.draft }, isDirty: false }));
  },

  resetToDefaults: () => {
    set((s) => {
      const defaults = {
        ...DEFAULT_SETTINGS,
        language: s.draft.language,
        theme: s.draft.theme,
      };
      return { draft: defaults, isDirty: !settingsEqual(defaults, s.applied) };
    });
  },

  cancelChanges: () => {
    set((s) => ({ draft: { ...s.applied }, isDirty: false }));
  },
}));
