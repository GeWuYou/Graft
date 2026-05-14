<template>
  <t-card v-if="product" theme="poster2" :bordered="false">
    <template #avatar>
      <t-avatar size="56px">
        <template #icon>
          <shop-icon v-if="product.type === 1" />
          <calendar-icon v-if="product.type === 2" />
          <service-icon v-if="product.type === 3" />
          <user-avatar-icon v-if="product.type === 4" />
          <laptop-icon v-if="product.type === 5" />
        </template>
      </t-avatar>
    </template>
    <template #status>
      <t-tag :theme="product.isSetup ? 'success' : 'default'" :disabled="!product.isSetup">
        {{ product.isSetup ? t('components.isSetup.on') : t('components.isSetup.off') }}
      </t-tag>
    </template>
    <template #content>
      <p class="list-card-item_detail--name">{{ product.name }}</p>
      <p class="list-card-item_detail--desc">{{ product.description }}</p>
    </template>
    <template #footer>
      <t-avatar-group cascading="left-up" :max="2">
        <t-avatar>{{ typeMap[product.type - 1] }}</t-avatar>
        <t-avatar
          ><template #icon>
            <add-icon />
          </template>
        </t-avatar>
      </t-avatar-group>
    </template>
    <template #actions>
      <t-dropdown
        :disabled="!product.isSetup"
        trigger="click"
        :options="[
          {
            content: t('components.manage'),
            value: 'manage',
            onClick: () => handleClickManage(product),
          },
          {
            content: t('components.delete'),
            value: 'delete',
            onClick: () => handleClickDelete(product),
          },
        ]"
      >
        <t-button theme="default" :disabled="!product.isSetup" shape="square" variant="text">
          <more-icon />
        </t-button>
      </t-dropdown>
    </template>
  </t-card>
</template>
<script setup lang="ts">
import {
  AddIcon,
  CalendarIcon,
  LaptopIcon,
  MoreIcon,
  ServiceIcon,
  ShopIcon,
  UserAvatarIcon,
} from 'tdesign-icons-vue-next';
import type { PropType } from 'vue';

import { t } from '@/locales';

export interface CardProductType {
  index: number;
  type: number;
  isSetup: boolean;
  description: string;
  name: string;
}

const props = defineProps({
  product: {
    type: Object as PropType<CardProductType>,
    default: undefined,
  },
});

const product = props.product;

const emit = defineEmits(['manage-product', 'delete-item']);

const typeMap = ['A', 'B', 'C', 'D', 'E'];

const handleClickManage = (product: CardProductType) => {
  emit('manage-product', product);
};

const handleClickDelete = (product: CardProductType) => {
  emit('delete-item', product);
};
</script>
<style lang="less" scoped>
.list-card-item {
  cursor: pointer;
  display: flex;
  flex-direction: column;

  &_detail {
    min-height: 140px;

    &--name {
      color: var(--td-text-color-primary);
      font: var(--td-font-title-medium);
      margin-bottom: var(--td-comp-margin-s);
    }

    &--desc {
      -webkit-box-orient: vertical;
      color: var(--td-text-color-secondary);
      display: -webkit-box;
      font: var(--td-font-body-small);
      -webkit-line-clamp: 2;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  }
}
</style>
