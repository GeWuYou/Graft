<template>
  <t-popup expand-animation placement="bottom-right" trigger="click">
    <template #content>
      <div class="header-msg">
        <div class="header-msg-top">
          <p>{{ t('layout.notice.title') }}</p>
          <t-button
            v-if="unreadMsg.length > 0"
            class="clear-btn"
            variant="text"
            theme="primary"
            @click="setRead('all')"
            >{{ t('layout.notice.clear') }}</t-button
          >
        </div>
        <t-list v-if="unreadMsg.length > 0" class="narrow-scrollbar" :split="false">
          <t-list-item v-for="(item, index) in unreadMsg" :key="index">
            <div>
              <p class="msg-content">{{ item.content }}</p>
              <p class="msg-type">{{ item.type }}</p>
            </div>
            <p class="msg-time">{{ item.date }}</p>
            <template #action>
              <t-button size="small" variant="outline" @click="setRead('radio', item)">
                {{ t('layout.notice.setRead') }}
              </t-button>
            </template>
          </t-list-item>
        </t-list>

        <div v-else class="empty-list">
          <img src="https://tdesign.gtimg.com/pro-template/personal/nothing.png" alt="空" />
          <p>{{ t('layout.notice.empty') }}</p>
        </div>
        <div v-if="unreadMsg.length > 0" class="header-msg-bottom">
          <t-button class="header-msg-bottom-link" variant="text" theme="default" block @click="goDetail">{{
            t('layout.notice.viewAll')
          }}</t-button>
        </div>
      </div>
    </template>
    <t-badge :count="unreadMsg.length" :offset="[4, 4]">
      <t-button theme="default" shape="square" variant="text">
        <t-icon name="mail" />
      </t-button>
    </t-badge>
  </t-popup>
</template>
<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { useRouter } from 'vue-router';

import { t } from '@/locales';
import { useNotificationStore } from '@/store';
import type { NotificationItem } from '@/utils/types';

const router = useRouter();
const store = useNotificationStore();
const { msgData, unreadMsg } = storeToRefs(store);

const setRead = (type: string, item?: NotificationItem) => {
  const changeMsg = msgData.value;
  if (type === 'all') {
    changeMsg.forEach((e) => {
      e.status = false;
    });
  } else {
    changeMsg.forEach((e) => {
      if (e.id === item?.id) {
        e.status = false;
      }
    });
  }
  store.setMsgData(changeMsg);
};

const goDetail = () => {
  router.push('/detail/secondary');
};
</script>
<style lang="less" scoped>
.header-msg {
  margin: calc(0px - var(--td-comp-paddingTB-xs)) calc(0px - var(--td-comp-paddingLR-s));
  width: 400px;

  .empty-list {
    color: var(--td-text-color-secondary);
    font: var(--td-font-body-medium);
    padding: var(--td-comp-paddingTB-xxl) 0;
    text-align: center;

    img {
      width: var(--td-comp-size-xxxxl);
    }

    p {
      margin-top: var(--td-comp-margin-xs);
    }
  }

  &-top {
    align-items: center;
    color: var(--td-text-color-primary);
    display: flex;
    font: var(--td-font-title-medium);
    justify-content: space-between;
    padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-xl) 0;
    position: relative;
    text-align: left;

    .clear-btn {
      right: calc(var(--td-comp-paddingTB-l) - var(--td-comp-paddingLR-xl));
    }
  }

  &-bottom {
    align-items: center;
    border-top: 1px solid var(--td-component-stroke);
    display: flex;
    justify-content: center;
    padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-s);

    &-link {
      color: var(--td-text-color-placeholder);
      cursor: pointer;
      text-decoration: none;
    }
  }

  .t-list {
    height: calc(100% - 104px);
    padding: var(--td-comp-margin-s) var(--td-comp-margin-s);
  }

  .t-list-item {
    border-radius: var(--td-radius-default);
    color: var(--td-text-color-primary);
    cursor: pointer;
    font: var(--td-font-body-medium);
    overflow: hidden;
    padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
    transition: background-color 0.2s linear;
    width: 100%;

    &:hover {
      background-color: var(--td-bg-color-container-hover);

      .msg-content {
        color: var(--td-brand-color);
      }

      .t-list-item__action {
        button {
          bottom: var(--td-comp-margin-l);
          opacity: 1;
        }
      }

      .msg-time {
        bottom: -6px;
        opacity: 0;
      }
    }

    .msg-content {
      margin-bottom: var(--td-comp-margin-s);
    }

    .msg-type {
      color: var(--td-text-color-secondary);
    }

    .t-list-item__action {
      button {
        bottom: -6px;
        opacity: 0;
        position: absolute;
        right: var(--td-comp-margin-xxl);
      }
    }

    .msg-time {
      bottom: var(--td-comp-margin-l);
      color: var(--td-text-color-secondary);
      opacity: 1;
      position: absolute;
      right: var(--td-comp-margin-xxl);
      transition: all 0.2s ease;
    }
  }
}
</style>
