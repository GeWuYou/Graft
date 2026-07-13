-- 删除已改由部署环境拥有的容器运行时配置覆盖值，避免遗留覆盖值成为第二配置来源。
DELETE FROM system_config_values
WHERE key IN ('ops.container.runtime', 'ops.container.docker.endpoint');
