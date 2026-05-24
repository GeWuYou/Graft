export function localizedApiErrorMessage(
  translate: (key: string) => string,
  messageKey?: string,
  fallback?: string | null,
) {
  if (messageKey) {
    const translated = translate(messageKey);
    if (translated !== messageKey) {
      return translated;
    }
  }

  return fallback?.trim() || '';
}
