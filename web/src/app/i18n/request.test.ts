import { describe, expect, it } from 'vitest';

import { createLocaleRequestHeaders, LOCALE_REQUEST_HEADER } from './request';

describe('createLocaleRequestHeaders', () => {
  it('adds the locale header without dropping existing headers', () => {
    expect(
      createLocaleRequestHeaders('zh-CN', {
        Authorization: 'Bearer token',
      }),
    ).toEqual({
      [LOCALE_REQUEST_HEADER]: 'zh-CN',
      Authorization: 'Bearer token',
    });
  });
});
