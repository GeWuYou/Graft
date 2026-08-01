type Translate = (key: string) => string;

type ResourceErrorFields = {
  stats_error_key?: string | null;
  stats_error_message?: string | null;
  unavailable_reason?: string | null;
};

const TECHNICAL_STATS_REASONS = new Set([
  'stats_incomplete',
  'stats_not_collected',
  'stats_timeout',
  'stats_unavailable',
]);

/** 优先使用稳定错误键本地化资源统计原因，后端文本仅作为无键场景的兼容兜底。 */
export function localizeContainerResourceError(
  translate: Translate,
  resource: ResourceErrorFields | null | undefined,
  fallbackKey: string,
) {
  const candidates = [resource?.stats_error_key, resource?.unavailable_reason]
    .map((value) => value?.trim())
    .filter((value): value is string => Boolean(value));

  for (const candidate of candidates) {
    const translated = translate(candidate);
    if (translated !== candidate) return translated;
    if (TECHNICAL_STATS_REASONS.has(candidate) || candidate.startsWith('container.stats:')) {
      return translate(fallbackKey);
    }
  }

  const message = resource?.stats_error_message?.trim();
  return message || translate(fallbackKey);
}
