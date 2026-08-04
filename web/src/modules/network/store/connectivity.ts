import { defineStore } from 'pinia';

import {
  type ConnectivityAggregate,
  type ConnectivityCheck,
  type ConnectivityReport,
  type ConnectivityTarget,
  getConnectivityAggregate,
  getConnectivityHistory,
  getConnectivityLatest,
  getConnectivityReport,
  getConnectivityTargets,
  runConnectivityBatch,
  runConnectivityTarget,
} from '../api/connectivity';

export const useConnectivityStore = defineStore('network-connectivity', {
  state: () => ({
    targets: [] as ConnectivityTarget[],
    latest: [] as ConnectivityCheck[],
    aggregate: null as ConnectivityAggregate | null,
    history: new Map<string, ConnectivityCheck[]>(),
    reports: new Map<string, ConnectivityReport>(),
    loading: false,
    running: false,
  }),
  actions: {
    async refresh() {
      this.loading = true;
      try {
        const [targets, latest, aggregate] = await Promise.all([
          getConnectivityTargets(),
          getConnectivityLatest(),
          getConnectivityAggregate(),
        ]);
        this.targets = targets.items;
        this.latest = latest.items;
        this.aggregate = aggregate;
      } finally {
        this.loading = false;
      }
    },
    async runAll() {
      this.running = true;
      try {
        await runConnectivityBatch();
        await this.refresh();
      } finally {
        this.running = false;
      }
    },
    async runTarget(targetId: string) {
      this.running = true;
      try {
        const result = await runConnectivityTarget(targetId);
        this.reports.set(`${targetId}:${result.check.check_id}`, result.report);
        await this.refresh();
        return result;
      } finally {
        this.running = false;
      }
    },
    async loadHistory(targetId: string) {
      const result = await getConnectivityHistory(targetId);
      this.history.set(targetId, result.items);
      return result.items;
    },
    async loadReport(targetId: string, checkId: number) {
      const key = `${targetId}:${checkId}`;
      const cached = this.reports.get(key);
      if (cached) return cached;
      const report = await getConnectivityReport(targetId, checkId);
      this.reports.set(key, report);
      return report;
    },
  },
});
