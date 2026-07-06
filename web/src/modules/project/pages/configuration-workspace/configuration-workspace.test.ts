import { describe, expect, it } from 'vitest';

import { buildConfigurationDraftRequest, normalizeTextBlock } from '../../shared/configuration-workspace';

describe('configuration workspace helpers', () => {
  it('normalizes text blocks with trimmed trailing whitespace and trailing newline', () => {
    expect(normalizeTextBlock('services:\r\n  api:  \r\n    image: app  \r\n')).toBe(
      'services:\n  api:\n    image: app\n',
    );
  });

  it('builds a normalized draft payload for diff, validate, and deploy endpoints', () => {
    expect(
      buildConfigurationDraftRequest({
        compose_file_content: 'services:\n  api:\n    image: app  ',
        env_file_content: 'APP_ENV=prod  ',
      }),
    ).toEqual({
      compose_file_content: 'services:\n  api:\n    image: app\n',
      env_file_content: 'APP_ENV=prod\n',
    });
  });
});
