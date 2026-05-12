module graft/server

go 1.25.0

// Gin 是 server 的 HTTP 路由、中间件和请求处理框架。
require github.com/gin-gonic/gin v1.12.0

require (
	// gopkg 提供 Sonic 等 ByteDance 组件复用的底层工具能力。
	github.com/bytedance/gopkg v0.1.4 // indirect
	// Sonic 提供 Gin 可选使用的高性能 JSON 编解码能力。
	github.com/bytedance/sonic v1.15.1 // indirect
	// sonic/loader 支持 Sonic 在不同平台加载运行时优化代码。
	github.com/bytedance/sonic/loader v0.5.1 // indirect
	// base64x 提供 Sonic 依赖的高性能 Base64 编解码实现。
	github.com/cloudwego/base64x v0.1.7 // indirect
	// mimetype 用于 Gin binding 判断上传内容或请求体的媒体类型。
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	// gin-contrib/sse 提供 Gin 的 Server-Sent Events 响应支持。
	github.com/gin-contrib/sse v1.1.1 // indirect
	// locales 提供 validator 翻译错误消息所需的本地化数据。
	github.com/go-playground/locales v0.14.1 // indirect
	// universal-translator 为 validator 提供多语言消息翻译能力。
	github.com/go-playground/universal-translator v0.18.1 // indirect
	// validator 提供 Gin binding 使用的结构体验证能力。
	github.com/go-playground/validator/v10 v10.30.2 // indirect
	// go-json 提供 Gin 可选使用的高性能 JSON 编解码实现。
	github.com/goccy/go-json v0.10.6 // indirect
	// go-yaml 提供 Gin binding 支持 YAML 请求体解析的能力。
	github.com/goccy/go-yaml v1.19.2 // indirect
	// json-iterator 提供 Gin 可选使用的兼容 JSON 编解码实现。
	github.com/json-iterator/go v1.1.12 // indirect
	// cpuid 用于 JSON 和压缩相关依赖检测 CPU 指令集能力。
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	// go-urn 提供 validator 使用的 URN 格式解析与校验支持。
	github.com/leodido/go-urn v1.4.0 // indirect
	// go-isatty 用于 Gin 判断输出是否连接到终端以调整日志表现。
	github.com/mattn/go-isatty v0.0.22 // indirect
	// concurrent 提供 json-iterator 依赖的并发辅助结构。
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	// reflect2 提供 json-iterator 依赖的反射增强能力。
	github.com/modern-go/reflect2 v1.0.2 // indirect
	// go-toml 提供 Gin binding 支持 TOML 请求体解析的能力。
	github.com/pelletier/go-toml/v2 v2.3.1 // indirect
	// qpack 提供 QUIC HTTP/3 依赖的 QPACK 头部压缩实现。
	github.com/quic-go/qpack v0.6.0 // indirect
	// quic-go 提供 Gin 依赖链中 HTTP/3 支持所需的 QUIC 实现。
	github.com/quic-go/quic-go v0.59.1 // indirect
	// golang-asm 提供 Sonic 生成和运行汇编优化代码所需的工具能力。
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	// ugorji/codec 提供 Gin binding 支持 MsgPack 等格式编解码的能力。
	github.com/ugorji/go/codec v1.3.1 // indirect
	// mongo-driver 提供 Gin binding 支持 BSON 请求体解析的能力。
	go.mongodb.org/mongo-driver/v2 v2.6.0 // indirect
	// x/arch 提供 Sonic 和汇编相关依赖需要的 CPU 架构描述能力。
	golang.org/x/arch v0.27.0 // indirect
	// x/crypto 提供 HTTP/3、校验和编码链路使用的扩展密码学能力。
	golang.org/x/crypto v0.51.0 // indirect
	// x/net 提供 Gin 与 HTTP/2、HTTP/3 链路依赖的网络协议扩展。
	golang.org/x/net v0.54.0 // indirect
	// x/sys 提供终端判断、网络和底层平台调用所需的系统接口。
	golang.org/x/sys v0.44.0 // indirect
	// x/text 提供 validator 与网络协议处理所需的 Unicode 和语言能力。
	golang.org/x/text v0.37.0 // indirect
	// protobuf 提供 Gin binding 处理 Protocol Buffers 请求体的能力。
	google.golang.org/protobuf v1.36.11 // indirect
)
