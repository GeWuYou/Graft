import { createConsola } from 'consola';

import type { LogEvent, LoggerTransport } from '@/utils/logger/types';

import { isPlainObject } from '../object';

// 日志级别由上层 LoggerCore 统一裁决，transport 必须放行已经通过筛选的事件。
const logger = createConsola({ level: Number.POSITIVE_INFINITY });
const MAX_FIELD_COUNT = 12;
const MAX_VALUE_LENGTH = 240;

/**
 * 将字符串中的换行、回车和制表符替换为空格，并移除首尾空白。
 *
 * @param value - 待处理的字符串
 * @returns 规范化后的单行字符串
 */
function toSingleLine(value: string) {
  return value.replaceAll(/[\r\n\t]+/g, ' ').trim();
}

/**
 * 将字符串截断至最大长度，超出长度时以省略号结尾。
 *
 * @param value - 待截断的字符串
 * @returns 截断后的字符串
 */
function truncate(value: string) {
  return value.length > MAX_VALUE_LENGTH ? `${value.slice(0, MAX_VALUE_LENGTH - 3)}...` : value;
}

/**
 * 将值格式化为适合日志输出的字符串。
 *
 * 字符串会进行单行化和截断；日期转换为 ISO 字符串；无法序列化的值表示为 `"[unserializable]"`。
 *
 * @param value - 要格式化的值
 * @returns 格式化后的字符串
 */
function formatValue(value: unknown): string {
  if (value === null) {
    return 'null';
  }

  if (value === undefined) {
    return 'undefined';
  }

  if (value instanceof Date) {
    return value.toISOString();
  }

  if (typeof value === 'string') {
    return truncate(JSON.stringify(toSingleLine(value)));
  }

  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value);
  }

  try {
    const serialized = JSON.stringify(value);
    return serialized === undefined ? '"[unserializable]"' : truncate(toSingleLine(serialized));
  } catch {
    return '"[unserializable]"';
  }
}

/**
 * 将值展开为带前缀的键值字段，并限制字段数量及循环引用的递归处理。
 *
 * @param value - 要展开或格式化的值
 * @param prefix - 当前字段使用的键名前缀
 * @param fields - 用于收集格式化字段的数组
 * @param seen - 用于检测循环引用的对象集合
 */
function appendFields(value: unknown, prefix: string, fields: string[], seen: WeakSet<object>) {
  if (fields.length >= MAX_FIELD_COUNT) {
    return;
  }

  if (value instanceof Error) {
    const errorPrefix = prefix || 'error';
    fields.push(`${errorPrefix}.name=${formatValue(value.name)}`);
    if (fields.length < MAX_FIELD_COUNT) {
      fields.push(`${errorPrefix}.message=${formatValue(value.message)}`);
    }
    return;
  }

  if (isPlainObject(value)) {
    if (seen.has(value)) {
      fields.push(`${prefix}="[circular]"`);
      return;
    }

    seen.add(value);
    for (const [key, child] of Object.entries(value)) {
      appendFields(child, prefix ? `${prefix}.${key}` : key, fields, seen);
      if (fields.length >= MAX_FIELD_COUNT) {
        break;
      }
    }
    seen.delete(value);
    return;
  }

  fields.push(`${prefix}=${formatValue(value)}`);
}

/**
 * 将日志事件格式化为包含时间戳、消息及附加字段的单行文本。
 *
 * @param event - 要格式化的日志事件
 * @returns 格式化后的单行日志文本
 */
function formatConsolaLine(event: LogEvent) {
  const fields: string[] = [];
  const seen = new WeakSet<object>();

  if (event.meta !== undefined) {
    appendFields(event.meta, '', fields, seen);
  }
  if (event.error !== undefined) {
    appendFields(event.error, 'error', fields, seen);
  }

  const eventLine = `${event.timestamp.toISOString()} | ${toSingleLine(event.message)}`;
  return fields.length > 0 ? `${eventLine} | ${fields.join(' ')}` : eventLine;
}

/**
 * 创建一个将日志事件格式化并路由到 Consola 的传输器。
 *
 * @returns 配置完成的 Consola 日志传输器
 */
export function createConsolaTransport(): LoggerTransport {
  return {
    log(event) {
      const taggedLogger = logger.withTag(event.moduleName);
      const eventLine = formatConsolaLine(event);

      switch (event.level) {
        case 'debug':
          taggedLogger.debug(eventLine);
          return;
        case 'info':
          taggedLogger.info(eventLine);
          return;
        case 'warn':
          taggedLogger.warn(eventLine);
          return;
        case 'error':
          taggedLogger.error(eventLine);
          return;
        case 'silent':
          return;
      }
    },
  };
}
