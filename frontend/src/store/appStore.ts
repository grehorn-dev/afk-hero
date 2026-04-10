import { create } from 'zustand';
import type { ActivationRuntimeState, RuntimeState } from '../types/state';

interface AppStore {
  runtimeState: RuntimeState;
  activationState: ActivationRuntimeState;
  setRuntimeState: (state: Partial<RuntimeState>) => void;
  setActivationState: (state: Partial<ActivationRuntimeState>) => void;
}

export const useAppStore = create<AppStore>((set) => ({
  runtimeState: {
    state: 'Disabled',
    enabled: true,
    progress: 0,
    statusKey: 'status.Disabled',
  },
  activationState: {
    targetFound: false,
    tracking: false,
    progress: 0,
    processName: '',
    windowTitle: '',
  },
  setRuntimeState: (update) =>
    set((s) => ({
      runtimeState: { ...s.runtimeState, ...update },
    })),
  setActivationState: (update) =>
    set((s) => ({
      activationState: { ...s.activationState, ...update },
    })),
}));
