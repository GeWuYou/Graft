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
    hasAnyPermission: (codes: string[]) => codes.some((code) => permissionState.grantedCodes.includes(code)),
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

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: {},
  }),
  useRouter: () => ({
    replace: vi.fn(),
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
    return () => h('div', [props.title, props.description, slots.default?.(), slots.action?.()]);
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
  setup(props, { emit, slots, attrs }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
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

const inputStub = defineComponent({
  name: 'TInputStub',
  props: {
    modelValue: {
      type: String,
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () =>
      h('input', {
        ...attrs,
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
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
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () =>
      h('textarea', {
        ...attrs,
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
      });
  },
});

const selectStub = defineComponent({
  name: 'TSelectStub',
  props: {
    modelValue: {
      type: [String, Number, null],
      default: null,
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () =>
      h(
        'select',
        {
          ...attrs,
          value: props.modelValue ?? '',
          onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value || null),
        },
        (props.options as Array<{ label: string; value: string | number | null }>).map((option) =>
          h('option', { value: option.value ?? '' }, option.label),
        ),
      );
  },
});

const tableStub = defineComponent({
  name: 'TTableStub',
  props: {
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

      return h(
        'div',
        (props.data as Array<Record<string, unknown>>).map((row, index) =>
          h('div', { 'data-testid': `role-row-${index}` }, [slots.role?.({ row }), slots.operation?.({ row })]),
        ),
      );
    };
  },
});

const drawerStub = defineComponent({
  name: 'TDrawerStub',
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
      props.visible ? h('section', { 'data-testid': 'drawer', 'data-header': props.header }, slots.default?.()) : null;
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
          'data-testid': 'permission-checkbox-group',
          'data-disabled': String(props.disabled),
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
      type: [Number, Boolean],
      default: undefined,
    },
  },
  setup(props, { slots }) {
    return () => h('label', { 'data-permission-id': String(props.value ?? '') }, slots.default?.());
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
        updated_at: '2026-05-18T00:00:00Z',
        permission_count: 2,
        user_count: 1,
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
        role_binding_count: 1,
      },
      {
        id: 2,
        code: 'role.update',
        display: 'Role Update',
        description: 'Update roles',
        category: 'role',
        role_binding_count: 1,
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
        't-checkbox': checkboxStub,
        't-checkbox-group': checkboxGroupStub,
        't-dropdown': passthroughStub,
        't-drawer': drawerStub,
        't-empty': passthroughStub,
        't-form': formStub,
        't-form-item': passthroughStub,
        't-input': inputStub,
        't-select': selectStub,
        't-table': tableStub,
        't-tag': passthroughStub,
        't-textarea': textareaStub,
        't-pagination': passthroughStub,
        't-tooltip': passthroughStub,
      },
    },
  });
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

  it('loads roles and permission definitions on mount', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());

    const wrapper = mountRolePage();
    await flushPromises();

    expect(rbacApiMocks.getRoles).toHaveBeenCalledTimes(1);
    expect(rbacApiMocks.getPermissions).toHaveBeenCalledTimes(1);
    expect(wrapper.attributes('data-page-type')).toBe('list-form-detail');
    expect(wrapper.text()).toContain('Editor');
    expect(wrapper.text()).not.toContain('rbac.roleList.stats.totalRoles');
  });

  it('submits the trimmed create payload', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_CREATE];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });
    rbacApiMocks.createRole.mockResolvedValue({
      id: 4,
      name: 'reviewer',
      display: 'Reviewer',
      description: null,
      builtin: false,
      updated_at: '2026-05-19T00:00:00Z',
      permission_count: 0,
      user_count: 0,
    });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-create"]').trigger('click');
    await flushPromises();
    await wrapper.get('input[placeholder="rbac.roleList.form.namePlaceholder"]').setValue(' reviewer ');
    await wrapper.get('input[placeholder="rbac.roleList.form.displayPlaceholder"]').setValue(' Reviewer ');
    await wrapper.get('textarea[placeholder="rbac.roleList.form.descriptionPlaceholder"]').setValue(' role summary ');
    await wrapper.get('[data-testid="role-form"]').trigger('submit');
    await flushPromises();

    expect(rbacApiMocks.createRole).toHaveBeenCalledWith({
      name: 'reviewer',
      display: 'Reviewer',
      description: 'role summary',
    });
    expect(messageMocks.success).toHaveBeenCalledWith('rbac.roleList.createSuccess');
  });

  it('submits the restored permission snapshot for the selected role', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [2, 1, 2] });
    rbacApiMocks.assignRolePermissions.mockResolvedValue(null);

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="permission-drawer-save"]').trigger('click');
    await flushPromises();

    expect(rbacApiMocks.assignRolePermissions).toHaveBeenCalledWith(2, {
      permission_ids: [1, 2],
    });
    expect(messageMocks.success).toHaveBeenCalledWith('rbac.roleList.assignSuccess');
  });

  it('renders the table empty state, clears filters, and opens the create drawer from the empty action', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_CREATE];
    rbacApiMocks.getRoles.mockResolvedValue({ items: [] });
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });

    const wrapper = mountRolePage();
    await flushPromises();

    expect(wrapper.text()).toContain('rbac.roleList.emptyTitle');
    expect(wrapper.text()).toContain('rbac.roleList.emptyDescription');
    expect(wrapper.find('[data-testid="role-empty-clear-filters"]').exists()).toBe(false);

    await wrapper.get('input[placeholder="rbac.roleList.toolbar.searchPlaceholder"]').setValue('editor');
    await flushPromises();

    expect(wrapper.find('[data-testid="role-empty-clear-filters"]').exists()).toBe(true);
    await wrapper.get('[data-testid="role-empty-clear-filters"]').trigger('click');
    await flushPromises();

    expect(
      (wrapper.get('input[placeholder="rbac.roleList.toolbar.searchPlaceholder"]').element as HTMLInputElement).value,
    ).toBe('');

    await wrapper.get('[data-testid="role-empty-create"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="drawer"]').exists()).toBe(true);
  });
});
