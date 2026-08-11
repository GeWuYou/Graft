UPDATE registry_connections
SET auth_mode = CASE WHEN credential_ref IS NULL OR credential_ref = '' THEN 'anonymous' ELSE 'credential_ref' END,
    insecure = LOWER(endpoint) LIKE 'http://%',
    verification_status = CASE WHEN verification_status = 'succeeded' THEN 'verified' ELSE verification_status END
WHERE deleted_at = 0;

DROP INDEX IF EXISTS uq_registry_connections_live_provider_endpoint;

WITH duplicate_connections AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY provider, endpoint ORDER BY updated_at DESC, id DESC) AS position
  FROM registry_connections
  WHERE deleted_at = 0
)
UPDATE registry_connections
SET deleted_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
    deleted_by = 0,
    updated_at = NOW(),
    updated_by = 0
WHERE id IN (SELECT id FROM duplicate_connections WHERE position > 1);

CREATE UNIQUE INDEX uq_registry_connections_live_provider_endpoint
ON registry_connections (provider, endpoint)
WHERE deleted_at = 0;
