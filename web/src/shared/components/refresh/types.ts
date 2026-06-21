export type RefreshControlValue = number | string;
export type RefreshControlStatus = 'running' | 'paused' | 'off';

export type RefreshControlOption = {
  label: string;
  value: RefreshControlValue;
};
