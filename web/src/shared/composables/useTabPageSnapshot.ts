import { watch } from 'vue';

import { useTabsRouterStore } from '@/store';
import type { TabPageSnapshot } from '@/utils/types';

type UseTabPageSnapshotOptions<TSnapshot extends TabPageSnapshot> = {
  apply: (snapshot: TSnapshot) => void;
  read: () => TSnapshot;
};

export function useTabPageSnapshot<TSnapshot extends TabPageSnapshot>({
  apply,
  read,
}: UseTabPageSnapshotOptions<TSnapshot>) {
  const tabsRouterStore = useTabsRouterStore();
  const tabKey = tabsRouterStore.activeTabKey;
  const restoredSnapshot = tabsRouterStore.getPageSnapshot<TSnapshot>(tabKey);

  if (restoredSnapshot) {
    apply(restoredSnapshot);
  }

  watch(
    read,
    (snapshot) => {
      tabsRouterStore.setPageSnapshot(tabKey, snapshot);
    },
    {
      deep: true,
      flush: 'post',
      immediate: true,
    },
  );
}
