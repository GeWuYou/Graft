-- Project Task owner references are cross-module resource identities. They use the
-- immutable public application ID rather than Project's module-private numeric key.
UPDATE tasks AS task
SET owner_id = project.application_id
FROM compose_projects AS project
WHERE task.owner_type = 'compose_project'
  AND task.owner_id = project.id::text
  AND btrim(project.application_id) <> '';
