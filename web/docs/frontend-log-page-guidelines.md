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
- 快捷筛选必须编译为当前页筛选器已支持的字段，不得注入筛选器无法表达的隐藏条件
- 快捷筛选应用后，所有条件必须同时满足：
  - 筛选器控件可回显
  - 已启用筛选标签可见
  - 用户可修改
  - 用户可移除
  - 用户可手动重新构造
- 快捷筛选本质上是一组普通筛选条件：
  - 不得写入业务钻口 `scope`
  - 不得借用只读 drilldown banner 表达快捷筛选
  - 不得把快捷筛选实现成“伪跨页跳转上下文”
- 业务钻口 `scope` 只允许来自其他页面的“查看审计”类跳转或后端返回的 drilldown authority：
  - 它表示外部导航上下文，不是快捷筛选
  - 它必须可退出或可转换为普通筛选
  - 退出后页面应回到纯普通筛选语义
- 多值快捷筛选只能建立在当前 authority 已支持的多值筛选字段上
- 如果后端契约只支持单值字段，前端不得自行扩展成多值快捷筛选语义
- URL query 统一使用 `snake_case`
- 日志页页面状态统一保存 `YYYY-MM-DD HH:mm:ss` 本地展示时间字符串
- Route Query 与 API Request 必须复用同一个 shared UTC conversion helper
- 禁止页面分别实现 UTC 转换逻辑
- 禁止 `Date -> toISOString() -> DatePicker / DateRangePicker` 直接绑定链路
- 提交请求时统一执行：`本地展示时间 -> ISO UTC`
- 恢复 deep link 时统一执行：`ISO UTC -> 本地展示时间`
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
