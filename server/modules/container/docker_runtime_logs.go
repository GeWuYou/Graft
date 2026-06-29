package container

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
)

// readDockerLogEntries 读取 Docker 日志并将其转换为日志条目，按指定尾部条数保留最近的记录。
func readDockerLogEntries(ctx context.Context, reader io.Reader, tail int, timestamps bool) ([]LogEntry, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	limit := tail
	if limit > defaultContainerLogsMaxTail {
		limit = defaultContainerLogsMaxTail
	}
	truncated := false
	if limit <= 0 {
		err := streamDockerLogLines(ctx, reader, timestamps, func(_ LogChunk) error {
			truncated = true
			return nil
		})
		if err != nil {
			return nil, false, err
		}
		return nil, truncated, nil
	}

	var buffer []LogEntry
	head := 0
	err := streamDockerLogLines(ctx, reader, timestamps, func(chunk LogChunk) error {
		entry := logEntryFromChunk(chunk)
		if len(buffer) < limit {
			buffer = append(buffer, entry)
			return nil
		}
		truncated = true
		buffer[head] = entry
		head = (head + 1) % limit
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return materializeBoundedLogEntries(buffer, len(buffer), limit, head), truncated, nil
}

func materializeBoundedLogEntries(buffer []LogEntry, count int, limit int, head int) []LogEntry {
	if count == 0 {
		return nil
	}
	if count < limit {
		entries := make([]LogEntry, count)
		copy(entries, buffer[:count])
		return entries
	}

	entries := make([]LogEntry, limit)
	for index := range entries {
		entries[index] = buffer[(head+index)%limit]
	}
	return entries
}

// 当上下文取消、回调返回错误或读取过程发生错误时返回相应错误。
func streamDockerLogLines(ctx context.Context, reader io.Reader, timestamps bool, emit func(LogChunk) error) error {
	if reader == nil {
		return nil
	}
	chunkEmitter := newDockerLogChunkEmitter(ctx, timestamps, emit)
	_, err := stdcopy.StdCopy(chunkEmitter.stdoutWriter(), chunkEmitter.stderrWriter(), reader)
	if flushErr := chunkEmitter.flush(); flushErr != nil {
		return flushErr
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

type dockerLogChunkEmitter struct {
	ctx        context.Context
	timestamps bool
	emit       func(LogChunk) error
	err        error
	stdout     dockerLogStreamWriter
	stderr     dockerLogStreamWriter
}

type dockerLogStreamWriter struct {
	parent *dockerLogChunkEmitter
	stream string
	buffer []byte
}

func newDockerLogChunkEmitter(ctx context.Context, timestamps bool, emit func(LogChunk) error) *dockerLogChunkEmitter {
	emitter := &dockerLogChunkEmitter{
		ctx:        ctx,
		timestamps: timestamps,
		emit:       emit,
	}
	emitter.stdout = dockerLogStreamWriter{parent: emitter, stream: "stdout", buffer: make([]byte, 0, dockerLogScannerInitSize)}
	emitter.stderr = dockerLogStreamWriter{parent: emitter, stream: "stderr", buffer: make([]byte, 0, dockerLogScannerInitSize)}
	return emitter
}

func (e *dockerLogChunkEmitter) stdoutWriter() io.Writer {
	return &e.stdout
}

func (e *dockerLogChunkEmitter) stderrWriter() io.Writer {
	return &e.stderr
}

func (e *dockerLogChunkEmitter) emitChunk(stream string, line string) error {
	if e.err != nil {
		return e.err
	}
	select {
	case <-e.ctx.Done():
		e.err = e.ctx.Err()
		return e.err
	default:
	}
	chunk := LogChunk{
		Line:   line,
		Stream: stream,
	}
	if e.timestamps {
		chunk.Timestamp, chunk.Line = parseDockerLogChunkTimestamp(line)
	}
	if err := e.emit(chunk); err != nil {
		e.err = err
		return err
	}
	return nil
}

// logEntryFromChunk 将日志块转换为日志条目，并将发生时间规范为 UTC。
func logEntryFromChunk(chunk LogChunk) LogEntry {
	occurredAt := chunk.Timestamp.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return LogEntry{
		Line:       chunk.Line,
		Stream:     strings.TrimSpace(chunk.Stream),
		OccurredAt: occurredAt,
	}
}

func (e *dockerLogChunkEmitter) flush() error {
	if err := e.stdout.flushRemainder(); err != nil {
		return err
	}
	if err := e.stderr.flushRemainder(); err != nil {
		return err
	}
	return e.err
}

func (w *dockerLogStreamWriter) Write(p []byte) (int, error) {
	if w.parent.err != nil {
		return 0, w.parent.err
	}
	w.buffer = append(w.buffer, p...)
	for {
		index := bytes.IndexByte(w.buffer, '\n')
		if index < 0 {
			if len(w.buffer) > dockerLogScannerMaxSize {
				w.parent.err = bufio.ErrTooLong
				return 0, w.parent.err
			}
			return len(p), nil
		}
		line := string(w.buffer[:index])
		w.buffer = w.buffer[index+1:]
		if err := w.parent.emitChunk(w.stream, line); err != nil {
			return 0, err
		}
	}
}

func (w *dockerLogStreamWriter) flushRemainder() error {
	if len(w.buffer) == 0 {
		return w.parent.err
	}
	if len(w.buffer) > dockerLogScannerMaxSize {
		w.parent.err = bufio.ErrTooLong
		return w.parent.err
	}
	line := string(w.buffer)
	w.buffer = nil
	return w.parent.emitChunk(w.stream, line)
}

func parseDockerLogChunkTimestamp(line string) (time.Time, string) {
	rawTimestamp, rest, ok := strings.Cut(line, " ")
	if !ok {
		return time.Time{}, line
	}
	timestamp, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(rawTimestamp))
	if err != nil {
		return time.Time{}, line
	}
	return timestamp.UTC(), rest
}
