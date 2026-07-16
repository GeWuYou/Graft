// 这是 Vite 的 Worker 入口包装文件；第三方模块在 Worker 上下文中注册 YAML 服务，不能移到主线程入口。
import 'monaco-yaml/yaml.worker.js';
