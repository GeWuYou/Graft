export type UpdateChannel = 'stable' | 'beta' | 'unknown';
export type UpdateCapability = 'compose_upgrade_available' | 'manual_guidance';

export type InstallationProfile = {
  declared_mode: 'compose' | 'binary' | 'unknown';
  detected_mode: 'compose' | 'binary';
  capability: UpdateCapability;
  guidance: string;
};

export type UpdateRelease = {
  version: string;
  channel: Exclude<UpdateChannel, 'unknown'>;
  notes: string;
  published_at: string;
  manifest_url: string;
  server_digest: string;
  web_digest: string;
  checksums_url?: string;
};

export type UpdateStatus = {
  current_version: string;
  channel: UpdateChannel;
  latest?: UpdateRelease;
  installation_profile: InstallationProfile;
  checked_at?: string;
  check_error?: string;
};
