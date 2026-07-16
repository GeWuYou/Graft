package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	containercontract "graft/server/modules/container/contract"
)

const (
	readLimitBytes  = 1024 * 1024
	writeWait       = 10 * time.Second
	pongWait        = 60 * time.Second
	pingPeriodRatio = 9
	pingPeriodBase  = 10
	pingPeriod      = (pongWait * pingPeriodRatio) / pingPeriodBase
	closeGrace      = 2 * time.Second
	bridgeErrBuffer = 3
)

// ClientMessageType 标识客户端发往服务端的 WebSocket 终端消息类型。
type ClientMessageType string

// ServerMessageType 标识服务端发往客户端的 WebSocket 终端消息类型。
type ServerMessageType string

const (
	// ClientMessageInput 传输终端标准输入字节。
	ClientMessageInput ClientMessageType = "input"
	// ClientMessageResize 更新终端几何尺寸。
	ClientMessageResize ClientMessageType = "resize"
	// ClientMessagePing 请求应用层 pong 响应。
	ClientMessagePing ClientMessageType = "ping"

	// ServerMessageOutput 传输终端标准输出或标准错误字节。
	ServerMessageOutput ServerMessageType = "output"
	// ServerMessageStatus 报告桥接状态变化。
	ServerMessageStatus ServerMessageType = "status"
	// ServerMessageError 报告终端或协议错误。
	ServerMessageError ServerMessageType = "error"
	// ServerMessagePong 确认客户端的 ping 请求。
	ServerMessagePong ServerMessageType = "pong"
)

// ClientMessage 是 WebSocket 客户端发送的 JSON 控制信封。
type ClientMessage struct {
	Type ClientMessageType `json:"type"`
	Data string            `json:"data,omitempty"`
	Cols int               `json:"cols,omitempty"`
	Rows int               `json:"rows,omitempty"`
}

// ServerMessage 是发送给 WebSocket 客户端的 JSON 控制信封；错误同时保留稳定 message key 供前端本地化。
type ServerMessage struct {
	Type       ServerMessageType `json:"type"`
	Data       string            `json:"data,omitempty"`
	State      string            `json:"state,omitempty"`
	Message    string            `json:"message,omitempty"`
	MessageKey string            `json:"messageKey,omitempty"`
}

// Bridge 将一个 WebSocket 连接绑定到一个终端会话，并串行化所有写操作以满足 Gorilla WebSocket 的并发约束。
type Bridge struct {
	conn    *websocket.Conn
	session Session
	once    sync.Once
	writeMu sync.Mutex
	closed  chan struct{}
}

// NewBridge 将一个 WebSocket 连接与一个终端会话绑定。
func NewBridge(conn *websocket.Conn, session Session) *Bridge {
	return &Bridge{conn: conn, session: session, closed: make(chan struct{})}
}

// Run 启动终端会话和读写/心跳协程，直到连接关闭、桥接报错或调用方上下文结束；退出时统一关闭两端资源。
func (b *Bridge) Run(ctx context.Context, initialSize Size) error {
	if b == nil || b.conn == nil || b.session == nil {
		return errors.New("terminal bridge is unavailable")
	}
	if err := b.session.Start(ctx, initialSize); err != nil {
		return err
	}
	defer b.close(context.Background())

	if err := b.writeJSON(ServerMessage{Type: ServerMessageStatus, State: "connected"}); err != nil {
		return err
	}

	errCh := make(chan error, bridgeErrBuffer)
	go b.readLoop(ctx, errCh)
	go b.writeLoop(ctx, errCh)
	go b.pingLoop(ctx, errCh)

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		if errors.Is(err, io.EOF) || websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			return nil
		}
		return err
	}
}

func (b *Bridge) readLoop(ctx context.Context, errCh chan<- error) {
	b.conn.SetReadLimit(readLimitBytes)
	if err := b.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		errCh <- fmt.Errorf("set websocket read deadline: %w", err)
		return
	}
	b.conn.SetPongHandler(func(string) error {
		return b.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		select {
		case <-ctx.Done():
			errCh <- nil
			return
		default:
		}

		var message ClientMessage
		if err := b.conn.ReadJSON(&message); err != nil {
			errCh <- err
			return
		}
		if err := b.handleClientMessage(ctx, message); err != nil {
			errCh <- err
			return
		}
	}
}

func (b *Bridge) writeLoop(ctx context.Context, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			errCh <- nil
			return
		case output, ok := <-b.session.Output():
			if !ok {
				errCh <- io.EOF
				return
			}
			if err := b.writeJSON(ServerMessage{Type: ServerMessageOutput, Data: string(output)}); err != nil {
				errCh <- err
				return
			}
		case err, ok := <-b.session.Errors():
			if !ok {
				errCh <- nil
				return
			}
			if err == nil {
				errCh <- nil
				return
			}
			if writeErr := b.writeJSON(ServerMessage{Type: ServerMessageError, Message: err.Error()}); writeErr != nil {
				errCh <- errors.Join(err, writeErr)
				return
			}
			errCh <- err
			return
		}
	}
}

func (b *Bridge) pingLoop(ctx context.Context, errCh chan<- error) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := b.writeControl(websocket.PingMessage, nil); err != nil {
				errCh <- err
				return
			}
		}
	}
}

func (b *Bridge) writeJSON(message ServerMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return b.writeControl(websocket.TextMessage, payload)
}

func (b *Bridge) writeControl(messageType int, payload []byte) error {
	b.writeMu.Lock()
	defer b.writeMu.Unlock()
	if err := b.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return fmt.Errorf("set websocket write deadline: %w", err)
	}
	return b.conn.WriteMessage(messageType, payload)
}

func (b *Bridge) close(ctx context.Context) {
	b.once.Do(func() {
		close(b.closed)
		closeCtx, cancel := context.WithTimeout(ctx, closeGrace)
		defer cancel()
		_ = errors.Join(b.session.Close(closeCtx), b.conn.Close())
	})
}

func (b *Bridge) handleClientMessage(ctx context.Context, message ClientMessage) error {
	switch message.Type {
	case ClientMessageInput:
		return b.session.Write(ctx, []byte(message.Data))
	case ClientMessageResize:
		return b.session.Resize(ctx, Size{
			Cols: positiveUint(message.Cols),
			Rows: positiveUint(message.Rows),
		})
	case ClientMessagePing:
		return b.writeJSON(ServerMessage{Type: ServerMessagePong})
	default:
		return b.writeJSON(ServerMessage{
			Type:       ServerMessageError,
			Message:    "unsupported terminal control message",
			MessageKey: containercontract.ContainerShellUnsupportedControlMessage.String(),
		})
	}
}

// positiveUint 将非正数归零，避免客户端提供的负尺寸转换为超大的无符号终端尺寸。
func positiveUint(value int) uint {
	if value <= 0 {
		return 0
	}
	return uint(value)
}
