<template>
  <div class="management-page-content">
    <slot />
  </div>
</template>
<style scoped lang="less">
.management-page-content {
  --graft-page-max-width: 1600px;
  --graft-page-width-ratio: 88vw;
  --graft-page-min-padding: 24px;

  display: flex;
  flex-direction: column;
  gap: 16px;
  margin: 0 auto;
  padding-bottom: max(calc(var(--td-comp-margin-xxl) + 8px), calc(var(--graft-floating-toolbar-safe-area) + 8px));
  width: min(var(--graft-page-width-ratio), var(--graft-page-max-width));
}

@media (width <= 1280px) {
  .management-page-content {
    width: calc(100% - 32px);
  }
}

@media (width <= 768px) {
  .management-page-content {
    gap: 12px;
    padding-bottom: var(--td-comp-margin-xl);
    width: calc(100% - var(--graft-page-min-padding));
  }
}
</style>
