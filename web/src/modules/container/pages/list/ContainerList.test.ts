import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import ContainerList from './ContainerList.vue';
import sourceText from './ContainerList.vue?raw';

vi.mock('@/shared/composables', () => ({
  useResponsiveVariant: () => ({ value: { density: 'spacious' } }),
}));

const ContainerResourceTableStub = defineComponent({
  name: 'ContainerResourceTableStub',
  setup(_, { slots }) {
    return () => h('section', { 'data-testid': 'resource-table' }, slots.cards?.());
  },
});

const ContainerCardStub = defineComponent({
  name: 'ContainerCardStub',
  props: ['row'],
  setup(props) {
    return () => h('article', { class: 'container-card-stub' }, props.row.id);
  },
});

function mountList() {
  return mount(ContainerList, {
    global: {
      stubs: {
        ContainerCard: ContainerCardStub,
        ContainerResourceTable: ContainerResourceTableStub,
      },
    },
    props: {
      alwaysVisibleColumnKeys: [],
      composeApplicationReferences: new Map(),
      current: 1,
      emptyDescription: 'No containers',
      emptyTitle: 'Empty',
      footerSummary: '2 containers',
      headDescription: 'Containers',
      headSummary: '2 containers',
      loading: false,
      moreActionsLabel: 'More',
      pageSize: 10,
      presentation: 'card',
      rowActions: () => [],
      rows: [{ id: 'container-1' }, { id: 'container-2' }] as never,
      selectedRowKeys: [],
      tableDensity: 'medium',
      total: 2,
      visibleColumnKeys: [],
    },
  });
}

describe('ContainerList', () => {
  it('renders only container cards in the cards grid without a last-card span override', () => {
    const wrapper = mountList();
    const cards = wrapper.get('.container-list__cards');

    expect(cards.element.children).toHaveLength(2);
    expect(cards.findAll('.container-card-stub')).toHaveLength(2);
    expect(sourceText).not.toContain('.container-list__cards > :last-child');
    expect(sourceText).not.toContain('grid-column: 1 / -1');
  });
});
