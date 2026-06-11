<template>
  <section>
    <strong>{{ presentNotification(item, t).title }}</strong>
    <span>{{ presentNotification(item, t).message }}</span>
    <span>{{ presentNotification(item, t).resourceTypeLabel }}</span>
  </section>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import { presentNotification } from '../shared/presentation';

const { t } = useI18n();
defineProps<{ item: { message: string; resource_type: string; title: string } }>();
</script>
