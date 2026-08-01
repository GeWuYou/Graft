declare module 'monaco-editor/esm/vs/editor/editor.api.js' {
  export * from 'monaco-editor';
}

// 以下声明对应 monaco-editor 0.56.0 的 ESM 副作用入口，由 web 类型边界维护，作为上游类型缺失时的兼容层。
// 升级 monaco-editor 后应重新运行类型检查；上游提供对应声明后删除不再需要的条目，避免兼容层长期滞留。
declare module 'monaco-editor/esm/vs/languages/definitions/dockerfile/register.js';
declare module 'monaco-editor/esm/vs/languages/definitions/hcl/register.js';
declare module 'monaco-editor/esm/vs/languages/definitions/ini/register.js';
declare module 'monaco-editor/esm/vs/languages/definitions/markdown/register.js';
declare module 'monaco-editor/esm/vs/languages/definitions/powershell/register.js';
declare module 'monaco-editor/esm/vs/languages/definitions/shell/register.js';
declare module 'monaco-editor/esm/vs/languages/definitions/sql/register.js';
declare module 'monaco-editor/esm/vs/languages/definitions/xml/register.js';
declare module 'monaco-editor/esm/vs/languages/definitions/yaml/register.js';
declare module 'monaco-editor/esm/vs/language/json/monaco.contribution';
