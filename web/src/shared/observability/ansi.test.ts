import { describe, expect, it } from 'vitest';

import { stripAnsiControlSequences } from './ansi';

describe('stripAnsiControlSequences', () => {
  it('removes CSI color sequences and OSC metadata sequences', () => {
    expect(stripAnsiControlSequences('\u001B]0;monitor\u0007\u001B[31mERROR\u001B[0m request failed')).toBe(
      'ERROR request failed',
    );
  });

  it('removes CSI sequences using the C1 control character form', () => {
    expect(stripAnsiControlSequences('\u009B33mWARN\u009B0m')).toBe('WARN');
  });
});
