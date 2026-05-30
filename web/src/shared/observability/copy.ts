import { createLogger } from '@/utils/logger';

const logger = createLogger('shared.observability.copy');

function legacyCopyText(value: string) {
  const textarea = document.createElement('textarea');
  textarea.value = value;
  textarea.setAttribute('readonly', 'true');
  textarea.style.position = 'fixed';
  textarea.style.opacity = '0';
  textarea.style.pointerEvents = 'none';
  textarea.style.top = '0';
  textarea.style.left = '0';
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();

  try {
    return document.execCommand('copy');
  } finally {
    document.body.removeChild(textarea);
  }
}

export async function copyText(value: string) {
  if (!value) {
    return false;
  }

  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return true;
    } catch (error) {
      logger.warn('clipboard api copy failed, fallback to execCommand', {
        error,
        secureContext: window.isSecureContext,
      });
    }
  }

  try {
    return legacyCopyText(value);
  } catch (error) {
    logger.error('legacy copy failed', { error });
    return false;
  }
}
