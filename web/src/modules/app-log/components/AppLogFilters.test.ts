import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import type { AppLogFilterState } from '../types/app-log';
import AppLogFilters from './AppLogFilters.vue';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

const baseFilters: AppLogFilterState = {
  keyword: '',
  occurredRange: [],
  severity: '',
  component: '',
  operation: '',
  requestId: '',
  traceId: '',
  message: '',
  error: '',
};

describe('AppLogFilters', () => {
  it('emits canonical filter state updates', async () => {
    const wrapper = mount(AppLogFilters, {
      props: {
        modelValue: baseFilters,
      },
      global: {
        stubs: {
          TForm: { template: '<form><slot /></form>' },
          TFormItem: { template: '<label><slot /></label>' },
          TInput: {
            props: ['value'],
            emits: ['change', 'enter'],
            template: '<input :value="value" @input="$emit(\'change\', $event.target.value)" />',
          },
          TDateRangePicker: true,
          TSelect: true,
          TButton: { template: '<button><slot /></button>' },
        },
      },
    });

    await wrapper.find('input').setValue('startup failed');

    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toMatchObject({
      keyword: 'startup failed',
    });
  });
});
