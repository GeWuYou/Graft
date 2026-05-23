<template>
  <div class="management-page-content">
    <slot />
  </div>
</template>
<style scoped lang="less">
.management-page-content {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
  padding-bottom: max(calc(var(--td-comp-margin-xxl) + 8px), calc(var(--graft-floating-toolbar-safe-area) + 8px));
  width: 100%;
}

@media (width <= 768px) {
  .management-page-content {
    gap: 12px;
    padding-bottom: var(--td-comp-margin-xl);
  }
}
</style>
