/**
 * 容器尺寸阈值必须与 style/variables.less 中既有断点保持一致。运行时仅暴露布局能力，
 * 不将阈值解释为设备类型。
 */
export const RESPONSIVE_CONTAINER_THRESHOLDS = {
  comfortable: 768,
  spacious: 992,
  wide: 1200,
} as const;

export type ResponsiveContainerThreshold = keyof typeof RESPONSIVE_CONTAINER_THRESHOLDS;
