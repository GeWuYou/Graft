import { describe, expect, it } from 'vitest';

import { localizeContainerResourceError } from './localize-resource-error';

const messages: Record<string, string> = {
  'container.list.stats.unavailableReasonFallback': 'Resource stats are currently unavailable',
  'ops.container.error.runtimeUnavailable': 'Container runtime connection is unavailable',
};

const translate = (key: string) => messages[key] ?? key;

describe('localizeContainerResourceError', () => {
  it('prefers a translated stable error key over backend fallback text', () => {
    expect(
      localizeContainerResourceError(
        translate,
        {
          stats_error_key: 'ops.container.error.runtimeUnavailable',
          stats_error_message: 'Container runtime connection is unavailable',
        },
        'container.list.stats.unavailableReasonFallback',
      ),
    ).toBe('Container runtime connection is unavailable');
  });

  it('maps technical stats reasons to the active locale fallback', () => {
    expect(
      localizeContainerResourceError(
        translate,
        { stats_error_key: 'stats_timeout', stats_error_message: 'Container stats collection timed out.' },
        'container.list.stats.unavailableReasonFallback',
      ),
    ).toBe('Resource stats are currently unavailable');
  });

  it('uses backend text only when no stable reason key is available', () => {
    expect(
      localizeContainerResourceError(
        translate,
        { stats_error_message: 'Legacy runtime message' },
        'container.list.stats.unavailableReasonFallback',
      ),
    ).toBe('Legacy runtime message');
  });
});
