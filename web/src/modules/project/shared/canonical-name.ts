const canonicalProjectNamePattern = /^[a-z0-9][a-z0-9_-]*$/;

export function isValidProjectCanonicalName(value: string) {
  return canonicalProjectNamePattern.test(value.trim());
}
