<template>
  <div v-if="references.length" class="container-reference-list">
    <t-tooltip
      v-for="reference in references.slice(0, 2)"
      :key="reference.id"
      :content="referenceTooltip(reference)"
      placement="top"
    >
      <t-tag
        class="container-reference-list__badge"
        role="link"
        size="small"
        tabindex="0"
        variant="light-outline"
        @click="emit('open', reference.id)"
        @keydown.enter.prevent="emit('open', reference.id)"
        @keydown.space.prevent="emit('open', reference.id)"
      >
        {{ reference.name || reference.id }}
      </t-tag>
    </t-tooltip>
    <t-popup
      v-if="references.length > 2"
      :delay="[120, 160]"
      :visible="overflowVisible"
      placement="top-left"
      trigger="hover"
      @visible-change="overflowVisible = $event"
    >
      <button
        class="container-reference-list__overflow-trigger"
        type="button"
        :aria-expanded="overflowVisible"
        aria-haspopup="dialog"
        :aria-label="title"
        @focus="overflowVisible = true"
        @keydown.esc.prevent="overflowVisible = false"
      >
        <t-tag class="container-reference-list__badge" size="small" variant="light-outline">
          +{{ references.length - 2 }}
        </t-tag>
      </button>
      <template #content>
        <div class="container-reference-list__overflow">
          <span class="container-reference-list__overflow-title">{{ title }}</span>
          <div class="container-reference-list">
            <t-tooltip
              v-for="reference in references.slice(2)"
              :key="reference.id"
              :content="referenceTooltip(reference)"
              placement="top"
            >
              <t-tag
                class="container-reference-list__badge"
                role="link"
                size="small"
                tabindex="0"
                variant="light-outline"
                @click="emit('open', reference.id)"
                @keydown.enter.prevent="emit('open', reference.id)"
                @keydown.space.prevent="emit('open', reference.id)"
              >
                {{ reference.name || reference.id }}
              </t-tag>
            </t-tooltip>
          </div>
        </div>
      </template>
    </t-popup>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue';

type ContainerReference = { id: string; name?: string | null };

defineProps<{
  references: ContainerReference[];
  title: string;
}>();

const emit = defineEmits<{ open: [id: string] }>();
const overflowVisible = ref(false);

function referenceTooltip(reference: ContainerReference) {
  return reference.name ? `${reference.name} (${reference.id})` : reference.id;
}
</script>
<style scoped lang="less">
.container-reference-list {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--td-comp-margin-xs);
}

.container-reference-list__badge {
  cursor: pointer;
}

.container-reference-list__badge:focus-visible,
.container-reference-list__overflow-trigger:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.container-reference-list__overflow-trigger {
  background: transparent;
  border: 0;
  cursor: pointer;
  display: inline-flex;
  padding: 0;
}

.container-reference-list__overflow {
  display: grid;
  gap: var(--td-comp-margin-s);
  max-height: min(40vh, 240px);
  max-width: 320px;
  overflow: auto;
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-s);
}

.container-reference-list__overflow-title {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
}
</style>
