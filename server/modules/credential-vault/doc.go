// Package credentialvault 拥有 Runtime Target Agent 的 PKI 签发边界。
//
// 它将密码材料和 PKI 操作明确委托给部署层提供的 Vault adapter。需要 Agent 信任的模块必须使用
// moduleapi.AgentCertificateIssuer，而不能直接依赖本包。Runtime Target 保留 Agent 登记、绑定与激活的业务权限。
package credentialvault
