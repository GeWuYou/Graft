import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import WorkspaceCreatePage from './create.vue';

const projectContractMocks = vi.hoisted(() => ({
  getApplicationCatalog: vi.fn(),
}));

vi.mock('@/modules/project/contract/application-catalog', () => ({
  getApplicationCatalog: projectContractMocks.getApplicationCatalog,
}));

vi.mock('../../api/build', () => ({
  createBuildWorkspace: vi.fn(),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

const selectStub = defineComponent({
  name: 'TSelectStub',
  inheritAttrs: false,
  props: {
    filterable: Boolean,
    loading: Boolean,
    modelValue: {
      type: String,
      default: '',
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['change', 'search', 'update:modelValue'],
  setup(props, { attrs }) {
    return () => h('div', { ...attrs, 'data-testid': 'application-select', 'data-value': props.modelValue });
  },
});

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  setup(_, { slots }) {
    return () => h('div', [slots.default?.(), slots.help?.(), slots.operation?.()]);
  },
});

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((resolver) => {
    resolve = resolver;
  });
  return { promise, resolve };
}

function application(applicationId: string, displayName: string) {
  return { application_id: applicationId, display_name: displayName };
}

function mountPage() {
  return mount(WorkspaceCreatePage, {
    global: {
      directives: {
        permission: () => undefined,
      },
      stubs: {
        'management-page-content': passthroughStub,
        'management-page-header': passthroughStub,
        'save-icon': passthroughStub,
        't-alert': passthroughStub,
        't-button': passthroughStub,
        't-card': passthroughStub,
        't-form': passthroughStub,
        't-form-item': passthroughStub,
        't-input': passthroughStub,
        't-select': selectStub,
      },
    },
  });
}

describe('WorkspaceCreatePage', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    projectContractMocks.getApplicationCatalog.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('invalidates the active application request as soon as the search input changes', async () => {
    const staleSearch = deferred<{ items: Array<ReturnType<typeof application>> }>();
    const currentSearch = deferred<{ items: Array<ReturnType<typeof application>> }>();
    projectContractMocks.getApplicationCatalog
      .mockResolvedValueOnce({ items: [application('initial', 'Initial application')] })
      .mockReturnValueOnce(staleSearch.promise)
      .mockReturnValueOnce(currentSearch.promise);

    const wrapper = mountPage();
    await flushPromises();
    const select = wrapper.getComponent(selectStub);

    select.vm.$emit('search', 'stale keyword', { e: new KeyboardEvent('keydown') });
    await vi.advanceTimersByTimeAsync(250);
    expect(select.props('loading')).toBe(true);

    select.vm.$emit('search', 'current keyword', { e: new KeyboardEvent('keydown') });
    staleSearch.resolve({ items: [application('stale', 'Stale application')] });
    await flushPromises();

    expect(select.props('options')).toEqual([{ label: 'Initial application', value: 'initial' }]);
    expect(select.props('loading')).toBe(true);

    await vi.advanceTimersByTimeAsync(250);
    expect(projectContractMocks.getApplicationCatalog).toHaveBeenNthCalledWith(3, {
      keyword: 'current keyword',
      limit: 20,
      offset: 0,
    });

    currentSearch.resolve({ items: [application('current', 'Current application')] });
    await flushPromises();
    expect(select.props('options')).toEqual([{ label: 'Current application', value: 'current' }]);
    expect(select.props('loading')).toBe(false);
    wrapper.unmount();
  });

  it('searches the server-side application universe and ignores stale responses while retaining the selection', async () => {
    const staleSearch = deferred<{ items: Array<ReturnType<typeof application>> }>();
    const currentSearch = deferred<{ items: Array<ReturnType<typeof application>> }>();
    const replacementSearch = deferred<{ items: Array<ReturnType<typeof application>> }>();
    projectContractMocks.getApplicationCatalog
      .mockResolvedValueOnce({ items: [application('initial', 'Initial application')] })
      .mockReturnValueOnce(staleSearch.promise)
      .mockReturnValueOnce(currentSearch.promise)
      .mockReturnValueOnce(replacementSearch.promise);

    const wrapper = mountPage();
    await flushPromises();
    const select = wrapper.getComponent(selectStub);

    expect(projectContractMocks.getApplicationCatalog).toHaveBeenNthCalledWith(1, { limit: 20, offset: 0 });
    expect(select.props('filterable')).toBe(true);

    select.vm.$emit('search', 'stale keyword', { e: new KeyboardEvent('keydown') });
    await vi.advanceTimersByTimeAsync(250);
    select.vm.$emit('search', 'remote candidate', { e: new KeyboardEvent('keydown') });
    await vi.advanceTimersByTimeAsync(250);
    expect(projectContractMocks.getApplicationCatalog).toHaveBeenNthCalledWith(3, {
      keyword: 'remote candidate',
      limit: 20,
      offset: 0,
    });

    currentSearch.resolve({ items: [application('beyond-initial-page', 'Remote candidate')] });
    await flushPromises();
    expect(select.props('options')).toContainEqual({ label: 'Remote candidate', value: 'beyond-initial-page' });

    select.vm.$emit('update:modelValue', 'beyond-initial-page');
    select.vm.$emit('change', 'beyond-initial-page', { trigger: 'check' });
    await flushPromises();
    select.vm.$emit('search', 'replacement', { e: new KeyboardEvent('keydown') });
    await vi.advanceTimersByTimeAsync(250);
    expect(select.props('loading')).toBe(true);
    expect(select.props('options')).toContainEqual({ label: 'Remote candidate', value: 'beyond-initial-page' });

    replacementSearch.resolve({ items: [application('replacement', 'Replacement application')] });
    await flushPromises();
    expect(select.props('options')).toEqual([
      { label: 'Remote candidate', value: 'beyond-initial-page' },
      { label: 'Replacement application', value: 'replacement' },
    ]);

    staleSearch.resolve({ items: [application('stale', 'Stale application')] });
    await flushPromises();
    expect(select.props('options')).not.toContainEqual({ label: 'Stale application', value: 'stale' });
    expect(select.props('modelValue')).toBe('beyond-initial-page');
    wrapper.unmount();
  });
});
