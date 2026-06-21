/**
 * Formats a countdown duration from seconds into a human-readable string.
 *
 * @param seconds - The duration in seconds
 * @returns The formatted countdown string (e.g., `"30s"`, `"5m 30s"`, `"2h 15m"`), or `'--'` if the input is invalid
 */

export function formatRefreshCountdown(seconds: number | null | undefined): string {
  if (typeof seconds !== 'number' || !Number.isFinite(seconds) || seconds < 0) {
    return '--';
  }

  const normalizedSeconds = Math.floor(seconds);
  if (normalizedSeconds < 60) {
    return `${normalizedSeconds}s`;
  }

  if (normalizedSeconds < 3600) {
    const minutes = Math.floor(normalizedSeconds / 60);
    const remainingSeconds = normalizedSeconds % 60;
    return `${minutes}m ${padTimeUnit(remainingSeconds)}s`;
  }

  const hours = Math.floor(normalizedSeconds / 3600);
  const minutes = Math.floor((normalizedSeconds % 3600) / 60);
  return `${hours}h ${padTimeUnit(minutes)}m`;
}

/**
 * Formats a number as a two-digit string with leading zeros.
 *
 * @returns A string representation of the value, left-padded with zeros to at least 2 characters.
 */
function padTimeUnit(value: number) {
  return String(value).padStart(2, '0');
}
