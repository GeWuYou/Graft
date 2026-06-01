import { formatLocaleDateTime } from './time';

export type TrendAxisPreset = 'last_24h' | 'last_7d' | 'last_30d';

export type TrendAxisPoint = {
  key: string;
  start?: string;
  end?: string;
};

function parseDate(value?: string | null) {
  if (!value) {
    return null;
  }

  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

function isCrossYear(points: TrendAxisPoint[]) {
  const years = new Set(
    points
      .map((point) => parseDate(point.start)?.getFullYear())
      .filter((value): value is number => typeof value === 'number'),
  );
  return years.size > 1;
}

function resolveAxisFormat(preset: TrendAxisPreset, crossYear: boolean): Intl.DateTimeFormatOptions {
  if (crossYear) {
    return { year: 'numeric', month: '2-digit', day: '2-digit' };
  }

  if (preset === 'last_24h') {
    return { hour: '2-digit', minute: '2-digit', hour12: false };
  }

  return { month: '2-digit', day: '2-digit' };
}

function clampTickCount(pointCount: number) {
  if (pointCount <= 0) {
    return 0;
  }
  return Math.min(7, Math.max(5, Math.min(pointCount, 7)));
}

function buildTickIndexes(pointCount: number) {
  const tickCount = clampTickCount(pointCount);
  if (tickCount === 0) {
    return new Set<number>();
  }
  if (pointCount <= tickCount) {
    return new Set(Array.from({ length: pointCount }, (_, index) => index));
  }

  const step = (pointCount - 1) / (tickCount - 1);
  const indexes = new Set<number>();
  for (let index = 0; index < tickCount; index += 1) {
    indexes.add(Math.round(index * step));
  }
  indexes.add(0);
  indexes.add(pointCount - 1);
  return indexes;
}

export function buildTrendAxisLabels(points: TrendAxisPoint[], preset: TrendAxisPreset, locale?: string) {
  const crossYear = isCrossYear(points);
  const formatOptions = resolveAxisFormat(preset, crossYear);
  const tickIndexes = buildTickIndexes(points.length);

  return points.map((point, index) => ({
    key: point.key,
    axisLabel: tickIndexes.has(index) ? formatLocaleDateTime(point.start, locale, formatOptions) : '',
  }));
}

export function formatTrendTooltipDateTime(value?: string | null, locale?: string) {
  return formatLocaleDateTime(value, locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
}
