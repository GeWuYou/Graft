import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import { RBAC_PERMISSION_CODE } from '../contract/permissions';
import RolePage from './index.vue';

const rbacApiMocks = vi.hoisted(() => ({
  addRolePermissions: vi.fn(),
  createRole: vi.fn(),
  deleteRole: vi.fn(),
  getPermissions: vi.fn(),
  getRoleDetail: vi.fn(),
  getRolePermissionBindings: vi.fn(),
  getRoles: vi.fn(),
  removeRolePermissions: vi.fn(),
  replaceRolePermissions: vi.fn(),
  updateRoleStatus: vi.fn(),
  updateRole: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
  warning: vi.fn(),
}));

const confirmMock = vi.hoisted(() => vi.fn(() => true));

const permissionState = vi.hoisted(() => ({
  grantedCodes: [] as string[],
}));

vi.mock('../api/rbac', () => ({
  addRolePermissions: rbacApiMocks.addRolePermissions,
  createRole: rbacApiMocks.createRole,
  deleteRole: rbacApiMocks.deleteRole,
  getPermissions: rbacApiMocks.getPermissions,
  getRoleDetail: rbacApiMocks.getRoleDetail,
  getRolePermissionBindings: rbacApiMocks.getRolePermissionBindings,
  getRoles: rbacApiMocks.getRoles,
  removeRolePermissions: rbacApiMocks.removeRolePermissions,
  replaceRolePermissions: rbacApiMocks.replaceRolePermissions,
  updateRoleStatus: rbacApiMocks.updateRoleStatus,
  updateRole: rbacApiMocks.updateRole,
}));

vi.mock('@/store', () => ({
  usePermissionStore: () => ({
    hasAnyPermission: (codes: string[]) => codes.some((code) => permissionState.grantedCodes.includes(code)),
    hasPermission: (code: string) => permissionState.grantedCodes.includes(code),
  }),
}));

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>();
  const translations: Record<string, string> = {
    'rbac.permissionCatalog.permissionRead.display': 'Read Permissions Localized',
    'rbac.permissionCatalog.permissionRead.description': 'Localized permission description',
    'rbac.permissionCatalog.roleUpdate.display': 'Update Roles Localized',
    'rbac.permissionCatalog.roleUpdate.description': 'Localized update-role description',
  };
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => translations[key] ?? key,
      locale: {
        value: 'en-US',
      },
    }),
  };
});

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

vi.stubGlobal('confirm', confirmMock);

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

const dropdownStub = defineComponent({
  name: 'TDropdownStub',
  props: {
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'div',
        {
          'data-testid': 'dropdown',
          'data-options': JSON.stringify(props.options),
          onClick: () => emit('click', { value: 'noop' }),
        },
        slots.default?.(),
      );
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
    footer: {
      type: [Boolean, Object, String, null],
      default: undefined,
    },
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
        ? h(
            'section',
            {
              'data-testid': 'drawer',
              'data-footer': String(props.footer),
              'data-header': props.header,
            },
            slots.default?.(),
          )
        : null;
  },
});

const formStub = defineComponent({
  name: 'TFormStub',
  props: {
    data: {
      type: Object,
      default: () => ({}),
    },
    rules: {
      type: Object,
      default: () => ({}),
    },
  },
  emits: ['submit'],
  setup(props, { emit, expose, slots }) {
    const validateMessages = ref<Record<string, string[]>>({});

    function clearValidate(fields?: string[]) {
      if (!fields?.length) {
        validateMessages.value = {};
        return;
      }

      const nextMessages = { ...validateMessages.value };
      fields.forEach((field) => {
        delete nextMessages[field];
      });
      validateMessages.value = nextMessages;
    }

    function setValidateMessage(message: Record<string, Array<{ message: string }>>) {
      const nextMessages = { ...validateMessages.value };
      Object.entries(message).forEach(([field, items]) => {
        nextMessages[field] = items.map((item) => item.message);
      });
      validateMessages.value = nextMessages;
    }

    expose({
      clearValidate,
      setValidateMessage,
    });

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
        [
          slots.default?.({ data: props.data }),
          Object.entries(validateMessages.value).map(([field, messages]) =>
            h(
              'div',
              { 'data-testid': `validate-${field}` },
              messages.map((message) => h('p', message)),
            ),
          ),
        ],
      );
  },
});

const formItemStub = defineComponent({
  name: 'TFormItemStub',
  props: {
    label: {
      type: String,
      default: '',
    },
    name: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-testid': `form-item-${props.name}` }, [
        props.label ? h('label', props.label) : null,
        slots.default?.(),
      ]);
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
  emits: ['update:modelValue'],
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
    disabled: {
      type: Boolean,
      default: false,
    },
    value: {
      type: [Number, Boolean],
      default: undefined,
    },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'label',
        {
          'data-disabled': String(props.disabled),
          'data-permission-id': String(props.value ?? ''),
        },
        slots.default?.(),
      );
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
        status: 'enabled',
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
          mounted(el, binding) {
            const value = binding.value;
            let allowed = false;

            if (typeof value === 'string') {
              allowed = permissionState.grantedCodes.includes(value);
            } else if (Array.isArray(value)) {
              allowed = value.every((code: string) => permissionState.grantedCodes.includes(code));
            } else if (value && typeof value === 'object') {
              const allOf = Array.isArray(value.allOf) ? value.allOf : [];
              const anyOf = Array.isArray(value.anyOf) ? value.anyOf : [];
              const matchesAll =
                allOf.length === 0 || allOf.every((code: string) => permissionState.grantedCodes.includes(code));
              const matchesAny =
                anyOf.length === 0 || anyOf.some((code: string) => permissionState.grantedCodes.includes(code));
              allowed = matchesAll && matchesAny;
            }

            if (!allowed) {
              el.remove();
            }
          },
        },
      },
      stubs: {
        't-button': buttonStub,
        't-checkbox': checkboxStub,
        't-checkbox-group': checkboxGroupStub,
        't-dropdown': dropdownStub,
        't-drawer': drawerStub,
        't-empty': passthroughStub,
        't-form': formStub,
        't-form-item': formItemStub,
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

function updatePermissionSelection(wrapper: ReturnType<typeof mountRolePage>, ids: number[]) {
  const checkboxGroup = wrapper.getComponent(checkboxGroupStub);
  checkboxGroup.vm.$emit('update:modelValue', ids);
}

function setPermissionMutationMode(wrapper: ReturnType<typeof mountRolePage>, mode: 'replace' | 'add' | 'remove') {
  const selects = wrapper.findAll('select');
  const mutationSelect = selects.at(-1);
  if (!mutationSelect) {
    throw new Error('permission mutation select not found');
  }

  return mutationSelect.setValue(mode);
}

describe('RolePage', () => {
  beforeEach(() => {
    permissionState.grantedCodes = [];
    confirmMock.mockReset();
    confirmMock.mockReturnValue(true);
    rbacApiMocks.addRolePermissions.mockReset();
    rbacApiMocks.createRole.mockReset();
    rbacApiMocks.deleteRole.mockReset();
    rbacApiMocks.getPermissions.mockReset();
    rbacApiMocks.getRoleDetail.mockReset();
    rbacApiMocks.getRolePermissionBindings.mockReset();
    rbacApiMocks.getRoles.mockReset();
    rbacApiMocks.removeRolePermissions.mockReset();
    rbacApiMocks.replaceRolePermissions.mockReset();
    rbacApiMocks.updateRoleStatus.mockReset();
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

  it('hides the edit button when role.update is missing', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());

    const wrapper = mountRolePage();
    await flushPromises();

    expect(wrapper.find('[data-testid="role-edit"]').exists()).toBe(false);
  });

  it('hides the create action when role.create is missing', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());

    const wrapper = mountRolePage();
    await flushPromises();

    expect(wrapper.find('[data-testid="role-create"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="role-empty-create"]').exists()).toBe(false);
  });

  it('hides the assign-permissions action when role.permission.assign is missing', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());

    const wrapper = mountRolePage();
    await flushPromises();

    expect(wrapper.find('[data-testid="role-assign-permissions"]').exists()).toBe(false);
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

  it('binds role form invalid-argument errors to the name field', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_CREATE];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });
    rbacApiMocks.createRole.mockRejectedValue({
      isApiRequestError: true,
      status: 400,
      code: 'COMMON_INVALID_ARGUMENT',
      message: '角色编码已存在',
      messageKey: '',
      responseData: {
        data: {
          field: 'name',
        },
      },
    });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-create"]').trigger('click');
    await flushPromises();
    await wrapper.get('input[placeholder="rbac.roleList.form.namePlaceholder"]').setValue('reviewer');
    await wrapper.get('input[placeholder="rbac.roleList.form.displayPlaceholder"]').setValue('Reviewer');
    await wrapper.get('[data-testid="role-form"]').trigger('submit');
    await flushPromises();

    expect(wrapper.get('[data-testid="validate-name"]').text()).toContain('角色编码已存在');
    expect(messageMocks.error).not.toHaveBeenCalled();
  });

  it('keeps replace mode submit disabled until the selection changes', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [2, 1, 2] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-drawer-save"]').attributes('disabled')).toBeDefined();
    expect(rbacApiMocks.replaceRolePermissions).not.toHaveBeenCalled();
  });

  it('allows clearing all permissions in replace mode', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1, 2] });
    rbacApiMocks.replaceRolePermissions.mockResolvedValue(null);

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    updatePermissionSelection(wrapper, []);
    await flushPromises();
    expect(wrapper.get('[data-testid="permission-drawer-save"]').attributes('disabled')).toBeUndefined();

    await wrapper.get('[data-testid="permission-drawer-save"]').trigger('click');
    await flushPromises();

    expect(rbacApiMocks.replaceRolePermissions).toHaveBeenCalledWith(2, {
      permission_ids: [],
    });
  });

  it('keeps replace mode submit disabled when the selection is unchanged', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-drawer-save"]').attributes('disabled')).toBeDefined();
  });

  it('disables the drawer default footer for permission assignment', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="drawer"]').attributes('data-footer')).toBe('false');
  });

  it('prompts before closing the permission drawer with unsaved changes', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });
    confirmMock.mockReturnValueOnce(false);

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    updatePermissionSelection(wrapper, [1, 2]);
    await flushPromises();

    await wrapper.get('[data-testid="permission-drawer-cancel"]').trigger('click');
    await flushPromises();

    expect(confirmMock).toHaveBeenCalledWith('rbac.roleList.permissionDialog.unsavedChangesConfirm');
    expect(wrapper.find('[data-testid="permission-drawer"]').exists()).toBe(true);
  });

  it('submits only newly selected permissions in add mode', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });
    rbacApiMocks.addRolePermissions.mockResolvedValue(null);

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    await setPermissionMutationMode(wrapper, 'add');
    await flushPromises();
    updatePermissionSelection(wrapper, [2]);
    await flushPromises();
    await wrapper.get('[data-testid="permission-drawer-save"]').trigger('click');
    await flushPromises();

    expect(rbacApiMocks.addRolePermissions).toHaveBeenCalledWith(2, {
      permission_ids: [2],
    });
    expect(rbacApiMocks.replaceRolePermissions).not.toHaveBeenCalled();
    expect(rbacApiMocks.removeRolePermissions).not.toHaveBeenCalled();
  });

  it('submits only removed permissions in remove mode', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1, 2] });
    rbacApiMocks.removeRolePermissions.mockResolvedValue(null);

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    await setPermissionMutationMode(wrapper, 'remove');
    await flushPromises();
    updatePermissionSelection(wrapper, [1]);
    await flushPromises();
    await wrapper.get('[data-testid="permission-drawer-save"]').trigger('click');
    await flushPromises();

    expect(rbacApiMocks.removeRolePermissions).toHaveBeenCalledWith(2, {
      permission_ids: [1],
    });
    expect(rbacApiMocks.replaceRolePermissions).not.toHaveBeenCalled();
    expect(rbacApiMocks.addRolePermissions).not.toHaveBeenCalled();
  });

  it('resets to explicit removal selection when switching to remove mode', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1, 2] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    await setPermissionMutationMode(wrapper, 'remove');
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-drawer-save"]').attributes('disabled')).toBeDefined();

    updatePermissionSelection(wrapper, [2]);
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-drawer-save"]').attributes('disabled')).toBeUndefined();
  });

  it('renders localized permission copy inside the assignment drawer', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Read Permissions Localized');
    expect(wrapper.text()).toContain('Localized permission description');
    expect(wrapper.text()).not.toContain('Permission Read');
    expect(wrapper.text()).not.toContain('Read permissions');
  });

  it('matches permission search against localized copy', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    await wrapper.get('input[placeholder="rbac.roleList.permissionDialog.searchPlaceholder"]').setValue('Localized');
    await flushPromises();

    expect(wrapper.text()).toContain('Read Permissions Localized');
    expect(wrapper.text()).toContain('Update Roles Localized');
  });

  it('shows lifecycle guidance in the detail drawer for role delete semantics', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRoleDetail.mockResolvedValue({
      id: 2,
      name: 'editor',
      display: 'Editor',
      description: 'Editor role',
      builtin: false,
      status: 'enabled',
      updated_at: '2026-05-18T00:00:00Z',
      created_at: '2026-05-17T00:00:00Z',
      permission_count: 2,
      user_count: 1,
    });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-detail"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="role-lifecycle-summary"]').text()).toContain(
      'rbac.roleList.lifecycle.statusLabel',
    );
    expect(wrapper.get('[data-testid="role-lifecycle-summary"]').text()).toContain(
      'rbac.roleList.lifecycle.statusEnabled',
    );
    expect(wrapper.get('[data-testid="role-lifecycle-summary"]').text()).toContain(
      'rbac.roleList.lifecycle.deleteNeedsDisable',
    );
  });

  it('blocks delete when the role is still enabled', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_DELETE];
    rbacApiMocks.getRoles.mockResolvedValue({
      items: [
        {
          ...createRoleListResponse().items[0],
          permission_count: 0,
          user_count: 0,
          status: 'enabled',
        },
      ],
    });
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });

    const wrapper = mountRolePage();
    await flushPromises();

    wrapper.getComponent(dropdownStub).vm.$emit('click', { value: 'delete' });
    await flushPromises();

    expect(rbacApiMocks.deleteRole).not.toHaveBeenCalled();
    expect(messageMocks.warning).toHaveBeenCalledWith('rbac.roleList.lifecycle.deleteNeedsDisable');
  });

  it('blocks delete when the disabled role still has bindings', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_DELETE];
    rbacApiMocks.getRoles.mockResolvedValue({
      items: [
        {
          ...createRoleListResponse().items[0],
          permission_count: 1,
          user_count: 0,
          status: 'disabled',
        },
      ],
    });
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });

    const wrapper = mountRolePage();
    await flushPromises();

    wrapper.getComponent(dropdownStub).vm.$emit('click', { value: 'delete' });
    await flushPromises();

    expect(rbacApiMocks.deleteRole).not.toHaveBeenCalled();
    expect(messageMocks.warning).toHaveBeenCalledWith('rbac.roleList.lifecycle.deleteNeedsBindingsCleared');
  });

  it('keeps permission assignment errors local when the backend rejects permission_ids', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [2, 1, 2] });
    rbacApiMocks.replaceRolePermissions.mockRejectedValue({
      isApiRequestError: true,
      status: 400,
      code: 'COMMON_INVALID_ARGUMENT',
      message: '权限列表包含无效条目',
      messageKey: '',
      responseData: {
        data: {
          field: 'permission_ids',
        },
      },
    });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    updatePermissionSelection(wrapper, [2]);
    await flushPromises();
    await wrapper.get('[data-testid="permission-drawer-save"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('权限列表包含无效条目');
    expect(messageMocks.error).not.toHaveBeenCalledWith('rbac.roleList.assignFailed');
  });

  it('keeps role-not-found assignment failures in the permission drawer feedback surface', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [2, 1, 2] });
    rbacApiMocks.replaceRolePermissions.mockRejectedValue({
      isApiRequestError: true,
      status: 404,
      code: 'ROLE_NOT_FOUND',
      message: '角色不存在',
      messageKey: 'role.not_found',
      responseData: {},
    });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    updatePermissionSelection(wrapper, [2]);
    await flushPromises();
    await wrapper.get('[data-testid="permission-drawer-save"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('角色不存在');
    expect(messageMocks.error).not.toHaveBeenCalledWith('rbac.roleList.assignFailed');
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
