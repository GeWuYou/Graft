import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { describe, expect, it } from 'vitest';

import {
  AUTH_SCHEME as GENERATED_AUTH_SCHEME,
  ERROR_CODE,
  HTTP_HEADER as GENERATED_HTTP_HEADER,
  MESSAGE_KEY as GENERATED_MESSAGE_KEY,
} from '../generated/platform';
import { API_CODE, type ApiCode, type ApiResponseCode } from './codes';
import { AUTH_SCHEME, HTTP_HEADER } from './headers';
import { MESSAGE_KEY, type MessageKey } from './messages';

describe('platform API contract compatibility exports', () => {
  it('exports the generated platform values without a second value map', () => {
    expect(API_CODE).toBe(ERROR_CODE);
    expect(MESSAGE_KEY).toBe(GENERATED_MESSAGE_KEY);
    expect(AUTH_SCHEME).toBe(GENERATED_AUTH_SCHEME);
    expect(HTTP_HEADER).toBe(GENERATED_HTTP_HEADER);

    const apiCode: ApiCode = API_CODE.AUTH_TOKEN_EXPIRED;
    const messageKey: MessageKey = MESSAGE_KEY.COMMON_COPYRIGHT;
    const responseCode: ApiResponseCode = 'SERVER_ADDED_CODE';

    expect(apiCode).toBe(ERROR_CODE.AUTH_TOKEN_EXPIRED);
    expect(messageKey).toBe(GENERATED_MESSAGE_KEY.COMMON_COPYRIGHT);
    expect(responseCode).toBe('SERVER_ADDED_CODE');
  });

  it('does not reintroduce projected server values as compatibility literals', () => {
    const projectedValues = [
      ...Object.values(ERROR_CODE),
      ...Object.values(GENERATED_MESSAGE_KEY),
      ...Object.values(GENERATED_AUTH_SCHEME),
      ...Object.values(GENERATED_HTTP_HEADER),
    ];
    const testDirectory = dirname(fileURLToPath(import.meta.url));
    const compatibilitySources = ['codes.ts', 'messages.ts', 'headers.ts'].map((fileName) =>
      readFileSync(resolve(testDirectory, fileName), 'utf8'),
    );

    for (const value of projectedValues) {
      for (const source of compatibilitySources) {
        expect(source).not.toContain(`'${value}'`);
        expect(source).not.toContain(`"${value}"`);
      }
    }
  });
});
