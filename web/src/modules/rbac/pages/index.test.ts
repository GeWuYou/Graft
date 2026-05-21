import { readFile, unlink, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { compileScript, parse } from '@vue/compiler-sfc';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Component } from 'vue';

import { RBAC_PERMISSION_CODE } from '../contract/permissions';

type RbacPageTestState = {
  permissionState: {
    grantedCodes: string[];
  };
  rbacApiMocks: {
    assignRolePermissions: ReturnType<typeof vi.fn>;
    createRole: ReturnType<typeof vi.fn>;
    getPermissions: ReturnType<typeof vi.fn>;
    getRolePermissionBindings: ReturnType<typeof vi.fn>;
    getRoles: ReturnType<typeof vi.fn>;
    updateRole: ReturnType<typeof vi.fn>;
  };
};

function getTestState(): RbacPageTestState {
  const globalState = globalThis as typeof globalThis & {
    __rbacPageTestState?: RbacPageTestState;
  };

  if (!globalState.__rbacPageTestState) {
    globalState.__rbacPageTestState = {
      permissionState: {
        grantedCodes: [],
      },
      rbacApiMocks: {
        assignRolePermissions: vi.fn(),
        createRole: vi.fn(),
        getPermissions: vi.fn(),
        getRolePermissionBindings: vi.fn(),
        getRoles: vi.fn(),
        updateRole: vi.fn(),
      },
    };
  }

  return globalState.__rbacPageTestState;
}

async function loadVueComponent(relativePath: string): Promise<Component> {
  const filename = fileURLToPath(new URL(relativePath, import.meta.url));
  const source = await readFile(filename, 'utf8');
  const { descriptor } = parse(source, { filename });
  const script = compileScript(descriptor, {
    id: filename,
    inlineTemplate: true,
  });
  const compiledPath = join(dirname(filename), `.bun-test-${Date.now()}-${Math.random().toString(36).slice(2)}.ts`);

  await writeFile(compiledPath, script.content, 'utf8');

  try {
    const module = (await import(`${pathToFileURL(compiledPath).href}?t=${Date.now()}`)) as {
      default: Component;
    };

    return module.default;
  } finally {
    await unlink(compiledPath).catch(() => undefined);
  }
}

function ensureDomEnvironment() {
  if (typeof window !== 'undefined' && typeof document !== 'undefined') {
    return;
  }

  const require = createRequire(import.meta.url);
  const { JSDOM } = require('jsdom') as {
    JSDOM: new (
      html?: string,
      options?: { url?: string },
    ) => {
      window: Window & typeof globalThis;
    };
  };
  const jsdom = new JSDOM('<!doctype html><html><body></body></html>', {
    url: 'http://localhost/',
  });
  const { window: domWindow } = jsdom;

  Object.assign(globalThis, {
    document: domWindow.document,
    Element: domWindow.Element,
    Event: domWindow.Event,
    HTMLElement: domWindow.HTMLElement,
    MutationObserver: domWindow.MutationObserver,
    Node: domWindow.Node,
    SVGElement: domWindow.SVGElement,
    navigator: domWindow.navigator,
    window: domWindow,
  });
}

ensureDomEnvironment();

const { flushPromises, mount } = await import('@vue/test-utils');
const { MessagePlugin } = await import('tdesign-vue-next');
const { defineComponent, h } = await import('vue');
const createMessagePromise = () => Promise.resolve({} as Awaited<ReturnType<typeof MessagePlugin.error>>);

vi.mock('../api/rbac', () => ({
  assignRolePermissions: getTestState().rbacApiMocks.assignRolePermissions,
  createRole: getTestState().rbacApiMocks.createRole,
  getPermissions: getTestState().rbacApiMocks.getPermissions,
  getRolePermissionBindings: getTestState().rbacApiMocks.getRolePermissionBindings,
  getRoles: getTestState().rbacApiMocks.getRoles,
  updateRole: getTestState().rbacApiMocks.updateRole,
}));

vi.mock('@/store', () => {
  return {
    usePermissionStore: () => ({
      hasAnyPermission: (codes: string[]) =>
        codes.some((code) => getTestState().permissionState.grantedCodes.includes(code)),
      hasPermission: (code: string) => getTestState().permissionState.grantedCodes.includes(code),
    }),
  };
});

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    locale: {
      value: 'en-US',
    },
  }),
}));
const messageMocks = {
  error: vi.spyOn(MessagePlugin, 'error').mockImplementation(() => createMessagePromise()),
  success: vi.spyOn(MessagePlugin, 'success').mockImplementation(() => createMessagePromise()),
  warning: vi.spyOn(MessagePlugin, 'warning').mockImplementation(() => createMessagePromise()),
};
const { permissionState, rbacApiMocks } = getTestState();
const RolePage = await loadVueComponent('./index.vue');

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    description: {
      type: String,
      default: '',
    },
    title: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('div', [props.title, props.description, slots.default?.()]);
  },
});

const tableStub = defineComponent({
  name: 'TTableStub',
  props: {
    columns: {
      type: Array,
      default: () => [],
    },
    data: {
      type: Array,
      default: () => [],
    },
  },
  setup(props, { slots }) {
    return () => {
      if (props.data.length === 0) {
        return h('div', slots.empty?.());
      }

      const hasOperationColumn = props.columns.some(
        (column) =>
          typeof column === 'object' && column !== null && 'colKey' in column && column.colKey === 'operation',
      );

      return h(
        'div',
        props.data.map((row) =>
          h('div', { 'data-role-id': String((row as { id: number }).id) }, [
            h('span', String((row as { name?: string }).name ?? '')),
            h('span', String((row as { display?: string }).display ?? '')),
            h('span', String((row as { description?: string | null }).description ?? '')),
            hasOperationColumn ? slots.operation?.({ row }) : null,
          ]),
        ),
      );
    };
  },
});

const dialogStub = defineComponent({
  name: 'TDialogStub',
  props: {
    header: {
      type: String,
      default: '',
    },
    visible: {
      type: Boolean,
      default: false,
    },
  },
  setup(props, { slots }) {
    return () =>
      props.visible
        ? h('section', { 'data-testid': 'dialog', 'data-header': props.header }, [slots.body?.(), slots.default?.()])
        : null;
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  props: {
    disabled: {
      type: Boolean,
      default: false,
    },
    loading: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          disabled: props.disabled,
          'data-loading': String(props.loading),
          onClick: (event: MouseEvent) => {
            if (!props.disabled) {
              emit('click', event);
            }
          },
        },
        slots.default?.(),
      );
  },
});

const formStub = defineComponent({
  name: 'TFormStub',
  emits: ['submit'],
  setup(_, { emit, slots }) {
    return () =>
      h(
        'form',
        {
          'data-testid': 'role-form',
          onSubmit: (event: Event) => {
            event.preventDefault();
            emit('submit', { validateResult: true });
          },
        },
        slots.default?.(),
      );
  },
});

const inputStub = defineComponent({
  name: 'TInputStub',
  props: {
    modelValue: {
      type: String,
      default: '',
    },
    placeholder: {
      type: String,
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        placeholder: props.placeholder,
        value: props.modelValue,
        onInput: (event: Event) => {
          emit('update:modelValue', (event.target as HTMLInputElement).value);
        },
      });
  },
});

const textareaStub = defineComponent({
  name: 'TTextareaStub',
  props: {
    modelValue: {
      type: String,
      default: '',
    },
    placeholder: {
      type: String,
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h(
        'textarea',
        {
          placeholder: props.placeholder,
          value: props.modelValue,
          onInput: (event: Event) => {
            emit('update:modelValue', (event.target as HTMLTextAreaElement).value);
          },
        },
        props.modelValue,
      );
  },
});

const checkboxGroupStub = defineComponent({
  name: 'TCheckboxGroupStub',
  props: {
    disabled: {
      type: Boolean,
      default: false,
    },
    modelValue: {
      type: Array<number>,
      default: () => [],
    },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-disabled': String(props.disabled),
          'data-testid': 'permission-checkbox-group',
          'data-selected-permission-ids': JSON.stringify(props.modelValue),
        },
        slots.default?.(),
      );
  },
});

const checkboxStub = defineComponent({
  name: 'TCheckboxStub',
  props: {
    value: {
      type: Number,
      required: true,
    },
  },
  setup(props, { slots }) {
    return () => h('label', { 'data-permission-id': String(props.value) }, slots.default?.());
  },
});

function createRoleListResponse() {
  return {
    items: [
      {
        id: 2,
        name: 'editor',
        display: 'Editor',
        description: 'Editor role',
        builtin: false,
      },
    ],
  };
}

function createPermissionListResponse() {
  return {
    items: [
      {
        id: 1,
        code: 'permission.read',
        display: 'Permission Read',
        description: 'Read permissions',
        category: 'permission',
      },
      {
        id: 2,
        code: 'role.update',
        display: 'Role Update',
        description: 'Update roles',
        category: 'role',
      },
    ],
  };
}

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;

  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  return {
    promise,
    reject,
    resolve,
  };
}

function mountRolePage() {
  return mount(RolePage, {
    global: {
      components: {
        TButton: buttonStub,
        TCard: passthroughStub,
        TCheckbox: checkboxStub,
        TCheckboxGroup: checkboxGroupStub,
        TCol: passthroughStub,
        TDialog: dialogStub,
        TEmpty: passthroughStub,
        TForm: formStub,
        TFormItem: passthroughStub,
        TInput: inputStub,
        TRow: passthroughStub,
        TTable: tableStub,
        TTag: passthroughStub,
        TTextarea: textareaStub,
      },
      directives: {
        permission: {
          mounted() {},
        },
      },
    },
  });
}

function findButtonByText(wrapper: ReturnType<typeof mountRolePage>, text: string) {
  return wrapper.findAll('button').find((button) => button.text().trim() === text);
}

describe('RolePage', () => {
  beforeEach(() => {
    permissionState.grantedCodes = [];
    rbacApiMocks.assignRolePermissions.mockReset();
    rbacApiMocks.createRole.mockReset();
    rbacApiMocks.getPermissions.mockReset();
    rbacApiMocks.getRolePermissionBindings.mockReset();
    rbacApiMocks.getRoles.mockReset();
    rbacApiMocks.updateRole.mockReset();
    messageMocks.error.mockReset();
    messageMocks.success.mockReset();
    messageMocks.warning.mockReset();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('loads roles and permission definitions on mount when permission read is granted', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());

    const wrapper = mountRolePage();
    await flushPromises();

    expect(rbacApiMocks.getRoles).toHaveBeenCalledTimes(1);
    expect(rbacApiMocks.getPermissions).toHaveBeenCalledTimes(1);
    expect(wrapper.attributes('data-page-type')).toBe('list-form-detail');
    expect(wrapper.text()).toContain('editor');
    expect(wrapper.text()).toContain('rbac.roleList.permissionSummary');
    expect(wrapper.text()).toContain('rbac.roleList.actionTitle');
  });

  it('submits the trimmed create payload and appends the created role', async () => {
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue({
      items: [],
    });
    rbacApiMocks.createRole.mockResolvedValue({
      id: 4,
      name: 'reviewer',
      display: 'Reviewer',
      description: null,
      builtin: false,
    });

    const wrapper = mountRolePage();
    await flushPromises();

    const openCreateButton = findButtonByText(wrapper, 'rbac.roleList.create');
    expect(openCreateButton).toBeDefined();

    await openCreateButton!.trigger('click');
    await flushPromises();

    await wrapper.get('input[placeholder="rbac.roleList.form.namePlaceholder"]').setValue(' reviewer ');
    await wrapper.get('input[placeholder="rbac.roleList.form.displayPlaceholder"]').setValue(' Reviewer ');
    await wrapper.get('textarea[placeholder="rbac.roleList.form.descriptionPlaceholder"]').setValue('   ');
    await wrapper.get('[data-testid="role-form"]').trigger('submit');
    await flushPromises();

    expect(rbacApiMocks.createRole).toHaveBeenCalledWith({
      name: 'reviewer',
      display: 'Reviewer',
      description: null,
    });
    expect(messageMocks.success).toHaveBeenCalledWith('rbac.roleList.createSuccess');
    expect(wrapper.text()).toContain('reviewer');
  });

  it('submits the edited role payload through the update API', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_UPDATE];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue({
      items: [],
    });
    rbacApiMocks.updateRole.mockResolvedValue({
      id: 2,
      name: 'editor',
      display: 'Editorial Team',
      description: 'Updated summary',
      builtin: false,
    });

    const wrapper = mountRolePage();
    await flushPromises();

    const openEditButton = findButtonByText(wrapper, 'components.commonTable.detail');
    expect(openEditButton).toBeDefined();

    await openEditButton!.trigger('click');
    await flushPromises();

    await wrapper.get('input[placeholder="rbac.roleList.form.displayPlaceholder"]').setValue(' Editorial Team ');
    await wrapper
      .get('textarea[placeholder="rbac.roleList.form.descriptionPlaceholder"]')
      .setValue(' Updated summary ');
    await wrapper.get('[data-testid="role-form"]').trigger('submit');
    await flushPromises();

    expect(rbacApiMocks.updateRole).toHaveBeenCalledWith(2, {
      name: 'editor',
      display: 'Editorial Team',
      description: 'Updated summary',
    });
    expect(messageMocks.success).toHaveBeenCalledWith('rbac.roleList.updateSuccess');
    expect(wrapper.text()).toContain('Editorial Team');
  });

  it('keeps replace-write blocked when the current role-permission snapshot cannot be restored', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({
      permission_ids: [999],
    });

    const wrapper = mountRolePage();
    await flushPromises();

    const openPermissionButton = findButtonByText(wrapper, 'rbac.roleList.assignPermissions');
    expect(openPermissionButton).toBeDefined();

    await openPermissionButton!.trigger('click');
    await flushPromises();

    expect(rbacApiMocks.getRolePermissionBindings).toHaveBeenCalledWith(2);
    expect(wrapper.find('[data-header="rbac.roleList.permissionDialog.title"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('rbac.roleList.permissionDialog.selectionUnavailable');
    expect(findButtonByText(wrapper, 'rbac.roleList.permissionDialog.retry')).toBeUndefined();
    expect(wrapper.get('[data-testid="permission-checkbox-group"]').attributes('data-disabled')).toBe('true');
    expect(rbacApiMocks.assignRolePermissions).not.toHaveBeenCalled();
  });

  it('retries the permission dialog load in place after the role-permission snapshot fails to load', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings
      .mockRejectedValueOnce(new Error('selection load failed'))
      .mockResolvedValueOnce({
        permission_ids: [2],
      });

    const wrapper = mountRolePage();
    await flushPromises();

    const openPermissionButton = findButtonByText(wrapper, 'rbac.roleList.assignPermissions');
    expect(openPermissionButton).toBeDefined();

    await openPermissionButton!.trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('selection load failed');
    expect(
      wrapper.findAll('button').filter((button) => button.text().trim() === 'rbac.roleList.permissionDialog.retry'),
    ).toHaveLength(1);

    const retryButton = findButtonByText(wrapper, 'rbac.roleList.permissionDialog.retry');
    expect(retryButton).toBeDefined();

    await retryButton!.trigger('click');
    await flushPromises();

    expect(rbacApiMocks.getRolePermissionBindings).toHaveBeenCalledTimes(2);
    expect(wrapper.text()).not.toContain('selection load failed');
    expect(wrapper.get('[data-testid="permission-checkbox-group"]').attributes('data-disabled')).toBe('false');
    expect(wrapper.get('[data-testid="permission-checkbox-group"]').attributes('data-selected-permission-ids')).toBe(
      '[2]',
    );
  });

  it('shows a single loading status while the current role-permission snapshot is still loading', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    const permissionBindingRequest = createDeferred<{ permission_ids: number[] }>();

    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockImplementationOnce(() => permissionBindingRequest.promise);

    const wrapper = mountRolePage();
    await flushPromises();

    const openPermissionButton = findButtonByText(wrapper, 'rbac.roleList.assignPermissions');
    expect(openPermissionButton).toBeDefined();

    await openPermissionButton!.trigger('click');
    await wrapper.vm.$nextTick();

    expect(wrapper.find('[data-header="rbac.roleList.permissionDialog.title"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('rbac.roleList.permissionDialog.loadingSelection');
    expect(
      wrapper.findAll('button').filter((button) => button.text().trim() === 'rbac.roleList.permissionDialog.retry'),
    ).toHaveLength(0);
    expect(wrapper.get('[data-testid="permission-checkbox-group"]').attributes('data-disabled')).toBe('true');

    permissionBindingRequest.resolve({
      permission_ids: [1],
    });
    await flushPromises();

    expect(wrapper.text()).not.toContain('rbac.roleList.permissionDialog.loadingSelection');
    expect(wrapper.get('[data-testid="permission-checkbox-group"]').attributes('data-disabled')).toBe('false');
  });

  it('submits the restored permission snapshot for the selected role', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({
      permission_ids: [2, 1, 2],
    });
    rbacApiMocks.assignRolePermissions.mockResolvedValue(null);

    const wrapper = mountRolePage();
    await flushPromises();

    const openPermissionButton = findButtonByText(wrapper, 'rbac.roleList.assignPermissions');
    expect(openPermissionButton).toBeDefined();

    await openPermissionButton!.trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-checkbox-group"]').attributes('data-selected-permission-ids')).toBe(
      '[1,2]',
    );

    const submitButton = findButtonByText(wrapper, 'rbac.roleList.permissionDialog.confirm');
    expect(submitButton).toBeDefined();

    await submitButton!.trigger('click');
    await flushPromises();

    expect(rbacApiMocks.assignRolePermissions).toHaveBeenCalledWith(2, {
      permission_ids: [1, 2],
    });
    expect(messageMocks.success).toHaveBeenCalledWith('rbac.roleList.assignSuccess');
    expect(wrapper.find('[data-header="rbac.roleList.permissionDialog.title"]').exists()).toBe(false);
  });

  it('keeps permission assignment enabled after refreshing the page data while the dialog stays open', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({
      permission_ids: [2],
    });
    rbacApiMocks.assignRolePermissions.mockResolvedValue(null);

    const wrapper = mountRolePage();
    await flushPromises();

    const openPermissionButton = findButtonByText(wrapper, 'rbac.roleList.assignPermissions');
    expect(openPermissionButton).toBeDefined();

    await openPermissionButton!.trigger('click');
    await flushPromises();

    const refreshButton = findButtonByText(wrapper, 'rbac.roleList.refresh');
    expect(refreshButton).toBeDefined();

    const initialSubmitButton = findButtonByText(wrapper, 'rbac.roleList.permissionDialog.confirm');
    expect(initialSubmitButton).toBeDefined();
    expect((initialSubmitButton!.element as HTMLButtonElement).disabled).toBe(false);

    await refreshButton!.trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-checkbox-group"]').attributes('data-selected-permission-ids')).toBe(
      '[2]',
    );

    const submitButton = findButtonByText(wrapper, 'rbac.roleList.permissionDialog.confirm');
    expect(submitButton).toBeDefined();
    expect((submitButton!.element as HTMLButtonElement).disabled).toBe(false);

    await submitButton!.trigger('click');
    await flushPromises();

    expect(rbacApiMocks.assignRolePermissions).toHaveBeenCalledWith(2, {
      permission_ids: [2],
    });
  });

  it('shows an error toast and keeps the dialog open when the permission assignment submit fails', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({
      permission_ids: [2],
    });
    rbacApiMocks.assignRolePermissions.mockRejectedValue(new Error('assign failed'));

    const wrapper = mountRolePage();
    await flushPromises();

    const openPermissionButton = findButtonByText(wrapper, 'rbac.roleList.assignPermissions');
    expect(openPermissionButton).toBeDefined();

    await openPermissionButton!.trigger('click');
    await flushPromises();

    const submitButton = findButtonByText(wrapper, 'rbac.roleList.permissionDialog.confirm');
    expect(submitButton).toBeDefined();

    await submitButton!.trigger('click');
    await flushPromises();

    expect(messageMocks.error).toHaveBeenCalledWith('assign failed');
    expect(wrapper.find('[data-header="rbac.roleList.permissionDialog.title"]').exists()).toBe(true);
  });

  it('ignores stale role-permission responses after the dialog is closed and reopened', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    const firstPermissionBindingRequest = createDeferred<{ permission_ids: number[] }>();

    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings
      .mockImplementationOnce(() => firstPermissionBindingRequest.promise)
      .mockResolvedValueOnce({
        permission_ids: [],
      });

    const wrapper = mountRolePage();
    await flushPromises();

    const openPermissionButton = findButtonByText(wrapper, 'rbac.roleList.assignPermissions');
    expect(openPermissionButton).toBeDefined();

    await openPermissionButton!.trigger('click');
    await flushPromises();

    const cancelButton = findButtonByText(wrapper, 'rbac.roleList.form.cancel');
    expect(cancelButton).toBeDefined();

    await cancelButton!.trigger('click');
    await flushPromises();

    await openPermissionButton!.trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-checkbox-group"]').attributes('data-selected-permission-ids')).toBe(
      '[]',
    );

    firstPermissionBindingRequest.resolve({
      permission_ids: [1, 2],
    });
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-checkbox-group"]').attributes('data-selected-permission-ids')).toBe(
      '[]',
    );
    expect(rbacApiMocks.getRolePermissionBindings).toHaveBeenCalledTimes(2);
  });

  it('ignores stale submit errors after the dialog is closed and reopened', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    const firstAssignRequest = createDeferred<null>();

    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({
      permission_ids: [2],
    });
    rbacApiMocks.assignRolePermissions
      .mockImplementationOnce(() => firstAssignRequest.promise)
      .mockResolvedValueOnce(null);

    const wrapper = mountRolePage();
    await flushPromises();

    const openPermissionButton = findButtonByText(wrapper, 'rbac.roleList.assignPermissions');
    expect(openPermissionButton).toBeDefined();

    await openPermissionButton!.trigger('click');
    await flushPromises();

    const submitButton = findButtonByText(wrapper, 'rbac.roleList.permissionDialog.confirm');
    expect(submitButton).toBeDefined();

    await submitButton!.trigger('click');
    await flushPromises();

    const cancelButton = findButtonByText(wrapper, 'rbac.roleList.form.cancel');
    expect(cancelButton).toBeDefined();

    await cancelButton!.trigger('click');
    await flushPromises();
    await openPermissionButton!.trigger('click');
    await flushPromises();

    firstAssignRequest.reject(new Error('stale failure'));
    await flushPromises();

    expect(messageMocks.error).not.toHaveBeenCalled();
    expect(wrapper.find('[data-header="rbac.roleList.permissionDialog.title"]').exists()).toBe(true);
  });

  it('ignores stale submit success after the dialog is closed and reopened', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    const firstAssignRequest = createDeferred<null>();

    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({
      permission_ids: [2],
    });
    rbacApiMocks.assignRolePermissions
      .mockImplementationOnce(() => firstAssignRequest.promise)
      .mockResolvedValueOnce(null);

    const wrapper = mountRolePage();
    await flushPromises();

    const openPermissionButton = findButtonByText(wrapper, 'rbac.roleList.assignPermissions');
    expect(openPermissionButton).toBeDefined();

    await openPermissionButton!.trigger('click');
    await flushPromises();

    const submitButton = findButtonByText(wrapper, 'rbac.roleList.permissionDialog.confirm');
    expect(submitButton).toBeDefined();

    await submitButton!.trigger('click');
    await flushPromises();

    const cancelButton = findButtonByText(wrapper, 'rbac.roleList.form.cancel');
    expect(cancelButton).toBeDefined();

    await cancelButton!.trigger('click');
    await flushPromises();
    await openPermissionButton!.trigger('click');
    await flushPromises();

    firstAssignRequest.resolve(null);
    await flushPromises();

    expect(messageMocks.success).not.toHaveBeenCalled();
    expect(wrapper.find('[data-header="rbac.roleList.permissionDialog.title"]').exists()).toBe(true);
  });
});
