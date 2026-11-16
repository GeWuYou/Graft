// @vitest-environment node

import { describe, expect, it } from 'vitest';

import config from './vitest.config';

describe('Vitest dependency optimizer', () => {
  it('prebundles the declared color package and its transitive imports', () => {
    expect(config.test?.deps?.optimizer?.client?.include).toEqual(['tvision-color']);
  });
});
