import type { AxiosRequestConfig } from 'axios';

import type { ApiResponseCode } from '@/contracts/api/codes';
import type { ApiErrorEnvelope } from '@/contracts/api/envelope';

export interface RequestOptions {
  apiUrl?: string;
  isJoinPrefix?: boolean;
  urlPrefix?: string;
  joinParamsToUrl?: boolean;
  formatDate?: boolean;
  isTransformResponse?: boolean;
  isReturnNativeResponse?: boolean;
  ignoreCancelToken?: boolean;
  joinTime?: boolean;
  withToken?: boolean;
  retry?: {
    count: number;
    delay: number;
  };
  throttle?: {
    delay: number;
  };
  debounce?: {
    delay: number;
  };
}

export interface ApiRequestError extends Error {
  status: number;
  code: ApiResponseCode;
  traceId: string;
  messageKey?: string;
  locale?: string;
  responseData?: ApiErrorEnvelope | unknown;
  isApiRequestError: true;
  isPlatformUnavailable?: boolean;
}

export interface AxiosRequestConfigRetry extends AxiosRequestConfig {
  retryCount?: number;
  _authRefreshAttempted?: boolean;
  _skipAuthRefresh?: boolean;
}
