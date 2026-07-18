import box from '@iconify-icons/lucide/box';
import imageIcon from '@iconify-icons/lucide/image';
import docker from '@iconify-icons/tabler/brand-docker';
import { describe, expect, it } from 'vitest';

import { resolveMenuIcon } from './menu-icon';

describe('resolveMenuIcon', () => {
  it('uses the Tabler Docker brand icon for Docker menus', () => {
    expect(resolveMenuIcon('docker')).toEqual(docker);
  });

  it('uses the Lucide box icon for container menus', () => {
    expect(resolveMenuIcon('container')).toEqual(box);
  });

  it('uses the Lucide image icon for image menus', () => {
    expect(resolveMenuIcon('image')).toEqual(imageIcon);
  });

  it('keeps application and runtime targets semantically distinct', () => {
    expect(resolveMenuIcon('application')).not.toEqual(resolveMenuIcon('runtime-target'));
  });

  it('keeps visible observability entries from falling back to one generic glyph', () => {
    const keys = ['dashboard', 'runtime-overview', 'dependencies', 'module-runtime', 'search', 'file-search'];

    expect(new Set(keys.map((key) => resolveMenuIcon(key))).size).toBe(keys.length);
  });

  it('uses a stable Lucide fallback for unknown server identifiers', () => {
    expect(resolveMenuIcon('unknown-menu-icon')).toEqual(resolveMenuIcon());
  });
});
