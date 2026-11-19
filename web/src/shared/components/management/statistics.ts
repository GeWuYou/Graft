export type ManagementStatisticTone = 'default' | 'success' | 'warning' | 'danger' | 'muted';

export type ManagementStatisticItem = {
  label: string;
  marker?: string;
  value: number | string;
  tone?: ManagementStatisticTone;
};
