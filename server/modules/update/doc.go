// Package update 提供 Graft 自身版本发现与安装能力预检。
//
// 本模块只拥有版本目录、安装画像和更新资格的业务语义。实际 Compose
// 执行、备份与迁移由后续模块能力协作完成，避免把部署控制面藏入 core。
package update
