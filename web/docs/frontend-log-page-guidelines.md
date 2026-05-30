# Frontend Log Page Guidelines

日志页统一落在 `query-builder-list-detail` 页型下。

## Summary

- 适用范围：`audit logs`、`access logs`，以及未来的 `app log`、`monitor log`
- 不创建统一业务模块；各日志能力继续留在所属模块
- 共享仅上提到 `web/src/shared/**`

## Page Structure

- `page header`
  - 所属领域
  - 页面标题
  - 页面说明
  - Refresh
- `query workbench`
  - 主搜索框
  - 时间范围
  - 动态筛选条件
  - 快捷筛选
  - 查询 / 重置
  - 已启用筛选标签
- `result surface`
  - 当前数量
  - 数据说明
  - 表格
  - 分页
- `detail drawer`
  - 完整上下文
  - 关联跳转
  - metadata

## Table Rules

- ID 字段统一使用：
  - monospace
  - 单行
  - ellipsis
  - tooltip
- 适用字段：
  - `requestId`
  - `traceId`
  - `troubleshootingId`
  - `correlationId`
- 时间字段统一使用 locale formatter
- 禁止直接展示原始 ISO
- 状态字段统一使用 `Tag`
- 操作列固定宽度并固定在右侧

## Query Rules

- 默认排序：按原始发生时间倒序
- 页面允许主搜索框 + 动态筛选构建器并存
- URL query 统一使用 `snake_case`
- 共享 query 键：
  - `request_id`
  - `trace_id`
  - `user_id`
  - `username`
  - `occurred_from`
  - `occurred_to`
- 页面进入时：
  - 自动回填条件
  - 自动执行查询

## I18n Rules

- 日志页禁止硬编码文案
- 标题、按钮、列名、筛选项、抽屉字段、空状态必须同时覆盖 `zh-CN` 与 `en-US`
