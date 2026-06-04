CREATE INDEX "idx_app_logs_keyword_search"
ON "app_logs"
USING GIN (
  to_tsvector(
    'simple',
    concat_ws(' ', "component", COALESCE("operation", ''), "message", COALESCE("error", ''))
  )
);
