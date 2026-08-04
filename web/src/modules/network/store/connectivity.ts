import { defineStore } from 'pinia';

import {
  type ConnectivityAggregate,
  type ConnectivityCheck,
  type ConnectivityCustomTarget,
  type ConnectivityReport,
  type ConnectivityTarget,
  createConnectivityCustomTarget,
  deleteConnectivityCustomTarget,
  getConnectivityAggregate,
  getConnectivityCustomTargets,
  getConnectivityExport,
  getConnectivityHistory,
  getConnectivityLatest,
  getConnectivityReport,
  getConnectivityTargets,
  getConnectivityTrace,
  runConnectivityBatch,
  runConnectivityTarget,
} from '../api/connectivity';

export const useConnectivityStore = defineStore('network-connectivity', {
  state: () => ({
    targets: [] as ConnectivityTarget[],
    customTargets: [] as ConnectivityCustomTarget[],
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
        const [targets, customTargets, latest, aggregate] = await Promise.all([
          getConnectivityTargets(),
          getConnectivityCustomTargets(),
          getConnectivityLatest(),
          getConnectivityAggregate(),
        ]);
        this.targets = targets.items;
        this.customTargets = customTargets.items;
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
    async loadTrace(targetId: string, checkId: number) {
      return getConnectivityTrace(targetId, checkId);
    },
    async exportReport(targetId: string, checkId: number) {
      return getConnectivityExport(targetId, checkId);
    },
    async createCustomTarget(data: Parameters<typeof createConnectivityCustomTarget>[0]) {
      const target = await createConnectivityCustomTarget(data);
      this.customTargets = [...this.customTargets.filter((item) => item.id !== target.id), target];
      return target;
    },
    async deleteCustomTarget(targetId: string) {
      await deleteConnectivityCustomTarget(targetId);
      this.customTargets = this.customTargets.filter((item) => item.id !== targetId);
      this.latest = this.latest.filter((item) => item.target_id !== targetId);
      this.history.delete(targetId);
      for (const key of this.reports.keys()) {
        if (key.startsWith(`${targetId}:`)) this.reports.delete(key);
      }
    },
  },
});
