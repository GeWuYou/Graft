import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next';
import { h, ref } from 'vue';

import VolumeRemovalConfirmContent from '../components/VolumeRemovalConfirmContent.vue';

type Translate = (key: string, named?: Record<string, unknown>) => string;

export type VolumeRemovalCandidate = {
  containerNames: string[];
  name: string;
  sizeBytes?: number | null;
};

type VolumeRemovalConfirmationOptions = {
  candidates: VolumeRemovalCandidate[];
  confirmLabel: string;
  confirmationName?: string;
  forceRequired?: boolean;
  header: string;
  onConfirm: (force: boolean) => Promise<boolean | void>;
  t: Translate;
};

/** openVolumeRemovalConfirmation 保证所有数据卷删除入口先展示同一组资产事实与风险。 */
export function openVolumeRemovalConfirmation(options: VolumeRemovalConfirmationOptions) {
  const typedName = ref('');
  const force = ref(false);
  const dialog = DialogPlugin.confirm({
    header: options.header,
    theme: 'danger',
    confirmBtn: options.confirmLabel,
    cancelBtn: options.t('container.volume.actions.cancel'),
    body: () =>
      h(VolumeRemovalConfirmContent, {
        candidates: options.candidates,
        confirmationName: options.confirmationName,
        force: force.value,
        forceRequired: Boolean(options.forceRequired),
        t: options.t,
        typedName: typedName.value,
        'onUpdate:force': (value: boolean) => (force.value = value),
        'onUpdate:typedName': (value: string) => (typedName.value = value),
      }),
    onConfirm: async () => {
      if (options.confirmationName && typedName.value !== options.confirmationName) {
        MessagePlugin.warning(options.t('container.volume.actions.nameRequired'));
        return;
      }
      if (options.forceRequired && !force.value) {
        MessagePlugin.warning(options.t('container.volume.actions.forceRequired'));
        return;
      }
      const shouldClose = await options.onConfirm(force.value);
      if (shouldClose !== false) dialog.hide();
    },
  });
  return dialog;
}
