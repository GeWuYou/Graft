// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { Ref } from 'vue';
import { onUnmounted, ref } from 'vue';

export const useCounter = (duration = 60): [Ref<number>, () => void] => {
  let intervalTimer: ReturnType<typeof setInterval> | undefined;
  const countDown = ref(0);

  onUnmounted(() => {
    if (intervalTimer) {
      clearInterval(intervalTimer);
    }
  });

  return [
    countDown,
    () => {
      countDown.value = duration;

      if (intervalTimer) {
        clearInterval(intervalTimer);
      }

      intervalTimer = setInterval(() => {
        if (countDown.value > 0) {
          countDown.value -= 1;
          return;
        }

        clearInterval(intervalTimer);
        intervalTimer = undefined;
        countDown.value = 0;
      }, 1000);
    },
  ];
};
