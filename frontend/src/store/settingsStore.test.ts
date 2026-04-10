import { beforeEach, describe, expect, it } from 'vitest';
import { DEFAULT_SETTINGS } from '../types/settings';
import { useSettingsStore } from './settingsStore';

describe('settingsStore', () => {
  beforeEach(() => {
    useSettingsStore.setState({
      draft: { ...DEFAULT_SETTINGS },
      applied: { ...DEFAULT_SETTINGS },
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
    expect(state.draft).toEqual({ ...DEFAULT_SETTINGS, language: 'rus', theme: 'Light' });
    expect(state.applied).toEqual(DEFAULT_SETTINGS);
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
