export type TerminalConnectionState = 'idle' | 'connecting' | 'connected' | 'disconnected' | 'error';

export type TerminalLifecycleCloseReason =
  | 'manual_disconnect'
  | 'remote_close'
  | 'component_unmount'
  | 'connect_error'
  | 'session_error';

export interface TerminalResizePayload {
  cols: number;
  rows: number;
}

export interface TerminalSessionOpenResult {
  url: string;
  protocols?: string[];
  meta?: Record<string, unknown>;
}

export interface TerminalSessionConnectorContext {
  cols: number;
  rows: number;
}

export interface TerminalSessionConnector {
  open(context: TerminalSessionConnectorContext): Promise<TerminalSessionOpenResult>;
}

export interface TerminalStatusMessage {
  type: 'status';
  state: string;
}

export interface TerminalOutputMessage {
  type: 'output';
  data: string;
}

export interface TerminalErrorMessage {
  type: 'error';
  message: string;
  messageKey?: string;
}

export interface TerminalPongMessage {
  type: 'pong';
}

export type TerminalServerMessage =
  | TerminalStatusMessage
  | TerminalOutputMessage
  | TerminalErrorMessage
  | TerminalPongMessage;

export type TerminalClientMessage =
  | { type: 'input'; data: string }
  | { type: 'resize'; cols: number; rows: number }
  | { type: 'ping' };
