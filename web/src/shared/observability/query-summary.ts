type SummaryPart = string | null | undefined | false;

export function joinQuerySummary(parts: SummaryPart[]) {
  return parts.filter((part): part is string => typeof part === 'string' && part.trim().length > 0).join(' · ');
}
