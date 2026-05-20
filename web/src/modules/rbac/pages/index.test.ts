import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { RBAC_PERMISSION_CODE } from '../contract/permissions';
import RolePage from './index.vue';

const rbacApiMocks = vi.hoisted(() => ({
  assignRolePermissions: vi.fn(),
  createRole: vi.fn(),
  getPermissions: vi.fn(),
  getRolePermissionBindings: vi.fn(),
  getRoles: vi.fn(),
  updateRole: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
  warning: vi.fn(),
}));

const permissionState = vi.hoisted(() => ({
  grantedCodes: [] as string[],
}));

vi.mock('../api/rbac', () => ({
  assignRolePermissions: rbacApiMocks.assignRolePermissions,
  createRole: rbacApiMocks.createRole,
  getPermissions: rbacApiMocks.getPermissions,
  getRolePermissionBindings: rbacApiMocks.getRolePermissionBindings,
  getRoles: rbacApiMocks.getRoles,
  updateRole: rbacApiMocks.updateRole,
}));

vi.mock('@/store', () => ({
  usePermissionStore: () => ({
    hasPermission: (code: string) => permissionState.grantedCodes.includes(code),
  }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    locale: {
      value: 'en-US',
    },
  }),
}));

vi.mock('tdesign-vue-next', () => ({
  MessagePlugin: {
    error: messageMocks.error,
    success: messageMocks.success,
    warning: messageMocks.warning,
  },
}));

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

function mountRolePage() {
  return mount(RolePage, {
    global: {
      directives: {
        permission: {
          mounted() {},
        },
      },
      stubs: {
        't-button': buttonStub,
        't-card': passthroughStub,
        't-checkbox': checkboxStub,
        't-checkbox-group': checkboxGroupStub,
        't-col': passthroughStub,
        't-dialog': dialogStub,
        't-empty': passthroughStub,
        't-form': formStub,
        't-form-item': passthroughStub,
        't-input': inputStub,
        't-row': passthroughStub,
        't-table': tableStub,
        't-tag': passthroughStub,
        't-textarea': textareaStub,
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
    expect(wrapper.text()).toContain('editor');
    expect(wrapper.text()).toContain('rbac.roleList.permissionSummary');
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

  it('blocks permission replacement when the current snapshot cannot be restored safely', async () => {
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
    expect(messageMocks.error).toHaveBeenCalledWith('rbac.roleList.permissionDialog.selectionUnavailable');
    expect(wrapper.find('[data-header="rbac.roleList.permissionDialog.title"]').exists()).toBe(false);
    expect(rbacApiMocks.assignRolePermissions).not.toHaveBeenCalled();
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
});
