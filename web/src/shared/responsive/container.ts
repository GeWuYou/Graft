export interface ResponsiveContainerSize {
  height: number;
  width: number;
}

export const EMPTY_RESPONSIVE_CONTAINER_SIZE: ResponsiveContainerSize = {
  height: 0,
  width: 0,
};

export function normalizeResponsiveContainerSize(width: number, height: number): ResponsiveContainerSize {
  return {
    height: Number.isFinite(height) ? Math.max(0, height) : 0,
    width: Number.isFinite(width) ? Math.max(0, width) : 0,
  };
}
