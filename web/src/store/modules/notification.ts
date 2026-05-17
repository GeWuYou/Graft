import { defineStore } from 'pinia';

import { t } from '@/locales';
import type { NotificationItem } from '@/utils/types';

const getMsgData = () => [
  {
    id: '123',
    content: t('notification.msg1'),
    type: t('notification.contract'),
    status: true,
    collected: false,
    date: '2021-01-01 08:00',
    quality: 'high',
  },
  {
    id: '124',
    content: t('notification.msg2'),
    type: t('notification.invoice'),
    status: true,
    collected: false,
    date: '2021-01-01 08:00',
    quality: 'low',
  },
  {
    id: '125',
    content: t('notification.msg3'),
    type: t('notification.meeting'),
    status: true,
    collected: false,
    date: '2021-01-01 08:00',
    quality: 'middle',
  },
  {
    id: '126',
    content: t('notification.msg4'),
    type: t('notification.invoice'),
    status: true,
    collected: false,
    date: '2021-01-01 08:00',
    quality: 'low',
  },
];

type MsgDataType = ReturnType<typeof getMsgData>;

export const useNotificationStore = defineStore('notification', {
  state: () => ({
    msgData: getMsgData(),
  }),
  getters: {
    unreadMsg: (state) => state.msgData.filter((item: NotificationItem) => item.status),
    readMsg: (state) => state.msgData.filter((item: NotificationItem) => !item.status),
  },
  actions: {
    setMsgData(data: MsgDataType) {
      this.msgData = data;
    },
    refreshMsgData() {
      this.msgData = getMsgData();
    },
  },
  persist: false,
});
