<template>
  <icon :icon="resolveMenuIcon(iconKey)" aria-hidden="true" />
</template>
<script setup lang="ts">
import { Icon } from '@iconify/vue';

import { resolveMenuIcon } from './menu-icon';

defineProps<{ iconKey?: string }>();
</script>
