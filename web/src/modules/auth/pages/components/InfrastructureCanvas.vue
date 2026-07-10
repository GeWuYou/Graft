<template>
  <aside class="infrastructure-canvas" aria-hidden="true">
    <div class="infrastructure-canvas__halo infrastructure-canvas__halo--primary"></div>
    <div class="infrastructure-canvas__halo infrastructure-canvas__halo--secondary"></div>

    <svg
      class="infrastructure-canvas__topology"
      viewBox="40 70 640 420"
      preserveAspectRatio="xMidYMid meet"
      focusable="false"
    >
      <defs>
        <linearGradient id="graft-active-path" x1="0%" y1="100%" x2="100%" y2="0%">
          <stop offset="0%" stop-color="var(--infrastructure-path-neutral)" />
          <stop offset="42%" stop-color="var(--infrastructure-path-active)" />
          <stop offset="100%" stop-color="var(--infrastructure-path-active-soft)" />
        </linearGradient>
        <radialGradient id="graft-node-acrylic" cx="30%" cy="20%" r="78%">
          <stop offset="0%" stop-color="var(--infrastructure-node-highlight)" />
          <stop offset="36%" stop-color="var(--infrastructure-node-fill)" />
          <stop offset="100%" stop-color="var(--infrastructure-node-depth)" />
        </radialGradient>
      </defs>

      <g class="topology__paths" fill="none" stroke-linecap="round" stroke-linejoin="round">
        <path
          class="topology__path topology__path--neutral"
          :class="{ 'topology__path--highlighted': isPathActive('left-lower') }"
          pathLength="1"
          d="M 104 392 C 164 374 228 344 294 315"
        />
        <path
          class="topology__path topology__path--neutral"
          :class="{ 'topology__path--highlighted': isPathActive('left-upper') }"
          pathLength="1"
          d="M 294 315 C 246 301 191 270 142 244"
        />
        <path
          class="topology__path topology__path--neutral"
          :class="{ 'topology__path--highlighted': isPathActive('left-top') }"
          pathLength="1"
          d="M 294 315 C 267 259 266 202 286 150"
        />
        <path
          class="topology__path topology__path--neutral"
          :class="{ 'topology__path--highlighted': isPathActive('left-bottom') }"
          pathLength="1"
          d="M 294 315 C 307 359 280 411 232 450"
        />
        <path
          class="topology__path topology__path--graft"
          :class="{ 'topology__path--highlighted': isPathActive('graft') }"
          pathLength="1"
          d="M 294 315 C 339 313 393 328 452 340"
        />
        <path
          class="topology__path topology__path--active"
          :class="{ 'topology__path--highlighted': isPathActive('right-upper') }"
          pathLength="1"
          d="M 452 340 C 509 323 573 283 640 238"
        />
        <path
          class="topology__path topology__path--active"
          :class="{ 'topology__path--highlighted': isPathActive('right-bottom') }"
          pathLength="1"
          d="M 452 340 C 497 369 547 414 594 438"
        />
        <path
          class="topology__path topology__path--active topology__path--branch"
          :class="{ 'topology__path--highlighted': isPathActive('right-top') }"
          pathLength="1"
          d="M 520 313 C 514 250 532 193 574 150"
        />
      </g>

      <g class="topology__nodes">
        <g
          class="topology__node topology__node--standard"
          :class="{ 'topology__node--highlighted': activeNode === 'left-lower' }"
          transform="translate(104 392)"
          @mouseenter="activateNode('left-lower')"
          @mouseleave="clearActiveNode"
        >
          <g class="topology__node-material">
            <circle r="10" />
            <circle class="topology__node-sheen" cx="-3" cy="-3.5" r="1.4" />
            <circle class="topology__node-core" r="2.5" />
          </g>
        </g>
        <g
          class="topology__node topology__node--terminal"
          :class="{ 'topology__node--highlighted': activeNode === 'left-upper' }"
          transform="translate(142 244)"
          @mouseenter="activateNode('left-upper')"
          @mouseleave="clearActiveNode"
        >
          <g class="topology__node-material">
            <rect x="-8" y="-8" width="16" height="16" rx="2" transform="rotate(45)" />
            <circle class="topology__node-sheen" cx="-2.5" cy="-3" r="1.25" />
            <circle class="topology__node-core" r="2.25" />
          </g>
        </g>
        <g
          class="topology__node topology__node--terminal"
          :class="{ 'topology__node--highlighted': activeNode === 'left-top' }"
          transform="translate(286 150)"
          @mouseenter="activateNode('left-top')"
          @mouseleave="clearActiveNode"
        >
          <g class="topology__node-material">
            <rect x="-8" y="-8" width="16" height="16" rx="2" transform="rotate(45)" />
            <circle class="topology__node-sheen" cx="-2.5" cy="-3" r="1.25" />
            <circle class="topology__node-core" r="2.25" />
          </g>
        </g>
        <g
          class="topology__node topology__node--junction"
          :class="{ 'topology__node--highlighted': activeNode === 'left-junction' }"
          transform="translate(294 315)"
          @mouseenter="activateNode('left-junction')"
          @mouseleave="clearActiveNode"
        >
          <g class="topology__node-material">
            <circle r="12" />
            <circle class="topology__node-ring" r="7" />
            <circle class="topology__node-sheen" cx="-3.8" cy="-4.2" r="1.5" />
            <circle class="topology__node-core" r="2.75" />
          </g>
        </g>
        <g
          class="topology__node topology__node--terminal"
          :class="{ 'topology__node--highlighted': activeNode === 'left-bottom' }"
          transform="translate(232 450)"
          @mouseenter="activateNode('left-bottom')"
          @mouseleave="clearActiveNode"
        >
          <g class="topology__node-material">
            <rect x="-8" y="-8" width="16" height="16" rx="2" transform="rotate(45)" />
            <circle class="topology__node-sheen" cx="-2.5" cy="-3" r="1.25" />
            <circle class="topology__node-core" r="2.25" />
          </g>
        </g>
        <g
          class="topology__node topology__node--junction topology__node--active"
          :class="{ 'topology__node--highlighted': activeNode === 'right-junction' }"
          transform="translate(452 340)"
          @mouseenter="activateNode('right-junction')"
          @mouseleave="clearActiveNode"
        >
          <g class="topology__node-material">
            <circle r="12" />
            <circle class="topology__node-ring" r="7" />
            <circle class="topology__node-sheen" cx="-3.8" cy="-4.2" r="1.5" />
            <circle class="topology__node-core" r="2.75" />
          </g>
        </g>
        <g
          class="topology__node topology__node--terminal topology__node--active"
          :class="{ 'topology__node--highlighted': activeNode === 'right-upper' }"
          transform="translate(640 238)"
          @mouseenter="activateNode('right-upper')"
          @mouseleave="clearActiveNode"
        >
          <g class="topology__node-material">
            <rect x="-8" y="-8" width="16" height="16" rx="2" transform="rotate(45)" />
            <circle class="topology__node-sheen" cx="-2.5" cy="-3" r="1.25" />
            <circle class="topology__node-core" r="2.25" />
          </g>
        </g>
        <g
          class="topology__node topology__node--terminal topology__node--active"
          :class="{ 'topology__node--highlighted': activeNode === 'right-bottom' }"
          transform="translate(594 438)"
          @mouseenter="activateNode('right-bottom')"
          @mouseleave="clearActiveNode"
        >
          <g class="topology__node-material">
            <rect x="-8" y="-8" width="16" height="16" rx="2" transform="rotate(45)" />
            <circle class="topology__node-sheen" cx="-2.5" cy="-3" r="1.25" />
            <circle class="topology__node-core" r="2.25" />
          </g>
        </g>
        <g
          class="topology__node topology__node--standard topology__node--active"
          :class="{ 'topology__node--highlighted': activeNode === 'right-top' }"
          transform="translate(574 150)"
          @mouseenter="activateNode('right-top')"
          @mouseleave="clearActiveNode"
        >
          <g class="topology__node-material">
            <circle r="10" />
            <circle class="topology__node-sheen" cx="-3" cy="-3.5" r="1.4" />
            <circle class="topology__node-core" r="2.5" />
          </g>
        </g>
      </g>

      <g
        class="topology__graft-connection"
        :class="{ 'topology__graft-connection--highlighted': activeNode === 'graft' }"
        transform="rotate(13 385 329)"
        @mouseenter="activateNode('graft')"
        @mouseleave="clearActiveNode"
      >
        <path class="topology__graft-sleeve" d="M 352 329 H 418" />
        <circle class="topology__graft-anchor" cx="363" cy="329" r="6" />
        <circle class="topology__graft-anchor" cx="385" cy="329" r="5.25" />
        <circle class="topology__graft-anchor" cx="407" cy="329" r="6" />
        <circle class="topology__graft-core" cx="363" cy="329" r="1.4" />
        <circle class="topology__graft-core" cx="385" cy="329" r="1.25" />
        <circle class="topology__graft-core" cx="407" cy="329" r="1.4" />
        <path class="topology__graft-seam" d="M 371 329 H 399" />
      </g>
    </svg>
  </aside>
</template>
<script setup lang="ts">
import { ref } from 'vue';

defineOptions({
  name: 'InfrastructureCanvas',
});

const activeNode = ref<string | null>(null);

const NODE_PATHS: Record<string, string[]> = {
  graft: ['graft'],
  'left-bottom': ['left-bottom'],
  'left-junction': ['left-lower', 'left-upper', 'left-top', 'left-bottom', 'graft'],
  'left-lower': ['left-lower'],
  'left-top': ['left-top'],
  'left-upper': ['left-upper'],
  'right-bottom': ['right-bottom'],
  'right-junction': ['graft', 'right-upper', 'right-bottom', 'right-top'],
  'right-top': ['right-top', 'right-upper'],
  'right-upper': ['right-upper'],
};

const activateNode = (node: string) => {
  activeNode.value = node;
};

const clearActiveNode = () => {
  activeNode.value = null;
};

const isPathActive = (path: string) => activeNode.value !== null && NODE_PATHS[activeNode.value].includes(path);
</script>
<style lang="less" scoped>
.infrastructure-canvas {
  --infrastructure-halo-primary-core: color-mix(in srgb, var(--td-brand-color-6) 18%, transparent);
  --infrastructure-halo-primary-edge: color-mix(in srgb, var(--td-brand-color-6) 5%, transparent);
  --infrastructure-halo-secondary-core: color-mix(in srgb, var(--td-brand-color-5) 10%, transparent);
  --infrastructure-halo-secondary-edge: color-mix(in srgb, var(--td-brand-color-5) 3%, transparent);
  --infrastructure-node-depth: color-mix(in srgb, var(--td-bg-color-container) 74%, var(--td-brand-color-6));
  --infrastructure-node-fill: color-mix(in srgb, var(--td-bg-color-container) 58%, var(--td-brand-color-6));
  --infrastructure-node-highlight: color-mix(in srgb, var(--td-bg-color-container) 82%, var(--td-brand-color-1));
  --infrastructure-node-halo: color-mix(in srgb, var(--td-brand-color-6) 16%, transparent);
  --infrastructure-node-rim: color-mix(in srgb, var(--td-brand-color-6) 68%, var(--td-component-border));
  --infrastructure-path-active: var(--td-brand-color-6);
  --infrastructure-path-active-soft: color-mix(in srgb, var(--td-brand-color-5) 68%, var(--td-component-border));
  --infrastructure-path-neutral: color-mix(in srgb, var(--td-component-border) 82%, var(--td-brand-color-5));

  isolation: isolate;
  min-height: min(540px, calc(100vh - 180px));
  position: relative;
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
  left: 10%;
  top: 34%;
  width: 64%;
}

.infrastructure-canvas__halo--secondary {
  background: radial-gradient(
    circle at 50% 50%,
    var(--infrastructure-halo-secondary-core) 0%,
    var(--infrastructure-halo-secondary-edge) 42%,
    transparent 72%
  );
  right: -2%;
  top: 4%;
  width: 38%;
}

.infrastructure-canvas__topology {
  display: block;
  height: auto;
  margin: 0 0 0 -9%;
  max-height: 560px;
  max-width: none;
  overflow: visible;
  pointer-events: auto;
  position: relative;
  width: min(122%, 860px);
  z-index: 1;
}

.topology__path {
  opacity: 0.76;
  stroke-dasharray: 1;
  stroke-dashoffset: 1;
  stroke-width: 1.4;
  transition:
    filter 190ms ease-out,
    opacity 190ms ease-out,
    stroke 190ms ease-out,
    stroke-width 190ms ease-out;
}

.topology__paths {
  pointer-events: none;
}

.topology__path--neutral {
  stroke: var(--infrastructure-path-neutral);
}

.topology__path--active {
  stroke: url('#graft-active-path');
  stroke-width: 1.7;
}

.topology__path--graft {
  stroke: url('#graft-active-path');
  stroke-width: 2.6;
}

.topology__path--branch {
  stroke-width: 1.35;
}

.topology__path--highlighted {
  filter: drop-shadow(0 0 4px var(--infrastructure-node-halo));
  opacity: 1;
  stroke: var(--infrastructure-path-active);
  stroke-width: 2.35;
}

.topology__path--graft.topology__path--highlighted {
  stroke-width: 3.2;
}

.topology__node {
  cursor: default;
  opacity: 0;
  pointer-events: all;
}

.topology__node-material {
  transform-box: fill-box;
  transform-origin: center;
  transition:
    filter 190ms ease-out,
    transform 190ms ease-out;
}

.topology__node circle:first-child,
.topology__node rect {
  fill: url('#graft-node-acrylic');
  stroke: var(--infrastructure-node-rim);
  stroke-width: 1.2;
}

.topology__node--active circle:first-child,
.topology__node--active rect {
  stroke: var(--infrastructure-path-active);
}

.topology__node-ring {
  fill: color-mix(in srgb, var(--td-bg-color-container) 48%, var(--td-brand-color-6));
  opacity: 0.88;
  stroke: color-mix(in srgb, var(--infrastructure-node-rim) 72%, var(--td-bg-color-container));
  stroke-width: 1;
}

.topology__node-sheen {
  fill: var(--infrastructure-node-highlight);
  opacity: 0.9;
  stroke: none;
}

.topology__node-core {
  fill: var(--infrastructure-path-active);
  opacity: 0.86;
  stroke: none;
  transition:
    filter 190ms ease-out,
    opacity 190ms ease-out;
}

.topology__node--highlighted .topology__node-material {
  filter: drop-shadow(0 0 7px var(--infrastructure-node-halo));
  transform: scale(1.07);
}

.topology__node--highlighted circle:first-child,
.topology__node--highlighted rect {
  stroke: var(--infrastructure-path-active);
  stroke-width: 1.45;
}

.topology__node--highlighted .topology__node-core {
  filter: drop-shadow(0 0 2px var(--infrastructure-node-halo));
  opacity: 1;
}

.topology__graft-connection {
  cursor: default;
  opacity: 0;
  pointer-events: all;
  transition: filter 190ms ease-out;
}

.topology__graft-sleeve {
  fill: none;
  stroke: color-mix(in srgb, var(--td-bg-color-container) 40%, var(--infrastructure-path-active));
  stroke-linecap: round;
  stroke-width: 13;
  transition: stroke 190ms ease-out;
}

.topology__graft-anchor {
  fill: url('#graft-node-acrylic');
  stroke: var(--infrastructure-node-rim);
  stroke-width: 1.25;
  transition:
    filter 190ms ease-out,
    stroke 190ms ease-out;
}

.topology__graft-core {
  fill: var(--infrastructure-path-active);
  opacity: 0.9;
  transition:
    filter 190ms ease-out,
    opacity 190ms ease-out;
}

.topology__graft-seam {
  fill: none;
  stroke: var(--td-bg-color-container);
  stroke-linecap: round;
  stroke-width: 1.25;
}

.topology__graft-connection--highlighted {
  filter: drop-shadow(0 0 7px var(--infrastructure-node-halo));
}

.topology__graft-connection--highlighted .topology__graft-sleeve {
  stroke: color-mix(in srgb, var(--td-bg-color-container) 24%, var(--infrastructure-path-active));
}

.topology__graft-connection--highlighted .topology__graft-anchor {
  stroke: var(--infrastructure-path-active);
}

.topology__graft-connection--highlighted .topology__graft-core {
  filter: drop-shadow(0 0 2px var(--infrastructure-node-halo));
  opacity: 1;
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
  .topology__graft-connection {
    animation: topology-node-in 260ms ease-out both;
    animation-delay: 300ms;
  }

  .topology__graft-connection {
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

:deep([data-theme-mode='dark'] .infrastructure-canvas) {
  --infrastructure-halo-primary-core: color-mix(in srgb, var(--td-brand-color-6) 28%, transparent);
  --infrastructure-halo-primary-edge: color-mix(in srgb, var(--td-brand-color-6) 10%, transparent);
  --infrastructure-halo-secondary-core: color-mix(in srgb, var(--td-brand-color-5) 15%, transparent);
  --infrastructure-halo-secondary-edge: color-mix(in srgb, var(--td-brand-color-5) 5%, transparent);
  --infrastructure-node-depth: color-mix(in srgb, var(--td-bg-color-container) 54%, var(--td-brand-color-6));
  --infrastructure-node-fill: color-mix(in srgb, var(--td-bg-color-container) 36%, var(--td-brand-color-6));
  --infrastructure-node-highlight: color-mix(in srgb, var(--td-bg-color-container) 72%, var(--td-brand-color-1));
  --infrastructure-node-halo: color-mix(in srgb, var(--td-brand-color-6) 26%, transparent);
  --infrastructure-node-rim: color-mix(in srgb, var(--td-brand-color-5) 82%, var(--td-component-border));
  --infrastructure-path-neutral: color-mix(in srgb, var(--td-component-border) 62%, var(--td-brand-color-5));
}

@media (width <= 1023px) {
  .infrastructure-canvas {
    min-height: 300px;
  }

  .infrastructure-canvas__topology {
    margin-left: -5%;
    width: 110%;
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
    transition-duration: 0ms !important;
  }

  .infrastructure-canvas__halo,
  .topology__node,
  .topology__graft-connection {
    opacity: 1;
  }

  .topology__path {
    stroke-dashoffset: 0;
  }
}
</style>
