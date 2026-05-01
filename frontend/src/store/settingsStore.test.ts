import { beforeEach, describe, expect, it } from 'vitest';
import type { Settings } from '../types/settings';
import { useSettingsStore } from './settingsStore';

const TEST_DEFAULTS: Settings = {
  enabled: true,
  advanced: false,
  logging: false,
  language: 'eng',
  theme: 'Dark',
  shape: 'Random',
  direction: 'Random',
  distance: { mode: 'Random', value: 150, minVal: 150, maxVal: 300 },
  interval: { mode: 'Random', value: 25, minVal: 25, maxVal: 35 },
  speed: { mode: 'Random', value: 1, minVal: 1, maxVal: 2 },
  inactivity: { mode: 'Fixed', value: 60, minVal: 60, maxVal: 120 },
  activationEnabled: false,
  activationMode: 'Auto',
  activationTimeout: 239,
  activationTargetWindowOnly: false,
  targetWindowTitle: '',
  targetWindowClass: '',
  targetProcessName: '',
};

describe('settingsStore', () => {
  beforeEach(() => {
    useSettingsStore.setState({
      draft: { ...TEST_DEFAULTS },
      applied: { ...TEST_DEFAULTS },
      defaults: { ...TEST_DEFAULTS },
      isDirty: false,
    });
  });

  it('marks draft as dirty when a field changes', () => {
    useSettingsStore.getState().setDraft({ language: 'rus' });

    const state = useSettingsStore.getState();
    expect(state.draft.language).toBe('rus');
    expect(state.isDirty).toBe(true);
  });

  it('tracks theme changes independently from language', () => {
    useSettingsStore.getState().setDraft({ theme: 'Light' });

    const state = useSettingsStore.getState();
    expect(state.draft.theme).toBe('Light');
    expect(state.isDirty).toBe(true);
  });

  it('tracks advanced toggle changes independently from advanced field values', () => {
    useSettingsStore.getState().setDraft({ advanced: true });

    const state = useSettingsStore.getState();
    expect(state.draft.advanced).toBe(true);
    expect(state.isDirty).toBe(true);
  });

  it('resets the draft to defaults while preserving the current language and theme', () => {
    const store = useSettingsStore.getState();
    store.setDraft({ language: 'rus', theme: 'Light', enabled: false });
    store.resetToDefaults();

    const state = useSettingsStore.getState();
    expect(state.draft).toEqual({ ...TEST_DEFAULTS, language: 'rus', theme: 'Light' });
    expect(state.applied).toEqual(TEST_DEFAULTS);
    expect(state.isDirty).toBe(true);
  });

  it('resets dirty state when canceling changes', () => {
    const store = useSettingsStore.getState();
    store.setDraft({ language: 'rus' });
    store.cancelChanges();

    const state = useSettingsStore.getState();
    expect(state.draft).toEqual(state.applied);
    expect(state.isDirty).toBe(false);
  });
});
