export type { ErrorCode as ApiCode } from '../generated/platform';
export { ERROR_CODE as API_CODE } from '../generated/platform';

// 后端可先于前端发布新增响应码，因此保留未知字符串，让通用错误处理继续展示服务端返回的信息。
export type ApiResponseCode = string;
