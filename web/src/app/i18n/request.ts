export const LOCALE_REQUEST_HEADER = 'Accept-Language';

/**
 * 统一构造 locale 请求头，后续无论是 axios、fetch 还是模块内自定义请求，
 * 都应复用这里而不是分散拼接 `Accept-Language`。
 */
export function createLocaleRequestHeaders(
  locale: string,
  headers: Record<string, string> = {},
): Record<string, string> {
  return {
    ...headers,
    [LOCALE_REQUEST_HEADER]: locale,
  };
}
