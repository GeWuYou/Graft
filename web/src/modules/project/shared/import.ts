import type {
  ProjectImportDirectoryRef,
  ProjectImportDirectorySource,
  ProjectImportInspectResponse,
} from '../types/import';

export function buildDirectorySelection(source: ProjectImportDirectorySource, path: string): ProjectImportDirectoryRef {
  return {
    provider: source.provider,
    root_id: source.root_id,
    path: normalizeDirectoryPath(path),
  };
}

export function initialDirectoryPath(source: ProjectImportDirectorySource) {
  return normalizeDirectoryPath(source.initial_path || '');
}

export function normalizeDirectoryPath(path: string | null | undefined) {
  const normalized = (path || '').trim().replace(/\\/g, '/');
  if (!normalized || normalized === '.') {
    return '';
  }

  return normalized.split('/').filter(Boolean).join('/');
}

function splitDirectorySegments(path: string) {
  const normalized = normalizeDirectoryPath(path);
  return normalized ? normalized.split('/') : [];
}

export function buildDirectoryBreadcrumbs(directory: ProjectImportDirectoryRef) {
  const segments = splitDirectorySegments(directory.path);
  return [
    {
      key: '',
      label: directory.root_id,
      path: '',
    },
    ...segments.map((segment, index) => ({
      key: `${index}:${segment}`,
      label: segment,
      path: segments.slice(0, index + 1).join('/'),
    })),
  ];
}

export function buildDirectorySourceLabel(source: ProjectImportDirectorySource) {
  const suffix = source.managed ? ' (Managed)' : '';
  return `${source.label}${suffix}`;
}

export function buildSuggestedDisplayName(result: ProjectImportInspectResponse) {
  return (result.display_name_suggested || result.canonical_project_name || '').trim();
}

export function hasBlockingImportConflicts(result: ProjectImportInspectResponse | null) {
  return Boolean(result?.conflicts?.length);
}
