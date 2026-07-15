// @vitest-environment node

import { describe, expect, it } from 'vitest';

import config from './vitest.config';

describe('Vitest dependency optimizer', () => {
  it('prebundles the material color dependency chain that uses extensionless ESM imports', () => {
    expect(config.test?.deps?.optimizer?.client?.include).toEqual([
      '@material/material-color-utilities',
      'tvision-color',
    ]);
  });
});
