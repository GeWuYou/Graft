<template>
  <aside class="infrastructure-canvas" aria-hidden="true">
    <div class="infrastructure-canvas__halo infrastructure-canvas__halo--primary"></div>
    <div class="infrastructure-canvas__halo infrastructure-canvas__halo--secondary"></div>
    <div class="infrastructure-canvas__grid"></div>
    <div class="topology topology--primary">
      <span class="topology__line topology__line--horizontal"></span>
      <span class="topology__line topology__line--vertical"></span>
      <span class="topology__line topology__line--diagonal"></span>
      <span class="topology__node topology__node--gateway"></span>
      <span class="topology__node topology__node--service-a"></span>
      <span class="topology__node topology__node--service-b"></span>
      <span class="topology__node topology__node--runtime"></span>
      <span class="topology__signal topology__signal--a"></span>
      <span class="topology__signal topology__signal--b"></span>
    </div>
    <div class="topology topology--secondary">
      <span class="topology__line topology__line--horizontal"></span>
      <span class="topology__node topology__node--gateway"></span>
      <span class="topology__node topology__node--service-a"></span>
      <span class="topology__node topology__node--service-b"></span>
    </div>
  </aside>
</template>
<script setup lang="ts">
defineOptions({
  name: 'InfrastructureCanvas',
});
</script>
<style lang="less" scoped>
.infrastructure-canvas {
  isolation: isolate;
  min-height: 540px;
  overflow: hidden;
  pointer-events: none;
  position: relative;
}

.infrastructure-canvas__grid {
  background-image:
    linear-gradient(to right, color-mix(in srgb, var(--td-component-stroke) 72%, transparent) 1px, transparent 1px),
    linear-gradient(to bottom, color-mix(in srgb, var(--td-component-stroke) 72%, transparent) 1px, transparent 1px);
  background-position: center;
  background-size: 48px 48px;
  inset: 0;
  mask-image: radial-gradient(ellipse 70% 72% at 58% 48%, #000 18%, transparent 76%);
  opacity: 0.4;
  position: absolute;
}

.infrastructure-canvas__halo {
  background: var(--td-brand-color);
  border-radius: 50%;
  filter: blur(116px);
  mix-blend-mode: normal;
  position: absolute;
}

.infrastructure-canvas__halo--primary {
  height: 360px;
  opacity: 0.12;
  right: 2%;
  top: 12%;
  width: 360px;
}

.infrastructure-canvas__halo--secondary {
  bottom: -24%;
  height: 280px;
  opacity: 0.06;
  right: 20%;
  width: 280px;
}

.topology {
  background: color-mix(in srgb, var(--td-bg-color-container) 34%, transparent);
  border: 1px solid color-mix(in srgb, var(--td-brand-color) 20%, var(--td-component-border));
  border-radius: 10px;
  box-shadow: 0 18px 42px color-mix(in srgb, var(--td-brand-color) 8%, transparent);
  position: absolute;
}

.topology--primary {
  height: 270px;
  right: 12%;
  top: calc(50% - 135px);
  transform: rotate(-4deg);
  width: min(420px, 78%);
}

.topology--secondary {
  height: 116px;
  right: 0;
  top: calc(50% + 136px);
  transform: rotate(3deg);
  width: min(250px, 48%);
}

.topology__line {
  background: color-mix(in srgb, var(--td-brand-color) 44%, var(--td-component-border));
  height: 1px;
  opacity: 0.76;
  position: absolute;
  transform-origin: left center;
}

.topology__line--horizontal {
  left: 19%;
  top: 51%;
  width: 60%;
}

.topology__line--vertical {
  left: 51%;
  top: 24%;
  transform: rotate(90deg);
  width: 54%;
}

.topology__line--diagonal {
  left: 23%;
  top: 31%;
  transform: rotate(30deg);
  width: 56%;
}

.topology__node {
  background: color-mix(in srgb, var(--td-bg-color-container) 86%, var(--td-brand-color));
  border: 1px solid color-mix(in srgb, var(--td-brand-color) 68%, var(--td-component-border));
  border-radius: 6px;
  box-shadow: 0 0 0 5px color-mix(in srgb, var(--td-brand-color) 7%, transparent);
  height: 26px;
  position: absolute;
  width: 26px;
}

.topology__node::after {
  background: var(--td-brand-color);
  border-radius: 50%;
  content: '';
  height: 5px;
  left: 9px;
  opacity: 0.8;
  position: absolute;
  top: 9px;
  width: 5px;
}

.topology__node--gateway {
  left: 15%;
  top: calc(50% - 13px);
}

.topology__node--service-a {
  left: calc(50% - 13px);
  top: 19%;
}

.topology__node--service-b {
  bottom: 18%;
  left: calc(50% - 13px);
}

.topology__node--runtime {
  right: 14%;
  top: calc(50% - 13px);
}

.topology__signal {
  background: var(--td-brand-color);
  border-radius: 50%;
  height: 6px;
  opacity: 0.65;
  position: absolute;
  width: 6px;
}

.topology__signal--a {
  left: 29%;
  top: 47%;
}

.topology__signal--b {
  right: 28%;
  top: 47%;
}

.topology--secondary .topology__line--horizontal {
  left: 18%;
  top: 50%;
  width: 64%;
}

.topology--secondary .topology__node {
  height: 18px;
  width: 18px;
}

.topology--secondary .topology__node::after {
  height: 4px;
  left: 6px;
  top: 6px;
  width: 4px;
}

.topology--secondary .topology__node--gateway {
  left: 11%;
  top: calc(50% - 9px);
}

.topology--secondary .topology__node--service-a {
  left: calc(50% - 9px);
  top: calc(50% - 9px);
}

.topology--secondary .topology__node--service-b {
  inset: calc(50% - 9px) 11% auto auto;
}

@media (width <= 1023px) {
  .infrastructure-canvas {
    min-height: 300px;
  }

  .topology--primary {
    right: 6%;
  }
}

@media (width <= 767px) {
  .infrastructure-canvas {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .infrastructure-canvas,
  .infrastructure-canvas * {
    transition: none !important;
  }
}
</style>
