<template>
  <aside class="infrastructure-canvas" aria-hidden="true">
    <div class="infrastructure-canvas__halo infrastructure-canvas__halo--primary"></div>
    <div class="infrastructure-canvas__halo infrastructure-canvas__halo--secondary"></div>
    <div class="infrastructure-canvas__grid"></div>

    <svg
      class="infrastructure-canvas__topology"
      viewBox="0 0 720 560"
      preserveAspectRatio="xMidYMid meet"
      focusable="false"
    >
      <defs>
        <linearGradient id="graft-active-path" x1="0%" y1="100%" x2="100%" y2="0%">
          <stop offset="0%" stop-color="var(--infrastructure-path-neutral)" />
          <stop offset="42%" stop-color="var(--infrastructure-path-active)" />
          <stop offset="100%" stop-color="var(--infrastructure-path-active-soft)" />
        </linearGradient>
      </defs>

      <g class="topology__paths" fill="none" stroke-linecap="round" stroke-linejoin="round">
        <path
          class="topology__path topology__path--neutral"
          pathLength="1"
          d="M 108 374 C 174 358 236 324 348 326 L 405 275"
        />
        <path class="topology__path topology__path--neutral" pathLength="1" d="M 348 326 C 290 290 186 250 100 246" />
        <path class="topology__path topology__path--neutral" pathLength="1" d="M 290 314 C 276 246 298 190 350 154" />
        <path class="topology__path topology__path--neutral" pathLength="1" d="M 348 326 C 358 390 327 443 280 476" />
        <path class="topology__path topology__path--neutral" pathLength="1" d="M 326 370 C 290 392 252 418 225 440" />
        <path class="topology__path topology__path--active" pathLength="1" d="M 405 275 C 469 245 552 188 642 140" />
        <path class="topology__path topology__path--active" pathLength="1" d="M 405 275 C 438 346 506 397 570 420" />
        <path
          class="topology__path topology__path--active topology__path--branch"
          pathLength="1"
          d="M 496 224 C 494 177 516 144 555 116"
        />
      </g>

      <g class="topology__nodes">
        <g class="topology__node topology__node--standard" transform="translate(108 374)">
          <circle r="9" />
          <circle class="topology__node-core" r="2.5" />
        </g>
        <g class="topology__node topology__node--terminal" transform="translate(100 246)">
          <rect x="-7" y="-7" width="14" height="14" rx="2" transform="rotate(45)" />
          <circle class="topology__node-core" r="2.25" />
        </g>
        <g class="topology__node topology__node--terminal" transform="translate(350 154)">
          <rect x="-7" y="-7" width="14" height="14" rx="2" transform="rotate(45)" />
          <circle class="topology__node-core" r="2.25" />
        </g>
        <g class="topology__node topology__node--standard" transform="translate(348 326)">
          <circle r="9" />
          <circle class="topology__node-core" r="2.5" />
        </g>
        <g class="topology__node topology__node--terminal" transform="translate(280 476)">
          <rect x="-7" y="-7" width="14" height="14" rx="2" transform="rotate(45)" />
          <circle class="topology__node-core" r="2.25" />
        </g>
        <g class="topology__node topology__node--standard" transform="translate(225 440)">
          <circle r="9" />
          <circle class="topology__node-core" r="2.5" />
        </g>
        <g class="topology__node topology__node--terminal topology__node--active" transform="translate(642 140)">
          <rect x="-7" y="-7" width="14" height="14" rx="2" transform="rotate(45)" />
          <circle class="topology__node-core" r="2.25" />
        </g>
        <g class="topology__node topology__node--terminal topology__node--active" transform="translate(570 420)">
          <rect x="-7" y="-7" width="14" height="14" rx="2" transform="rotate(45)" />
          <circle class="topology__node-core" r="2.25" />
        </g>
        <g class="topology__node topology__node--active" transform="translate(555 116)">
          <circle r="9" />
          <circle class="topology__node-core" r="2.5" />
        </g>
      </g>

      <g class="topology__graft-point" transform="translate(405 275)">
        <circle class="topology__graft-halo" r="35" />
        <circle class="topology__graft-outer" r="16" />
        <circle class="topology__graft-inner" r="10" />
        <circle class="topology__graft-core" r="3" />
      </g>
    </svg>
  </aside>
</template>
<script setup lang="ts">
defineOptions({
  name: 'InfrastructureCanvas',
});
</script>
<style lang="less" scoped>
.infrastructure-canvas {
  --infrastructure-grid-opacity: 0.2;
  --infrastructure-halo-primary-core: color-mix(in srgb, var(--td-brand-color-6) 15%, transparent);
  --infrastructure-halo-primary-edge: color-mix(in srgb, var(--td-brand-color-6) 4%, transparent);
  --infrastructure-halo-secondary-core: color-mix(in srgb, var(--td-brand-color-5) 9%, transparent);
  --infrastructure-halo-secondary-edge: color-mix(in srgb, var(--td-brand-color-5) 2%, transparent);
  --infrastructure-node-fill: color-mix(in srgb, var(--td-bg-color-container) 78%, var(--td-brand-color-6));
  --infrastructure-node-halo: color-mix(in srgb, var(--td-brand-color-6) 9%, transparent);
  --infrastructure-path-active: var(--td-brand-color-6);
  --infrastructure-path-active-soft: color-mix(in srgb, var(--td-brand-color-5) 68%, var(--td-component-border));
  --infrastructure-path-neutral: color-mix(in srgb, var(--td-component-border) 82%, var(--td-brand-color-5));

  isolation: isolate;
  min-height: min(540px, calc(100vh - 180px));
  position: relative;
}

.infrastructure-canvas__grid {
  background-image:
    linear-gradient(to right, color-mix(in srgb, var(--td-component-stroke) 72%, transparent) 1px, transparent 1px),
    linear-gradient(to bottom, color-mix(in srgb, var(--td-component-stroke) 72%, transparent) 1px, transparent 1px);
  background-position: center;
  background-size: 48px 48px;
  inset: 0;
  mask-image: radial-gradient(ellipse 72% 76% at 54% 49%, #000 18%, transparent 78%);
  opacity: var(--infrastructure-grid-opacity);
  position: absolute;
}

.infrastructure-canvas__halo {
  aspect-ratio: 1;
  opacity: 0;
  position: absolute;
}

.infrastructure-canvas__halo--primary {
  background: radial-gradient(
    circle at 50% 50%,
    var(--infrastructure-halo-primary-core) 0%,
    var(--infrastructure-halo-primary-edge) 38%,
    transparent 72%
  );
  left: 30%;
  top: 19%;
  width: 57%;
}

.infrastructure-canvas__halo--secondary {
  background: radial-gradient(
    circle at 50% 50%,
    var(--infrastructure-halo-secondary-core) 0%,
    var(--infrastructure-halo-secondary-edge) 42%,
    transparent 72%
  );
  right: -3%;
  top: 3%;
  width: 33%;
}

.infrastructure-canvas__topology {
  display: block;
  height: auto;
  margin: 0 auto;
  max-height: 560px;
  max-width: 720px;
  overflow: visible;
  pointer-events: none;
  position: relative;
  width: 100%;
  z-index: 1;
}

.topology__path {
  stroke-dasharray: 1;
  stroke-dashoffset: 1;
  stroke-width: 1.4;
}

.topology__path--neutral {
  stroke: var(--infrastructure-path-neutral);
}

.topology__path--active {
  stroke: url('#graft-active-path');
  stroke-width: 1.7;
}

.topology__path--branch {
  stroke-width: 1.35;
}

.topology__node {
  fill: var(--infrastructure-node-fill);
  opacity: 0;
  stroke: var(--infrastructure-path-neutral);
  stroke-width: 1.2;
}

.topology__node circle:first-child,
.topology__node rect {
  filter: drop-shadow(0 0 5px var(--infrastructure-node-halo));
}

.topology__node-core {
  fill: var(--infrastructure-path-active);
  opacity: 0.82;
  stroke: none;
}

.topology__node--active {
  stroke: var(--infrastructure-path-active);
}

.topology__graft-point {
  opacity: 0;
  pointer-events: all;
}

.topology__graft-halo {
  fill: transparent;
  stroke: var(--infrastructure-node-halo);
  stroke-width: 12;
  transition: stroke 180ms ease-out;
}

.topology__graft-outer {
  fill: var(--infrastructure-node-fill);
  stroke: var(--infrastructure-path-active);
  stroke-width: 1.6;
  transition:
    fill 180ms ease-out,
    stroke 180ms ease-out,
    stroke-width 180ms ease-out;
}

.topology__graft-inner {
  fill: color-mix(in srgb, var(--td-bg-color-container) 67%, var(--td-brand-color-6));
  stroke: color-mix(in srgb, var(--td-brand-color-6) 72%, var(--td-component-border));
  stroke-width: 1;
  transition:
    fill 180ms ease-out,
    stroke 180ms ease-out;
}

.topology__graft-core {
  fill: var(--infrastructure-path-active);
  transition: fill 180ms ease-out;
}

@media (hover: hover) {
  .topology__graft-point {
    cursor: pointer;
  }

  .topology__graft-point:hover .topology__graft-halo {
    stroke: color-mix(in srgb, var(--td-brand-color-6) 28%, transparent);
  }

  .topology__graft-point:hover .topology__graft-outer {
    fill: color-mix(in srgb, var(--td-bg-color-container) 58%, var(--td-brand-color-6));
    stroke: var(--td-brand-color-6);
    stroke-width: 2;
  }

  .topology__graft-point:hover .topology__graft-inner {
    fill: color-mix(in srgb, var(--td-bg-color-container) 48%, var(--td-brand-color-6));
    stroke: var(--td-brand-color-5);
  }

  .topology__graft-point:hover .topology__graft-core {
    fill: var(--td-brand-color-5);
  }
}

@media (prefers-reduced-motion: no-preference) {
  .infrastructure-canvas__halo {
    animation: topology-halo-in 560ms ease-out both;
  }

  .infrastructure-canvas__halo--secondary {
    animation-delay: 90ms;
  }

  .topology__path {
    animation: topology-path-reveal 560ms cubic-bezier(0.22, 1, 0.36, 1) both;
  }

  .topology__path:nth-child(2),
  .topology__path:nth-child(5) {
    animation-delay: 70ms;
  }

  .topology__path:nth-child(3),
  .topology__path:nth-child(4) {
    animation-delay: 120ms;
  }

  .topology__path:nth-child(6),
  .topology__path:nth-child(7) {
    animation-delay: 180ms;
  }

  .topology__path:nth-child(8) {
    animation-delay: 240ms;
  }

  .topology__node,
  .topology__graft-point {
    animation: topology-node-in 260ms ease-out both;
    animation-delay: 300ms;
  }

  .topology__graft-point {
    animation-delay: 390ms;
  }
}

@keyframes topology-halo-in {
  from {
    opacity: 0;
  }

  to {
    opacity: 1;
  }
}

@keyframes topology-path-reveal {
  to {
    stroke-dashoffset: 0;
  }
}

@keyframes topology-node-in {
  from {
    opacity: 0;
  }

  to {
    opacity: 1;
  }
}

[theme-mode='dark'] .infrastructure-canvas {
  --infrastructure-grid-opacity: 0.28;
  --infrastructure-halo-primary-core: color-mix(in srgb, var(--td-brand-color-6) 22%, transparent);
  --infrastructure-halo-primary-edge: color-mix(in srgb, var(--td-brand-color-6) 8%, transparent);
  --infrastructure-halo-secondary-core: color-mix(in srgb, var(--td-brand-color-5) 13%, transparent);
  --infrastructure-halo-secondary-edge: color-mix(in srgb, var(--td-brand-color-5) 5%, transparent);
  --infrastructure-node-fill: color-mix(in srgb, var(--td-bg-color-container) 62%, var(--td-brand-color-6));
  --infrastructure-node-halo: color-mix(in srgb, var(--td-brand-color-6) 16%, transparent);
  --infrastructure-path-neutral: color-mix(in srgb, var(--td-component-border) 62%, var(--td-brand-color-5));
}

@media (width <= 1023px) {
  .infrastructure-canvas {
    min-height: 300px;
  }

  .infrastructure-canvas__topology {
    max-width: 560px;
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
    animation: none !important;
    transition: none !important;
  }

  .infrastructure-canvas__halo,
  .topology__node,
  .topology__graft-point {
    opacity: 1;
  }

  .topology__path {
    stroke-dashoffset: 0;
  }
}
</style>
