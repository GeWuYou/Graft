// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package terminal

import (
	"context"
	"encoding/json"
	"errors"
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

// ClientMessageType identifies one client-to-server websocket terminal message kind.
type ClientMessageType string

// ServerMessageType identifies one server-to-client websocket terminal message kind.
type ServerMessageType string

const (
	// ClientMessageInput streams terminal stdin bytes.
	ClientMessageInput ClientMessageType = "input"
	// ClientMessageResize updates the terminal geometry.
	ClientMessageResize ClientMessageType = "resize"
	// ClientMessagePing requests an application-level pong frame.
	ClientMessagePing ClientMessageType = "ping"

	// ServerMessageOutput streams terminal stdout/stderr bytes.
	ServerMessageOutput ServerMessageType = "output"
	// ServerMessageStatus reports high-level bridge state changes.
	ServerMessageStatus ServerMessageType = "status"
	// ServerMessageError reports a terminal or protocol error.
	ServerMessageError ServerMessageType = "error"
	// ServerMessagePong acknowledges a client ping request.
	ServerMessagePong ServerMessageType = "pong"
)

// ClientMessage is the JSON control envelope accepted from the websocket client.
type ClientMessage struct {
	Type ClientMessageType `json:"type"`
	Data string            `json:"data,omitempty"`
	Cols int               `json:"cols,omitempty"`
	Rows int               `json:"rows,omitempty"`
}

// ServerMessage is the JSON control envelope emitted to the websocket client.
type ServerMessage struct {
	Type       ServerMessageType `json:"type"`
	Data       string            `json:"data,omitempty"`
	State      string            `json:"state,omitempty"`
	Message    string            `json:"message,omitempty"`
	MessageKey string            `json:"messageKey,omitempty"`
}

// Bridge binds one websocket connection to one terminal session.
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

// Run starts the websocket bridge and blocks until the bridge or caller context finishes.
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
	_ = b.conn.SetReadDeadline(time.Now().Add(pongWait))
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
			_ = b.writeJSON(ServerMessage{Type: ServerMessageError, Message: err.Error()})
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
	_ = b.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return b.conn.WriteMessage(messageType, payload)
}

func (b *Bridge) close(ctx context.Context) {
	b.once.Do(func() {
		close(b.closed)
		closeCtx, cancel := context.WithTimeout(ctx, closeGrace)
		defer cancel()
		_ = b.session.Close(closeCtx)
		_ = b.conn.Close()
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

// positiveUint converts an int to uint, returning the value if positive and zero if less than or equal to zero.
func positiveUint(value int) uint {
	if value <= 0 {
		return 0
	}
	return uint(value)
}
