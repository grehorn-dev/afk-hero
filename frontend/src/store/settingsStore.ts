import { create } from 'zustand';
import type { Settings, NumericRange } from '../types/settings';

interface SettingsStore {
  ready: boolean;
  draft: Settings;
  applied: Settings;
  defaults: Settings;
  isDirty: boolean;

  setDraft: (update: Partial<Settings>) => void;
  setDraftRange: (field: keyof Pick<Settings, 'distance' | 'interval' | 'speed' | 'inactivity'>, update: Partial<NumericRange>) => void;
  initFromApplied: (settings: Settings, defaults?: Settings) => void;
  setApplied: (settings: Settings) => void;
  markClean: () => void;
  resetToDefaults: () => void;
  cancelChanges: () => void;
}

const PLACEHOLDER: Settings = {} as Settings;

function settingsEqual(a: Settings, b: Settings): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

export const useSettingsStore = create<SettingsStore>((set) => ({
  ready: false,
  draft: PLACEHOLDER,
  applied: PLACEHOLDER,
  defaults: PLACEHOLDER,
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

  initFromApplied: (settings, defaults) => {
    const update: Partial<SettingsStore> = {
      ready: true,
      draft: { ...settings },
      applied: { ...settings },
      isDirty: false,
    };
    if (defaults) {
      update.defaults = { ...defaults };
    }
    set(update);
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
      const reset = {
        ...s.defaults,
        language: s.draft.language,
        theme: s.draft.theme,
      };
      return { draft: reset, isDirty: !settingsEqual(reset, s.applied) };
    });
  },

  cancelChanges: () => {
    set((s) => ({ draft: { ...s.applied }, isDirty: false }));
  },
}));
