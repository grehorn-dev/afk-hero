export type AppState =
  | 'Disabled'
  | 'WaitingForInactivity'
  | 'WaitingForInterval'
  | 'Animating'
  | 'Error';

export interface RuntimeState {
  state: AppState;
  enabled: boolean;
  progress: number;
  statusKey: string;
}

export interface ActivationRuntimeState {
  targetFound: boolean;
  tracking: boolean;
  progress: number;
  processName: string;
  windowTitle: string;
}
