import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT_DIR = fileURLToPath(new URL('..', import.meta.url));
const MANIFEST_PATH = join(ROOT_DIR, 'docs/responsive/manifest.json');
const BUSINESS_DIRS = [join(ROOT_DIR, 'src/app'), join(ROOT_DIR, 'src/modules')];
const SOURCE_EXTENSIONS = new Set(['.ts', '.tsx', '.vue']);
const FORBIDDEN_PATTERNS = [
  'window.innerWidth',
  'document.body.clientWidth',
  'screen.width',
  'matchMedia',
  'useMediaQuery',
  'isMobile',
];

type ManifestProfile = { components: string[]; kind: string; path: string; status: string };
type ManifestException = { cleanupPhase: string; path: string; reason: string; replacement: string };
type ManifestDebt = {
  deadline: string;
  issue: string;
  migrationPhase: string;
  owner: string;
  reason: string;
  replacement: string;
};
type ResponsiveManifest = {
  defaults: { pageEntry: string; requiredWidths: number[] };
  debt: ManifestDebt[];
  exceptions: ManifestException[];
  profiles: ManifestProfile[];
  version: number;
};

function walk(directory: string): string[] {
  if (!existsSync(directory)) return [];
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) return walk(path);
    return SOURCE_EXTENSIONS.has(extname(path)) && !/\.(test|spec)\.[tj]sx?$/u.test(path) ? [path] : [];
  });
}

function validateManifest(manifest: ResponsiveManifest): string[] {
  const findings: string[] = [];
  if (manifest.version !== 1) findings.push('manifest version must be 1');
  if (manifest.defaults?.pageEntry !== 'ResponsivePage') findings.push('defaults.pageEntry must be ResponsivePage');
  if (JSON.stringify(manifest.defaults?.requiredWidths) !== JSON.stringify([375, 768, 992, 1200]))
    findings.push('defaults.requiredWidths must be [375, 768, 992, 1200]');
  for (const profile of manifest.profiles ?? [])
    if (!profile.path || !profile.kind || !profile.status || profile.components.length === 0)
      findings.push(`profile is incomplete: ${profile.path || '<unknown>'}`);
  for (const debt of manifest.debt ?? [])
    if (!debt.issue || !debt.owner || !debt.deadline || !debt.migrationPhase || !debt.reason || !debt.replacement)
      findings.push(`debt is incomplete: ${debt.issue || '<unknown>'}`);
  for (const exception of manifest.exceptions ?? [])
    if (!exception.path || !exception.reason || !exception.replacement || !exception.cleanupPhase)
      findings.push(`exception is incomplete: ${exception.path || '<unknown>'}`);
  return findings;
}

function validateBusinessSource(): string[] {
  const findings: string[] = [];
  for (const file of BUSINESS_DIRS.flatMap(walk)) {
    const source = readFileSync(file, 'utf8');
    const path = relative(ROOT_DIR, file).replaceAll('\\', '/');
    for (const token of FORBIDDEN_PATTERNS)
      if (source.includes(token)) findings.push(`${path}: forbidden responsive device API ${token}`);
  }
  return findings;
}

export function runResponsiveGovernanceAudit(): string[] {
  if (!existsSync(MANIFEST_PATH)) return ['missing docs/responsive/manifest.json'];
  return [
    ...validateManifest(JSON.parse(readFileSync(MANIFEST_PATH, 'utf8')) as ResponsiveManifest),
    ...validateBusinessSource(),
  ];
}

const findings = runResponsiveGovernanceAudit();
if (findings.length > 0) {
  process.stderr.write(`Responsive governance failed:\n${findings.map((finding) => `- ${finding}`).join('\n')}\n`);
  process.exitCode = 1;
} else process.stdout.write('Responsive governance passed.\n');
