/** 运行时 primitive 与响应式样式层共用的 CSS token 名称。 */
export const RESPONSIVE_STYLE_TOKENS = {
  containerComfortableMin: '--graft-responsive-container-comfortable-min',
  containerSpaciousMin: '--graft-responsive-container-spacious-min',
  containerWideMin: '--graft-responsive-container-wide-min',
  contentGutter: '--graft-responsive-content-gutter',
  dialogCompactMax: '--graft-responsive-dialog-compact-max',
  dialogMediumMax: '--graft-responsive-dialog-medium-max',
  dialogLargeMax: '--graft-responsive-dialog-large-max',
  touchTargetMin: '--graft-responsive-touch-target-min',
} as const;

export type ResponsiveStyleToken = (typeof RESPONSIVE_STYLE_TOKENS)[keyof typeof RESPONSIVE_STYLE_TOKENS];
