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
)

const (
	readLimitBytes  = 1024 * 1024
	writeWait       = 10 * time.Second
	pongWait        = 60 * time.Second
	pingPeriod      = (pongWait * 9) / 10
	closeGrace      = 2 * time.Second
)

type ClientMessageType string
type ServerMessageType string

const (
	ClientMessageInput  ClientMessageType = "input"
	ClientMessageResize ClientMessageType = "resize"
	ClientMessagePing   ClientMessageType = "ping"

	ServerMessageOutput ServerMessageType = "output"
	ServerMessageStatus ServerMessageType = "status"
	ServerMessageError  ServerMessageType = "error"
	ServerMessagePong   ServerMessageType = "pong"
)

type ClientMessage struct {
	Type ClientMessageType `json:"type"`
	Data string            `json:"data,omitempty"`
	Cols int               `json:"cols,omitempty"`
	Rows int               `json:"rows,omitempty"`
}

type ServerMessage struct {
	Type    ServerMessageType `json:"type"`
	Data    string            `json:"data,omitempty"`
	State   string            `json:"state,omitempty"`
	Message string            `json:"message,omitempty"`
}

// Bridge binds one websocket connection to one terminal session.
type Bridge struct {
	conn   *websocket.Conn
	session Session
	once   sync.Once
	closed chan struct{}
}

func NewBridge(conn *websocket.Conn, session Session) *Bridge {
	return &Bridge{conn: conn, session: session, closed: make(chan struct{})}
}

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

	errCh := make(chan error, 3)
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
		switch message.Type {
		case ClientMessageInput:
			if err := b.session.Write(ctx, []byte(message.Data)); err != nil {
				errCh <- err
				return
			}
		case ClientMessageResize:
			if err := b.session.Resize(ctx, Size{
				Cols: positiveUint(message.Cols),
				Rows: positiveUint(message.Rows),
			}); err != nil {
				errCh <- err
				return
			}
		case ClientMessagePing:
			if err := b.writeJSON(ServerMessage{Type: ServerMessagePong}); err != nil {
				errCh <- err
				return
			}
		default:
			if err := b.writeJSON(ServerMessage{Type: ServerMessageError, Message: "unsupported terminal control message"}); err != nil {
				errCh <- err
				return
			}
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

func positiveUint(value int) uint {
	if value <= 0 {
		return 0
	}
	return uint(value)
}
