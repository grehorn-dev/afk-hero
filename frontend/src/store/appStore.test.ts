import { beforeEach, describe, expect, it } from 'vitest';
import { useAppStore } from './appStore';

describe('appStore', () => {
  beforeEach(() => {
    useAppStore.setState({
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
    });
  });

  it('merges partial runtime state updates', () => {
    useAppStore.getState().setRuntimeState({
      state: 'Animating',
      progress: 0.5,
    });

    const state = useAppStore.getState().runtimeState;
    expect(state.state).toBe('Animating');
    expect(state.progress).toBe(0.5);
    expect(state.statusKey).toBe('status.Disabled');
  });

  it('merges partial activation state updates', () => {
    useAppStore.getState().setActivationState({
      targetFound: true,
      tracking: true,
      progress: 0.4,
      processName: 'game.exe',
      windowTitle: 'Game',
    });

    const state = useAppStore.getState().activationState;
    expect(state.targetFound).toBe(true);
    expect(state.tracking).toBe(true);
    expect(state.progress).toBe(0.4);
    expect(state.processName).toBe('game.exe');
    expect(state.windowTitle).toBe('Game');
  });
});
