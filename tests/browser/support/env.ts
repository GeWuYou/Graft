// Browser acceptance environment inputs. Every value is injected by
// scripts/browser_acceptance.py and mirrors tests/browser/compose/browser.env.example.
export const browserEnv = {
  baseURL: process.env.GRAFT_BROWSER_WEB_BASE_URL ?? 'http://127.0.0.1:4173',
  adminUsername: process.env.GRAFT_BROWSER_ADMIN_USERNAME ?? 'graft',
  adminPassword: process.env.GRAFT_BROWSER_ADMIN_PASSWORD ?? 'graft-admin',
  acceptancePassword: process.env.GRAFT_BROWSER_ACCEPTANCE_PASSWORD ?? 'graft-browser-acceptance-01',
  suite: process.env.GRAFT_BROWSER_SUITE ?? '',
  engine: process.env.GRAFT_BROWSER_ENGINE ?? 'chromium',
  tier: process.env.GRAFT_BROWSER_TIER ?? 'local',
} as const;

// Deterministic fixture identifiers; must stay aligned with the
// TestSeedBrowserFixture Go fixtures in server/modules/{project,task,runtime-target}.
export const browserFixture = {
  applicationId: 'app_APPBR0WSERFXTR000000000000',
  applicationDisplayName: 'Browser Fixture Application',
  runtimeTargetDisplayName: 'Browser Fixture Target A',
} as const;
