<template>
  <section>
    <strong>{{ presented.title }}</strong>
    <span>{{ presented.message }}</span>
    <span>{{ presented.resourceTypeLabel }}</span>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { presentNotification } from '../shared/presentation';

const { t } = useI18n();
const props = defineProps<{ item: { message: string; resource_type: string; title: string } }>();
const presented = computed(() => presentNotification(props.item, t));
</script>
