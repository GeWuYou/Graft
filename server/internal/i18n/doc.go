// Package i18n 提供 server 运行时使用的项目级本地化 facade。
//
// 该包统一收口 locale 解析、消息查找、注册期 registry 与 freeze 语义。
// 调用方应只依赖这里定义的项目概念，而不是自行扩散第二套本地化入口或
// 直接绑定未来可能替换的底层实现。
package i18n
