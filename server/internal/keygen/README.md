# keygen

`keygen` 用于生成可直接粘贴到 `server/.env` 的本地 auth 密钥配置行。

边界说明：

* 只负责生成随机密钥文本
* 不负责修改 `.env` 文件
* 不参与 `server` 运行时配置加载
* 不改变 `JWTSecret`、`SigningKey` 或 token 的语义

当前由 `server/cmd/graft-jwt-secret` 与 `server/cmd/graft-signing-key` 两个独立小程序复用。
