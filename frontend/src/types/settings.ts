export type Shape = 'Random' | 'Pentagon' | 'Rectangle' | 'Triangle' | 'Segment' | 'OnePixel';
export type Direction = 'Random' | 'Center' | 'Up' | 'Down' | 'Left' | 'Right';
export type RangeMode = 'Fixed' | 'Random';
export type ActivationMode = 'Auto' | 'Manual';
export type Theme = 'Dark' | 'Light';

export interface NumericRange {
  mode: RangeMode;
  value: number;
  minVal: number;
  maxVal: number;
}

export interface Settings {
  enabled: boolean;
  advanced: boolean;
  logging: boolean;
  language: string;
  theme: Theme;
  shape: Shape;
  direction: Direction;
  distance: NumericRange;
  interval: NumericRange;
  speed: NumericRange;
  inactivity: NumericRange;
  activationEnabled: boolean;
  activationMode: ActivationMode;
  activationTimeout: number;
  activationTargetWindowOnly: boolean;
  targetWindowTitle: string;
  targetWindowClass: string;
  targetProcessName: string;
}

export interface WindowInfo {
  id: number;
  title: string;
  className: string;
  processName: string;
  autoCandidate: boolean;
}

export interface Language {
  code: string;
  name: string;
}

export const SHAPES: Shape[] = ['Random', 'Pentagon', 'Rectangle', 'Triangle', 'Segment', 'OnePixel'];
export const DIRECTIONS: Direction[] = ['Random', 'Center', 'Up', 'Down', 'Left', 'Right'];
export const RANGE_MODES: RangeMode[] = ['Fixed', 'Random'];
export const THEMES: Theme[] = ['Dark', 'Light'];
