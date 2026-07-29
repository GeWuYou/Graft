import { resolveResponsiveDensity, type ResponsiveDensity } from '@/shared/responsive';

export type ResponsiveDialogPurpose = 'confirm' | 'detail' | 'form' | 'workspace';
export type ResponsiveDialogSize = 'compact' | 'medium' | 'large';
export type ResponsiveDialogSurface = 'dialog' | 'drawer' | 'sheet' | 'fullscreen';

export interface ResponsiveDialogPolicy {
  density: ResponsiveDensity;
  interaction: 'interactive' | 'readonly';
  surface: ResponsiveDialogSurface;
}

/**
 * 对话框表面只由可用容器和业务意图决定，调用方不传递设备类型或像素宽度。
 */
export function resolveResponsiveDialogPolicy(
  width: number,
  purpose: ResponsiveDialogPurpose,
  size: ResponsiveDialogSize,
): ResponsiveDialogPolicy {
  const density = resolveResponsiveDensity(width);

  if (density === 'compact') {
    if (purpose === 'confirm') {
      return { density, interaction: 'interactive', surface: 'sheet' };
    }

    if (purpose === 'workspace') {
      return { density, interaction: 'readonly', surface: 'fullscreen' };
    }

    if (purpose === 'detail') {
      return { density, interaction: 'interactive', surface: 'drawer' };
    }

    if (purpose === 'form' && size !== 'compact') {
      return { density, interaction: 'interactive', surface: 'fullscreen' };
    }

    return { density, interaction: 'interactive', surface: 'sheet' };
  }

  return { density, interaction: purpose === 'workspace' ? 'readonly' : 'interactive', surface: 'drawer' };
}
