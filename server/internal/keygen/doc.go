// Package keygen 负责生成可直接写入本地 `.env` 的 auth 密钥文本。
//
// 该包只提供开发辅助型的随机密钥生成能力，不参与运行时 token 语义、
// 配置回退策略或任何 auth 校验流程。
package keygen
