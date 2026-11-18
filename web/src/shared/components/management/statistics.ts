export type ManagementStatisticTone = 'default' | 'success' | 'warning' | 'danger' | 'muted';

export type ManagementStatisticItem = {
  label: string;
  marker?: string;
  value: number | string;
  // summary 布局用它给数值上语义色；inline/chips 布局忽略该字段，保持行内聚合的既有呈现。
  tone?: ManagementStatisticTone;
};
