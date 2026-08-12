import archive from '@iconify-icons/lucide/archive';
import container from '@iconify-icons/lucide/container';
import downloadCloud from '@iconify-icons/lucide/download-cloud';
import hammer from '@iconify-icons/lucide/hammer';
import hardDrive from '@iconify-icons/lucide/hard-drive';
import imageIcon from '@iconify-icons/lucide/image';
import packageOpen from '@iconify-icons/lucide/package-open';
import waypoints from '@iconify-icons/lucide/waypoints';
import wrench from '@iconify-icons/lucide/wrench';
import docker from '@iconify-icons/tabler/brand-docker';
import { describe, expect, it } from 'vitest';

import { resolveMenuIcon } from './menu-icon';

describe('resolveMenuIcon', () => {
  it('uses the Tabler Docker brand icon for Docker provider menus', () => {
    expect(resolveMenuIcon('docker-provider')).toEqual(docker);
  });

  it('uses a workload icon for container menus', () => {
    expect(resolveMenuIcon('container-workload')).toEqual(container);
  });

  it('uses an artifact icon for image menus', () => {
    expect(resolveMenuIcon('image-artifact')).toEqual(imageIcon);
  });

  it('uses an open package icon for image registries', () => {
    expect(resolveMenuIcon('image-registry')).toEqual(packageOpen);
    expect(resolveMenuIcon('image-registry')).not.toEqual(resolveMenuIcon('image-artifact'));
  });

  it('uses a persistent-storage icon for volume menus', () => {
    expect(resolveMenuIcon('persistent-volume')).toEqual(hardDrive);
  });

  it('uses maintenance and release icons for platform update navigation', () => {
    expect(resolveMenuIcon('system-maintenance')).toEqual(wrench);
    expect(resolveMenuIcon('platform-update')).toEqual(downloadCloud);
  });

  it('uses an archive icon for Backup history navigation', () => {
    expect(resolveMenuIcon('backup')).toEqual(archive);
  });

  it('uses the build hammer icon for Build Jobs navigation', () => {
    expect(resolveMenuIcon('build')).toEqual(hammer);
  });

  it('keeps application and runtime targets semantically distinct', () => {
    expect(resolveMenuIcon('application-portfolio')).not.toEqual(resolveMenuIcon('runtime-target'));
  });

  it('uses a distinct route icon for platform outbound network navigation', () => {
    expect(resolveMenuIcon('platform-network')).toEqual(waypoints);
    expect(resolveMenuIcon('platform-network')).not.toEqual(resolveMenuIcon('infrastructure-domain'));
  });

  it('keeps visible observability entries from falling back to one generic glyph', () => {
    const keys = [
      'observability-overview',
      'service-health',
      'dependency-health',
      'module-health',
      'access-log',
      'application-log',
      'request-performance',
    ];

    expect(new Set(keys.map((key) => resolveMenuIcon(key))).size).toBe(keys.length);
  });

  it('resolves every server-owned semantic menu key without fallback', () => {
    const keys = [
      'application-domain',
      'application-portfolio',
      'application-template',
      'infrastructure-domain',
      'build-domain',
      'build',
      'resources-domain',
      'observability-domain',
      'security-domain',
      'platform-domain',
      'runtime-target',
      'docker-provider',
      'container-workload',
      'image-artifact',
      'image-registry',
      'persistent-volume',
      'network-resource',
      'observability-overview',
      'service-health',
      'dependency-health',
      'request-performance',
      'access-log',
      'application-log',
      'module-health',
      'security-posture',
      'user-identity',
      'role-groups',
      'access-policy',
      'audit-trail',
      'scheduled-automation',
      'platform-configuration',
      'platform-network',
      'system-maintenance',
      'platform-update',
      'backup',
      'announcement-publishing',
    ];

    const fallback = resolveMenuIcon();
    expect(keys.every((key) => resolveMenuIcon(key) !== fallback)).toBe(true);
  });

  it('uses a stable Lucide fallback for unknown server identifiers', () => {
    expect(resolveMenuIcon('unknown-menu-icon')).toEqual(resolveMenuIcon());
  });
});
