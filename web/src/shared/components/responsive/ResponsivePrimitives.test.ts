import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import { resolveResponsiveDialogPolicy } from './dialog-policy';
import ResourceDetailLayout from './ResourceDetailLayout.vue';
import ResponsiveCardList from './ResponsiveCardList.vue';
import ResponsiveContent from './ResponsiveContent.vue';
import responsiveContentSource from './ResponsiveContent.vue?raw';
import ResponsiveDialog from './ResponsiveDialog.vue';
import ResponsiveEmpty from './ResponsiveEmpty.vue';
import ResponsiveForm from './ResponsiveForm.vue';
import ResponsiveHeader from './ResponsiveHeader.vue';
import ResponsivePage from './ResponsivePage.vue';
import ResponsiveTable from './ResponsiveTable.vue';
import ResponsiveToolbar from './ResponsiveToolbar.vue';

class ResizeObserverMock {
  static instances: ResizeObserverMock[] = [];

  callback: ResizeObserverCallback;
  disconnect = vi.fn();
  observe = vi.fn();

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
    ResizeObserverMock.instances.push(this);
  }

  emit(width: number) {
    this.callback([{ contentRect: { height: 0, width } } as ResizeObserverEntry], this as unknown as ResizeObserver);
  }
}

describe('responsive primitives', () => {
  afterEach(() => {
    ResizeObserverMock.instances = [];
    vi.unstubAllGlobals();
  });

  it('keeps page, content and header composition semantic while preserving slots', () => {
    const page = mount(ResponsivePage, { props: { layout: 'grid' }, slots: { default: '<p>content</p>' } });
    const content = mount(ResponsiveContent, { props: { layout: 'wide-split' }, slots: { default: '<p>detail</p>' } });
    const header = mount(ResponsiveHeader, {
      slots: {
        actions: '<button>create</button>',
        description: '<p>summary</p>',
        title: '<h1>Projects</h1>',
      },
    });

    expect(page.classes()).toContain('responsive-page--grid');
    expect(page.text()).toContain('content');
    expect(content.classes()).toContain('responsive-content--wide-split');
    expect(content.find('.responsive-content__inner').text()).toContain('detail');
    expect(header.find('.responsive-header__actions').text()).toContain('create');
    expect(header.find('.responsive-header__description').text()).toContain('summary');
  });

  it('uses the measured content container for wide split activation', () => {
    expect(responsiveContentSource).toContain('@container (width >= 75rem)');
    expect(responsiveContentSource).not.toContain('@media (width >= 75rem)');
  });

  it('keeps data tables scrollable and reserves compact cards for entity presentation', () => {
    const dataTable = mount(ResponsiveTable, {
      slots: { default: '<table><tbody><tr><td>row</td></tr></tbody></table>' },
    });
    const entityTable = mount(ResponsiveTable, {
      props: { presentation: 'entity' },
      slots: { cards: '<article>card</article>', default: '<table><tbody><tr><td>row</td></tr></tbody></table>' },
    });
    const form = mount(ResponsiveForm, {
      slots: { default: '<input aria-label="name">', actions: '<button>save</button>' },
    });
    const cards = mount(ResponsiveCardList, { slots: { default: '<article>entity</article>' } });

    expect(dataTable.attributes('data-responsive-presentation')).toBe('data');
    expect(dataTable.find('.responsive-table__scroll table').exists()).toBe(true);
    expect(entityTable.attributes('data-responsive-presentation')).toBe('entity');
    expect(form.find('.responsive-form__actions').text()).toContain('save');
    expect(cards.text()).toContain('entity');
  });

  it('allows entity cards to remain available through the comfortable density', async () => {
    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
    const wrapper = mount(ResponsiveTable, {
      props: { entityCardLayout: 'adaptive', presentation: 'entity' },
      slots: { cards: '<article>card</article>', default: '<table><tbody><tr><td>row</td></tr></tbody></table>' },
    });
    await nextTick();
    ResizeObserverMock.instances[0]?.emit(800);
    await nextTick();

    expect(wrapper.attributes('data-responsive-entity-card-layout')).toBe('adaptive');
    expect(wrapper.attributes('data-responsive-density')).toBe('comfortable');
    expect(wrapper.classes()).toContain('responsive-table--adaptive');
    expect(wrapper.find('.responsive-table__cards').text()).toContain('card');
    expect(wrapper.find('.responsive-table__scroll').exists()).toBe(false);
  });

  it('provides named toolbar and empty-state slots without business props', () => {
    const toolbar = mount(ResponsiveToolbar, {
      slots: {
        filters: '<input aria-label="filter">',
        overflow: '<button>more</button>',
        primary: '<button>create</button>',
      },
    });
    const empty = mount(ResponsiveEmpty, {
      props: { tone: 'error' },
      slots: { actions: '<button>retry</button>', description: '<p>Try again</p>', title: '<h2>Unavailable</h2>' },
    });

    expect(toolbar.find('.responsive-toolbar__filters input').exists()).toBe(true);
    expect(toolbar.find('.responsive-toolbar__primary').text()).toContain('create');
    expect(toolbar.find('.responsive-toolbar__overflow').text()).toContain('more');
    expect(empty.classes()).toContain('responsive-empty--error');
    expect(empty.find('.responsive-empty__actions').text()).toContain('retry');
  });

  it('resolves dialog surfaces from semantic purpose and size without pixel props', () => {
    expect(resolveResponsiveDialogPolicy(375, 'confirm', 'compact')).toMatchObject({
      interaction: 'interactive',
      surface: 'sheet',
    });
    expect(resolveResponsiveDialogPolicy(375, 'form', 'large')).toMatchObject({ surface: 'fullscreen' });
    expect(resolveResponsiveDialogPolicy(768, 'workspace', 'large')).toMatchObject({
      interaction: 'readonly',
      surface: 'drawer',
    });
    expect(resolveResponsiveDialogPolicy(992, 'detail', 'medium')).toMatchObject({ surface: 'drawer' });

    const wrapper = mount(ResponsiveDialog, {
      props: { purpose: 'form', size: 'large', title: 'Edit', visible: true },
      slots: { default: '<p>form fields</p>', footer: '<button>save</button>' },
      global: {
        stubs: { 't-dialog': { template: '<div><slot /></div>' }, 't-drawer': { template: '<div><slot /></div>' } },
      },
    });

    expect(wrapper.find('.responsive-dialog').classes()).toContain('responsive-dialog--drawer');
    expect(wrapper.text()).toContain('form fields');
    expect(wrapper.find('.responsive-dialog__footer').text()).toContain('save');
    expect(Object.keys(wrapper.props())).toEqual(expect.arrayContaining(['purpose', 'size']));
    expect(Object.keys(wrapper.props())).not.toContain('width');
  });

  it('uses a fullscreen detail surface below the compact breakpoint without an empty actions region', async () => {
    vi.stubGlobal('innerWidth', 390);
    const wrapper = mount(ResourceDetailLayout, {
      props: { backLabel: 'Back', title: 'app-shared-postgres', visible: true },
      slots: { default: '<p>IPAM configuration</p>' },
      global: { stubs: { 't-dialog': { template: '<div class="dialog"><slot /></div>' } } },
    });
    await nextTick();

    expect(wrapper.find('.dialog').exists()).toBe(true);
    expect(wrapper.text()).toContain('app-shared-postgres');
    expect(wrapper.find('.resource-detail-content__actions').exists()).toBe(false);
    expect(Object.keys(wrapper.props())).not.toContain('isMobile');
  });

  it('lets large detail drawers use the available width through comfortable and spacious densities', async () => {
    const DrawerStub = defineComponent({
      name: 'TDrawerStub',
      props: { size: { type: String, required: true } },
      setup(props, { slots }) {
        return () =>
          h('aside', { 'data-testid': 'responsive-detail-drawer', 'data-size': props.size }, slots.default?.());
      },
    });

    vi.stubGlobal('innerWidth', 800);
    const comfortableWrapper = mount(ResourceDetailLayout, {
      props: { backLabel: 'Back', size: 'large', title: 'Task detail', visible: true },
      slots: { default: '<p>Execution logs</p>' },
      global: { stubs: { 't-drawer': DrawerStub } },
    });
    await nextTick();

    expect(comfortableWrapper.get('[data-testid="responsive-detail-drawer"]').attributes('data-size')).toBe(
      'var(--graft-resource-detail-large-comfortable-width)',
    );
    comfortableWrapper.unmount();

    vi.stubGlobal('innerWidth', 1440);
    const wrapper = mount(ResourceDetailLayout, {
      props: { backLabel: 'Back', size: 'large', title: 'Task detail', visible: true },
      slots: { default: '<p>Execution logs</p>' },
      global: { stubs: { 't-drawer': DrawerStub } },
    });
    await nextTick();

    expect(wrapper.get('[data-testid="responsive-detail-drawer"]').attributes('data-size')).toBe(
      'var(--graft-resource-detail-large-fluid-width)',
    );
  });
});
