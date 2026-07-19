<template>
  <t-drawer
    :visible="visible"
    :header="t('container.images.tagManager.title')"
    size="680px"
    placement="right"
    @update:visible="emit('update:visible', $event)"
  >
    <t-loading :loading="loading">
      <template v-if="image">
        <div class="tag-manager-summary">
          <div>
            <span>{{ t('container.images.tagManager.tagCount', { count: tags.length }) }}</span>
            <strong>{{ tags.length }}</strong>
          </div>
          <div>
            <span>{{ t('container.images.tagManager.containerCount') }}</span>
            <strong>{{ image.container_references?.length ?? 0 }}</strong>
          </div>
        </div>

        <t-descriptions bordered :column="1" size="small">
          <t-descriptions-item :label="t('container.images.fields.id')">
            <div class="tag-manager-copyable">
              <span class="tag-manager-copyable__value">{{ image.id }}</span>
              <t-button size="small" variant="text" @click="copy(image.id)">{{
                t('container.images.tagManager.copy')
              }}</t-button>
            </div>
          </t-descriptions-item>
          <t-descriptions-item :label="t('container.images.fields.digests')">
            <t-space v-if="image.repository_digests.length" direction="vertical" size="small">
              <div v-for="digest in image.repository_digests" :key="digest" class="tag-manager-copyable">
                <span class="tag-manager-copyable__value">{{ digest }}</span>
                <t-button size="small" variant="text" @click="copy(digest)">{{
                  t('container.images.tagManager.copy')
                }}</t-button>
              </div>
            </t-space>
            <span v-else>-</span>
          </t-descriptions-item>
          <t-descriptions-item :label="t('container.images.fields.containers')">
            <t-space v-if="image.container_references?.length" size="small" break-line>
              <t-tooltip v-for="container in image.container_references" :key="container.id" :content="container.id">
                <t-tag size="small" variant="light-outline">{{ container.name }}</t-tag>
              </t-tooltip>
            </t-space>
            <span v-else>{{ t('container.images.unused') }}</span>
          </t-descriptions-item>
        </t-descriptions>

        <section class="tag-manager-tags">
          <div class="tag-manager-tags__heading">
            <h3>{{ t('container.images.fields.tags') }}</h3>
            <span>{{ t('container.images.tagManager.tagCount', { count: tags.length }) }}</span>
          </div>
          <t-empty v-if="!tags.length" :title="t('container.images.untagged')" />
          <t-list v-else split>
            <t-list-item v-for="reference in tags" :key="reference">
              <t-space direction="vertical" size="small">
                <div class="tag-manager-copyable">
                  <span class="tag-manager-copyable__value">{{ reference }}</span>
                  <t-button size="small" variant="text" @click="copy(reference)">{{
                    t('container.images.tagManager.copy')
                  }}</t-button>
                </div>
                <div class="tag-manager-copyable">
                  <span class="tag-manager-copyable__label">{{ t('container.images.tagManager.pullCommand') }}</span>
                  <span class="tag-manager-copyable__value">{{ `docker pull ${reference}` }}</span>
                  <t-button size="small" variant="text" @click="copy(`docker pull ${reference}`)">{{
                    t('container.images.tagManager.copy')
                  }}</t-button>
                </div>
              </t-space>
              <template #action>
                <t-button theme="danger" variant="text" @click="openRemoveTag(reference)">
                  {{ t('container.images.actions.untag') }}
                </t-button>
              </template>
            </t-list-item>
          </t-list>
        </section>
      </template>
      <t-empty v-else-if="!loading" :title="t('container.images.detail.loadFailed')" />
    </t-loading>

    <t-dialog
      v-model:visible="removeDialogVisible"
      theme="danger"
      :header="t('container.images.untag.title')"
      :confirm-btn="t('container.images.actions.untag')"
      :confirm-loading="removing"
      @confirm="submitRemoveTag"
    >
      <p>{{ t('container.images.untag.confirm', { reference: selectedReference }) }}</p>
      <t-alert v-if="tags.length === 1" theme="warning" :message="t('container.images.untag.lastTagWarning')" />
    </t-dialog>
  </t-drawer>
</template>
<script setup lang="ts">
// 标签抽屉统一承担 Image 的命名引用管理；删除镜像仍由页面按 Image ID 独立处理。
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { copyText } from '@/shared/observability';

import { type DockerImageRecord, getDockerImage } from '../api/container';
import { untagDockerImage } from '../api/image-actions';

const props = defineProps<{
  visible: boolean;
  imageId: string | null;
}>();
const emit = defineEmits<{
  'update:visible': [visible: boolean];
  refreshed: [image: DockerImageRecord | null];
}>();
const { t } = useI18n();
const image = ref<DockerImageRecord | null>(null);
const loading = ref(false);
const removing = ref(false);
const removeDialogVisible = ref(false);
const selectedReference = ref('');
const tags = computed(() => image.value?.repository_tags.filter(Boolean) ?? []);

async function loadImage() {
  if (!props.imageId) return;
  loading.value = true;
  try {
    image.value = await getDockerImage(props.imageId);
  } catch {
    image.value = null;
    MessagePlugin.error(t('container.images.detail.loadFailed'));
  } finally {
    loading.value = false;
  }
}

function openRemoveTag(reference: string) {
  selectedReference.value = reference;
  removeDialogVisible.value = true;
}

async function submitRemoveTag() {
  if (!image.value || !selectedReference.value) return;
  removing.value = true;
  try {
    await untagDockerImage(image.value.id, { reference: selectedReference.value });
    removeDialogVisible.value = false;
    MessagePlugin.success(t('container.images.untag.success'));
    await loadImage();
    emit('refreshed', image.value);
  } catch {
    MessagePlugin.error(t('container.images.untag.failed'));
  } finally {
    removing.value = false;
  }
}

async function copy(value: string) {
  if (await copyText(value)) MessagePlugin.success(t('container.images.tagManager.copySuccess'));
}

watch(
  () => [props.visible, props.imageId] as const,
  ([visible]) => {
    if (visible) void loadImage();
  },
  { immediate: true },
);
</script>
<style scoped lang="less">
.tag-manager-summary {
  display: grid;
  gap: var(--td-comp-margin-l);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: var(--td-comp-margin-l);
}

.tag-manager-summary > div {
  background: var(--td-bg-color-container-hover);
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.tag-manager-summary span,
.tag-manager-tags__heading span,
.tag-manager-copyable__label {
  color: var(--td-text-color-secondary);
  display: block;
  font-size: var(--td-font-size-s);
}

.tag-manager-summary strong {
  display: block;
  font-size: var(--td-font-size-title-large);
  margin-top: var(--td-comp-margin-s);
}

.tag-manager-tags {
  margin-top: var(--td-comp-margin-xl);
}

.tag-manager-tags__heading {
  align-items: baseline;
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--td-comp-margin-s);
}

.tag-manager-tags__heading h3 {
  font-size: var(--td-font-size-title-medium);
  margin: 0;
}

.tag-manager-copyable {
  align-items: baseline;
  display: flex;
  gap: var(--td-comp-margin-s);
  max-width: 560px;
}

.tag-manager-copyable__value {
  font-family: var(--td-font-family-mono);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
