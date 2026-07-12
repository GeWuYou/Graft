import activity from '@iconify-icons/lucide/activity';
import roles from '@iconify-icons/lucide/badge-check';
import bell from '@iconify-icons/lucide/bell';
import box from '@iconify-icons/lucide/box';
import scheduledTasks from '@iconify-icons/lucide/calendar-clock';
import clock from '@iconify-icons/lucide/clock';
import database from '@iconify-icons/lucide/database';
import fileSearch from '@iconify-icons/lucide/file-search';
import folder from '@iconify-icons/lucide/folder';
import history from '@iconify-icons/lucide/history';
import keyRound from '@iconify-icons/lucide/key-round';
import dashboard from '@iconify-icons/lucide/layout-dashboard';
import lock from '@iconify-icons/lucide/lock';
import application from '@iconify-icons/lucide/package';
import search from '@iconify-icons/lucide/search';
import server from '@iconify-icons/lucide/server';
import settings from '@iconify-icons/lucide/settings';
import shield from '@iconify-icons/lucide/shield';
import shieldCheck from '@iconify-icons/lucide/shield-check';
import terminal from '@iconify-icons/lucide/terminal';
import users from '@iconify-icons/lucide/users';
import workflow from '@iconify-icons/lucide/workflow';
import docker from '@iconify-icons/tabler/brand-docker';
import cloudComputing from '@iconify-icons/tabler/cloud-computing';

const menuIcons = {
  application,
  'app-log': fileSearch,
  'access-log': search,
  activity,
  announcements: bell,
  audit: history,
  build: terminal,
  container: box,
  'cloud-computing': cloudComputing,
  config: settings,
  dashboard,
  dependencies: database,
  docker,
  infrastructure: server,
  'module-runtime': workflow,
  notification: bell,
  observability: activity,
  permissions: keyRound,
  platform: settings,
  resources: database,
  roles,
  'runtime-overview': activity,
  'runtime-target': server,
  'scheduled-tasks': scheduledTasks,
  security: shieldCheck,
  'security-overview': shieldCheck,
  folder,
  history,
  lock,
  server,
  setting: settings,
  secured: shield,
  search,
  'file-search': fileSearch,
  time: clock,
  usergroup: users,
  users,
  workflow,
} as const;

export type MenuIconKey = keyof typeof menuIcons;

/**
 * 将服务器提供的菜单图标标识符解析为已静态打包的图标数据。
 *
 * @param key - 菜单图标标识符
 * @returns 对应的图标数据；标识符缺失或未匹配时返回文件夹图标
 */
export function resolveMenuIcon(key?: string) {
  if (key && key in menuIcons) {
    return menuIcons[key as MenuIconKey];
  }

  return folder;
}
