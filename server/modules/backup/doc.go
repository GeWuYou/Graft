// Package backup 拥有平台配置快照、数据库 dump 和恢复证据的备份事实。
//
// 模块对外只提供窄 BackupService capability。它不返回工件内容、不提供数据库
// restore HTTP API，也不执行 Atlas migration；这些行为分别属于受控恢复设计和 core CLI。
package backup
