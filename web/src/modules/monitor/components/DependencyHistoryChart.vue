<template>
  <div class="dependency-history-chart" :data-history-state="state">
    <div v-if="state === 'ready' || state === 'partial'" ref="chartElement" class="dependency-history-chart__canvas" />
    <p v-else class="dependency-history-chart__message">{{ message }}</p>
    <p v-if="state === 'partial'" class="dependency-history-chart__message">{{ message }}</p>
  </div>
</template>
<script setup lang="ts">
import { LineChart } from 'echarts/charts';
import { GridComponent, TooltipComponent } from 'echarts/components';
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue';

import { useSettingStore } from '@/store';

import { formatChartTimeOnly } from '../shared/time-display';
import type { ServerStatusDependencyHistoryPoint } from '../types/server-status';

echarts.use([GridComponent, TooltipComponent, LineChart, CanvasRenderer]);

// 依赖历史图只消费 monitor 返回的聚合点，并随当前主题重新同步 ECharts 配置。
const props = defineProps<{
  state: 'ready' | 'empty' | 'partial' | 'unavailable';
  message: string;
  points: ServerStatusDependencyHistoryPoint[];
  availabilityLabel: string;
  latencyLabel: string;
}>();

const settingStore = useSettingStore();
const chartElement = ref<HTMLDivElement | null>(null);
let instance: echarts.ECharts | null = null;

function renderChart() {
  if ((props.state !== 'ready' && props.state !== 'partial') || !chartElement.value) {
    return;
  }

  instance ??= echarts.init(chartElement.value);
  const availabilityColor = readThemeColor('--td-success-color-5', '#2ba471');
  const latencyColor = readThemeColor('--td-brand-color', '#0052d9');
  const labels = props.points.map((point) => formatChartTimeOnly(point.observed_at));
  instance.setOption({
    animation: false,
    grid: { left: 6, right: 6, top: 10, bottom: 4 },
    tooltip: {
      trigger: 'axis',
      backgroundColor: settingStore.chartColors.containerColor,
      borderColor: settingStore.chartColors.borderColor,
      textStyle: { color: settingStore.chartColors.textColor },
      formatter: (items: Array<{ axisValueLabel?: string; color: string; data: number | null; seriesName: string }>) =>
        items
          .filter((item) => item.data !== null)
          .map(
            (item, index) =>
              `${index === 0 ? `${item.axisValueLabel ?? ''}<br/>` : ''}<span style="color:${item.color}">●</span> ${item.seriesName}: ${item.data}`,
          )
          .join('<br/>'),
    },
    xAxis: { type: 'category', data: labels, show: false, boundaryGap: false },
    yAxis: [
      { type: 'value', min: 0, max: 100, show: false },
      { type: 'value', show: false },
    ],
    series: [
      {
        name: props.availabilityLabel,
        type: 'line',
        yAxisIndex: 0,
        data: props.points.map((point) => point.availability_percent ?? null),
        smooth: true,
        showSymbol: false,
        connectNulls: false,
        lineStyle: { color: availabilityColor, width: 2 },
        itemStyle: { color: availabilityColor },
      },
      {
        name: props.latencyLabel,
        type: 'line',
        yAxisIndex: 1,
        data: props.points.map((point) => point.latency_p95_ms ?? null),
        smooth: true,
        showSymbol: false,
        connectNulls: false,
        lineStyle: { color: latencyColor, width: 2 },
        itemStyle: { color: latencyColor },
      },
    ],
  });
}

function readThemeColor(token: string, fallback: string) {
  void settingStore.resolvedThemeTokensForDisplayMode;
  return getComputedStyle(document.documentElement).getPropertyValue(token).trim() || fallback;
}

function resizeChart() {
  instance?.resize();
}

watch(
  () => [
    props.state,
    props.points,
    props.availabilityLabel,
    props.latencyLabel,
    settingStore.displayMode,
    settingStore.brandTheme,
  ],
  () => nextTick(renderChart),
  { deep: true },
);

onMounted(() => {
  void nextTick(renderChart);
  window.addEventListener('resize', resizeChart);
});

onUnmounted(() => {
  instance?.dispose();
  window.removeEventListener('resize', resizeChart);
});
</script>
<style scoped lang="less">
.dependency-history-chart {
  min-height: 72px;
}

.dependency-history-chart__canvas {
  height: 72px;
  width: 100%;
}

.dependency-history-chart__message {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  font: var(--td-font-body-small);
  margin: 0;
  min-height: 72px;
}
</style>
