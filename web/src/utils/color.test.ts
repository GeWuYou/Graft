import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { syncFaviconColor } from './color';

describe('syncFaviconColor', () => {
  beforeEach(() => {
    document.head.innerHTML = '<link id="graft-favicon" rel="icon" type="image/svg+xml" href="/favicon.svg" />';
  });

  afterEach(() => {
    document.head.innerHTML = '';
  });

  it('replaces the existing favicon with a brand-colored SVG', () => {
    syncFaviconColor('#2BA471');

    const favicon = document.getElementById('graft-favicon') as HTMLLinkElement;
    const href = favicon.getAttribute('href') ?? '';

    expect(href).toMatch(/^data:image\/svg\+xml,/u);
    expect(decodeURIComponent(href.slice('data:image/svg+xml,'.length))).toContain('fill="#2BA471"');
  });

  it('does nothing when the favicon link is unavailable', () => {
    document.head.innerHTML = '';

    expect(() => syncFaviconColor('#2BA471')).not.toThrow();
  });
});
