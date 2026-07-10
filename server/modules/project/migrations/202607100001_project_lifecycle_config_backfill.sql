UPDATE compose_projects
SET lifecycle_config_json = jsonb_build_object(
  'remove_orphans', true,
  'wait_timeout_seconds', 120,
  'renew_anon_volumes', false
) || lifecycle_config_json
WHERE NOT lifecycle_config_json ?& ARRAY[
  'remove_orphans',
  'wait_timeout_seconds',
  'renew_anon_volumes'
];
