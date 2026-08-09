// Package credentialvault 拥有 Runtime Target Agent 的非秘密机器身份生命周期边界。
//
// 它将密码材料和 PKI 操作明确委托给部署层提供的 Vault adapter。需要 Agent 信任的模块必须使用
// moduleapi.MachineIdentityAuthority，而不能直接依赖本包。
package credentialvault
