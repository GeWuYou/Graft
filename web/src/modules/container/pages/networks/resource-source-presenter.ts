export type NetworkSourceKind = 'compose' | 'swarm' | 'docker' | 'custom' | 'unknown';

export interface NetworkResourceSource {
  kind?: NetworkSourceKind;
  compose_project?: string;
  compose_network?: string;
  compose_version?: string;
  swarm_stack?: string;
}

export interface NetworkLabelGroups {
  system?: Record<string, string>;
  user?: Record<string, string>;
}

export interface NetworkResourceMetadata {
  source?: NetworkResourceSource;
  label_groups?: NetworkLabelGroups;
}

export interface ResourceSourcePresentation {
  kind: NetworkSourceKind;
  identity?: string;
  userLabel?: string;
  remainingUserLabelCount: number;
  sourceFields: Array<{ key: 'composeProject' | 'composeNetwork' | 'composeVersion' | 'swarmStack'; value: string }>;
  systemLabels: Array<[string, string]>;
  userLabels: Array<[string, string]>;
}

function sortedEntries(labels: Record<string, string> | undefined): Array<[string, string]> {
  return Object.entries(labels ?? {}).sort(([left], [right]) => left.localeCompare(right));
}

/** 仅呈现服务端已经归类的网络元数据，不从原始 Docker labels 推导来源或标签归属。 */
export function presentNetworkResourceSource(metadata: NetworkResourceMetadata): ResourceSourcePresentation {
  const source = metadata.source;
  const kind = source?.kind ?? 'unknown';
  const userLabels = sortedEntries(metadata.label_groups?.user);
  const sourceFields =
    kind === 'compose'
      ? [
          source?.compose_project && { key: 'composeProject' as const, value: source.compose_project },
          source?.compose_network && { key: 'composeNetwork' as const, value: source.compose_network },
          source?.compose_version && { key: 'composeVersion' as const, value: source.compose_version },
        ].filter((field): field is { key: 'composeProject' | 'composeNetwork' | 'composeVersion'; value: string } =>
          Boolean(field),
        )
      : kind === 'swarm' && source?.swarm_stack
        ? [{ key: 'swarmStack' as const, value: source.swarm_stack }]
        : [];

  return {
    kind,
    identity: kind === 'compose' ? source?.compose_project : kind === 'swarm' ? source?.swarm_stack : undefined,
    userLabel: userLabels[0] ? `${userLabels[0][0]}=${userLabels[0][1]}` : undefined,
    remainingUserLabelCount: Math.max(0, userLabels.length - 1),
    sourceFields,
    systemLabels: sortedEntries(metadata.label_groups?.system),
    userLabels,
  };
}
