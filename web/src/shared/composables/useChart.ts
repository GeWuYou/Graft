import * as echarts from 'echarts/core';
import type { ShallowRef } from 'vue';
import { onMounted, onUnmounted, shallowRef } from 'vue';

export const useChart = (domId: string): ShallowRef<echarts.ECharts | undefined> => {
  let chartContainer: HTMLCanvasElement | null = null;
  const chart = shallowRef<echarts.ECharts>();

  const updateContainer = () => {
    if (!chart.value || !chartContainer) {
      return;
    }

    chart.value.resize({
      width: chartContainer.clientWidth,
      height: chartContainer.clientHeight,
    });
  };

  onMounted(() => {
    if (!chartContainer) {
      chartContainer = document.getElementById(domId) as HTMLCanvasElement | null;
    }

    if (!chartContainer) {
      return;
    }

    chart.value = echarts.init(chartContainer);
    window.addEventListener('resize', updateContainer, false);
  });

  onUnmounted(() => {
    window.removeEventListener('resize', updateContainer);
    chart.value?.dispose();
  });

  return chart;
};
