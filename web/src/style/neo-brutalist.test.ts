import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, it } from 'vitest';

const styleSource = readFileSync(join(process.cwd(), 'src/style/neo-brutalist.less'), 'utf8');
const projectDetailSource = readFileSync(join(process.cwd(), 'src/modules/project/pages/detail/index.vue'), 'utf8');

describe('neo-brutalist hard surfaces', () => {
  it('frames ordinary tabs without changing shell route tabs', () => {
    expect(styleSource).toContain('.t-tabs:not(.tdesign-starter-layout-tabs-nav) {');
    expect(styleSource).toContain('border: 2px solid var(--graft-neo-ink);');
    expect(styleSource).toContain('box-shadow: var(--graft-neo-shadow);');
    expect(styleSource).toContain('.t-tabs:not(.tdesign-starter-layout-tabs-nav) .t-tabs__header {');
  });

  it('gives feedback alerts a hard surface boundary while preserving semantic fills', () => {
    expect(styleSource).toContain('.t-alert {');
    expect(styleSource).toContain('border-radius: 0;');
    expect(styleSource).toContain('box-shadow: var(--td-shadow-1);');
  });

  it('uses squared, high-contrast status tags without changing their semantic fill', () => {
    expect(styleSource).toContain('.t-tag {');
    expect(styleSource).toContain('border-color: var(--graft-neo-ink);');
    expect(styleSource).toContain('border-radius: 0;');
  });

  it('uses a toned-down yellow surface with a consistent foreground in dark mode', () => {
    expect(styleSource).toContain(":root[theme-mode='dark'][data-graft-hard-surface] {");
    expect(styleSource).toContain(
      '--graft-neo-active-surface: color-mix(in srgb, var(--graft-neo-accent) 30%, #171717);',
    );
    expect(styleSource).toContain('.t-button--theme-primary.t-button--variant-base,');
    expect(styleSource).toContain('background: var(--graft-neo-active-surface);');
    expect(styleSource).toContain('color: var(--graft-neo-ink);');
  });

  it('keeps active dark side navigation text legible on the accent background', () => {
    expect(styleSource).toContain('.t-default-menu.t-menu--dark .t-menu__item.t-is-active');
    expect(styleSource).toContain('.t-default-menu.t-menu--dark .t-menu__item.t-is-active .t-menu__content');
    expect(styleSource).toContain('.t-default-menu.t-menu--dark .t-submenu.t-is-active > .t-menu__item,');
    expect(styleSource).toContain('color: var(--graft-neo-ink);');
  });

  it('keeps project detail content clear of the hard tab frame', () => {
    expect(projectDetailSource).toContain('.project-detail-tabs :deep(.t-tabs__content) {');
    expect(projectDetailSource).toContain('padding: var(--graft-density-gap-16);');
  });
});
