ALTER TABLE platform_connectivity_checks
  ADD COLUMN http_status INTEGER NULL,
  ADD CONSTRAINT platform_connectivity_checks_http_status_check CHECK (http_status IS NULL OR (http_status >= 100 AND http_status <= 599));

COMMENT ON COLUMN platform_connectivity_checks.http_status IS '最近一次 HTTP 探针收到的响应状态码，未收到响应或不适用时为空';
