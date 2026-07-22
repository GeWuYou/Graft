import activity from '@iconify-icons/lucide/activity';
import roles from '@iconify-icons/lucide/badge-check';
import bell from '@iconify-icons/lucide/bell';
import boxes from '@iconify-icons/lucide/boxes';
import scheduledTasks from '@iconify-icons/lucide/calendar-clock';
import clock from '@iconify-icons/lucide/clock';
import container from '@iconify-icons/lucide/container';
import downloadCloud from '@iconify-icons/lucide/download-cloud';
import fileSearch from '@iconify-icons/lucide/file-search';
import folder from '@iconify-icons/lucide/folder';
import gauge from '@iconify-icons/lucide/gauge';
import hammer from '@iconify-icons/lucide/hammer';
import hardDrive from '@iconify-icons/lucide/hard-drive';
import heartPulse from '@iconify-icons/lucide/heart-pulse';
import history from '@iconify-icons/lucide/history';
import imageIcon from '@iconify-icons/lucide/image';
import layers from '@iconify-icons/lucide/layers';
import library from '@iconify-icons/lucide/library';
import listTree from '@iconify-icons/lucide/list-tree';
import megaphone from '@iconify-icons/lucide/megaphone';
import network from '@iconify-icons/lucide/network';
import application from '@iconify-icons/lucide/package';
import route from '@iconify-icons/lucide/route';
import search from '@iconify-icons/lucide/search';
import serverCog from '@iconify-icons/lucide/server-cog';
import settings from '@iconify-icons/lucide/settings';
import shield from '@iconify-icons/lucide/shield';
import slidersHorizontal from '@iconify-icons/lucide/sliders-horizontal';
import target from '@iconify-icons/lucide/target';
import users from '@iconify-icons/lucide/users';
import workflow from '@iconify-icons/lucide/workflow';
import wrench from '@iconify-icons/lucide/wrench';
import docker from '@iconify-icons/tabler/brand-docker';

// 服务端菜单声明拥有语义键 authority；此处只负责把已审查的语义键映射到静态资源。
const menuIcons = {
  'application-domain': boxes,
  'application-portfolio': application,
  'application-template': layers,
  'infrastructure-domain': network,
  'build-domain': hammer,
  'resources-domain': library,
  'observability-domain': activity,
  'security-domain': shield,
  'platform-domain': slidersHorizontal,
  'runtime-target': target,
  'docker-provider': docker,
  'container-workload': container,
  'image-artifact': imageIcon,
  'persistent-volume': hardDrive,
  'network-resource': network,
  'observability-overview': activity,
  'service-health': heartPulse,
  'dependency-health': route,
  'request-performance': gauge,
  'access-log': search,
  'application-log': fileSearch,
  'module-health': serverCog,
  'security-posture': shield,
  'user-identity': users,
  'role-groups': roles,
  'access-policy': listTree,
  'audit-trail': history,
  'scheduled-automation': scheduledTasks,
  'platform-configuration': settings,
  'system-maintenance': wrench,
  'platform-update': downloadCloud,
  'announcement-publishing': megaphone,
  notification: bell,
  time: clock,
  workflow,
} as const;

export type MenuIconKey = keyof typeof menuIcons;

/**
 * 将服务器提供的菜单图标标识符解析为已静态打包的图标数据。
 *
 * @param key - 菜单图标标识符
 * @returns 对应的图标数据；标识符缺失或未匹配时返回稳定的异常保护图形
 */
export function resolveMenuIcon(key?: string) {
  if (key && key in menuIcons) {
    return menuIcons[key as MenuIconKey];
  }

  return folder;
}
