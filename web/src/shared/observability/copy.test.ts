import { afterEach, describe, expect, it, vi } from 'vitest';

import { copyText } from './copy';

const originalClipboard = navigator.clipboard;
const originalExecCommand = document.execCommand;

afterEach(() => {
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: originalClipboard,
  });
  document.execCommand = originalExecCommand;
  vi.restoreAllMocks();
});

describe('copyText', () => {
  it('uses clipboard api when available', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    });

    await expect(copyText('req-1')).resolves.toBe(true);
    expect(writeText).toHaveBeenCalledWith('req-1');
  });

  it('falls back to execCommand when clipboard api rejects', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('denied'));
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    });
    document.execCommand = vi.fn().mockReturnValue(true);

    await expect(copyText('trace-1')).resolves.toBe(true);
    expect(document.execCommand).toHaveBeenCalledWith('copy');
  });
});
