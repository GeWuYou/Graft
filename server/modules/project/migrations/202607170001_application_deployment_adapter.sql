ALTER TABLE applications
  RENAME COLUMN application_type TO deployment_adapter_kind;

ALTER TABLE applications
  RENAME CONSTRAINT applications_application_type_check TO applications_deployment_adapter_kind_check;

COMMENT ON COLUMN applications.deployment_adapter_kind IS '应用定义格式和部署适配器类型，当前固定为 compose；运行目标能力决定实际执行模式';
