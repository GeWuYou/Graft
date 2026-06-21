import { describe, expect, it } from 'vitest';

import {
  buildDisplayLogLine,
  parseContainerLogLine,
  type ParsedLogMetadata,
  parseLogLine,
  summarizeMetadata,
} from './log-parser';

describe('log-parser', () => {
  it('parses ordinary INFO logs into structured fields', () => {
    const line = parseLogLine(
      '2026-06-17T06:31:40.491+0800 INFO service/pricing_service.go:145 [Pricing] Service initialized',
      8,
    );

    expect(line.lineNo).toBe(8);
    expect(line.timestamp).toBe('2026-06-17T06:31:40.491+0800');
    expect(line.level).toBe('INFO');
    expect(line.source).toBe('service/pricing_service.go:145');
    expect(line.sourceShort).toBe('pricing_service.go:145');
    expect(line.message).toBe('[Pricing] Service initialized');
    expect(line.metadata).toBeNull();
    expect(line.parsed.format).toBe('structured');
  });

  it('parses WARN logs and maps warning tone without strong row coloring', () => {
    const line = parseLogLine(
      '2026-06-17T06:31:40.497+0800 WARN stdlog server/wire_gen.go:262 Warning: server.trusted_proxies is empty',
      15,
    );

    expect(line.level).toBe('WARN');
    expect(line.source).toBe('stdlog server/wire_gen.go:262');
    expect(line.sourceShort).toBe('wire_gen.go:262');
    expect(line.tone).toBe('warning');
    expect(line.message).toBe('Warning: server.trusted_proxies is empty');
  });

  it('extracts trailing JSON metadata and summarizes common fields first', () => {
    const line = parseLogLine(
      '2026-06-17T06:31:42.585+0800 INFO middleware/logger.go:61 http request completed {"service":"sub2api","env":"production","component":"http","request_id":"abc","duration":"12ms","method":"GET","path":"/health","status":200}',
      29,
    );
    const metadata = line.metadata as ParsedLogMetadata;
    const summary = summarizeMetadata(metadata, 3);

    expect(line.message).toBe('http request completed');
    expect(metadata.service).toBe('sub2api');
    expect(metadata.status).toBe(200);
    expect(summary.tags.map(([key]) => key)).toEqual(['request_id', 'path', 'method']);
    expect(summary.hiddenCount).toBe(5);
  });

  it('parses a full JSON structured log into normalized display fields', () => {
    const line = parseContainerLogLine(
      '{"time":"2026-06-17T08:30:27.324+0800","level":"INFO","msg":"http request completed","path":"/api/v1/auth/me","caller":"middleware/logger.go:61"}',
    );

    expect(line.format).toBe('json');
    expect(line.time).toBe('2026-06-17T08:30:27.324+0800');
    expect(line.level).toBe('INFO');
    expect(line.source).toBe('middleware/logger.go:61');
    expect(line.message).toBe('http request completed');
    expect(line.fields.path).toBe('/api/v1/auth/me');
    expect(line.display.subtitleParts).toEqual(['2026-06-17T08:30:27.324+0800', 'middleware/logger.go:61']);
  });

  it('prioritizes HTTP structured metadata in important fields', () => {
    const line = parseContainerLogLine(
      '2026-06-17T06:31:42.585+0800 INFO middleware/logger.go:61 http request completed {"service":"sub2api","env":"production","component":"http","request_id":"abc","latency_ms":12,"method":"GET","path":"/health","status_code":200}',
    );

    expect(line.importantFields.map((field) => field.key)).toEqual([
      'request_id',
      'path',
      'method',
      'status_code',
      'latency_ms',
    ]);
  });

  it('parses logfmt key value lines with quoted values', () => {
    const line = parseContainerLogLine(
      'time=2026-06-16T22:27:57.106Z level=INFO msg="server run start" error="context canceled"',
    );

    expect(line.format).toBe('logfmt');
    expect(line.message).toBe('server run start');
    expect(line.time).toBe('2026-06-16T22:27:57.106Z');
    expect(line.level).toBe('INFO');
    expect(line.fields.msg).toBe('server run start');
    expect(line.fields.error).toBe('context canceled');
    expect(line.importantFields.map((field) => `${field.key}=${field.value}`)).toEqual([
      'time=2026-06-16T22:27:57.106Z',
      'level=INFO',
      'msg=server run start',
      'error=context canceled',
    ]);
    expect(line.display.subtitleParts).toEqual(['2026-06-16T22:27:57.106Z']);
  });

  it('falls back to plain text without empty metadata', () => {
    const line = parseLogLine('GitHub MCP Server running on stdio', 1);

    expect(line.parsed.format).toBe('plain');
    expect(line.level).toBe('LOG');
    expect(line.message).toBe('GitHub MCP Server running on stdio');
    expect(line.metadata).toBeNull();
    expect(line.parsed.importantFields).toEqual([]);
    expect(line.parsed.display.subtitleParts).toEqual([]);
  });

  it('detects standalone severities in plain text fallback lines', () => {
    const errorLine = parseLogLine('ERROR failed to connect upstream service', 1);
    const warnLine = parseLogLine('warning cache is nearly full', 2);
    const infoLine = parseLogLine('info background worker started', 3);
    const unknownLine = parseLogLine('unknown host name after DNS lookup', 4);

    expect(errorLine.parsed.format).toBe('plain');
    expect(errorLine.level).toBe('ERROR');
    expect(warnLine.level).toBe('WARN');
    expect(infoLine.level).toBe('INFO');
    expect(unknownLine.level).toBe('LOG');
  });

  it('keeps stack trace-like lines from gaining false time or source fields', () => {
    const line = parseLogLine('github.com/xxx/service.(*PricingService).refresh', 1);

    expect(line.parsed.format).toBe('stack');
    expect(line.message).toBe('github.com/xxx/service.(*PricingService).refresh');
    expect(line.timestamp).toBe('');
    expect(line.source).toBe('');
  });

  it('maps common level aliases into stable semantics', () => {
    expect(parseContainerLogLine('level=warning msg="warned"').level).toBe('WARN');
    expect(parseContainerLogLine('level=WARN msg="warned"').level).toBe('WARN');
    expect(parseContainerLogLine('level=ERROR msg="failed"').level).toBe('ERROR');
    expect(parseContainerLogLine('level=FATAL msg="failed"').level).toBe('FATAL');
    expect(parseContainerLogLine('level=debug msg="debugged"').level).toBe('DEBUG');
    expect(parseContainerLogLine('level=log msg="ordinary"').level).toBe('LOG');
  });

  it('folds repeated low-signal metadata out of the default tag summary', () => {
    const line = parseLogLine(
      '2026-06-17T06:43:20.498+0800 INFO stdlog service/timing_wheel.go:37 flushed {"service":"sub2api","env":"production","legacy_stdlog":true,"request_id":"req-1","component":"scheduler"}',
      171,
    );
    const summary = summarizeMetadata(line.metadata, 3);

    expect(summary.tags.map(([key]) => key)).toEqual(['request_id']);
    expect(summary.hiddenCount).toBe(4);
  });

  it('falls back to raw text when trailing JSON parsing fails', () => {
    const raw =
      '2026-06-17T06:31:40.491+0800 INFO service/pricing_service.go:145 [Pricing] Service initialized {"service":';
    const line = parseLogLine(raw, 1);

    expect(line.metadata).toBeNull();
    expect(line.raw).toBe(raw);
    expect(line.message).toContain('{"service":');
  });

  it('highlights search keywords without changing log order', () => {
    const line = buildDisplayLogLine(
      parseLogLine(
        '2026-06-17T06:31:42.585+0800 INFO middleware/logger.go:61 http request completed {"request_id":"abc"}',
        2,
      ),
      'request',
    );

    expect(line.lineNo).toBe(2);
    expect(line.searchMatchCount).toBe(2);
    expect(line.messageTokens.some((token) => token.type === 'keyword' && token.text === 'request')).toBe(true);
  });
});
