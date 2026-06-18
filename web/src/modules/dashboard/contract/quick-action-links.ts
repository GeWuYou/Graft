// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export type DashboardQuickActionLink = {
  id: string;
  module_key: string;
  title_key?: string;
  title?: string;
  group_key?: string;
  group?: string;
  full_label_key?: string;
  full_label?: string;
  description_key?: string;
  description?: string;
  icon?: string;
  route_location: string;
  required_permissions?: string[];
  order: number;
};
