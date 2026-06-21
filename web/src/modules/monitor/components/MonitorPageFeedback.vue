<template>
  <section-card v-if="message" :title="title" :description="message" />
</template>
<script setup lang="ts">
import SectionCard from './SectionCard.vue';

defineProps<{
  title: string;
  message?: string;
}>();
</script>
