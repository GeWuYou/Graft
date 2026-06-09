// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"graft/server/internal/config"
)

// TestNewUsesConfiguredLogLevel 验证日志预留口子至少会把配置中的级别装配
// 到统一 logger 上，避免 runtime 与模块读取到不同的日志阈值。
func TestNewUsesConfiguredLogLevel(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name: "graft",
			Env:  "test",
		},
		Log: config.LogConfig{
			Level: "debug",
		},
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	t.Cleanup(func() {
		_ = Close(logger)
	})

	if !logger.Core().Enabled(zap.DebugLevel) {
		t.Fatal("expected debug level to be enabled")
	}
}

// TestNewRejectsInvalidLogLevel 验证非法日志级别会在 runtime 装配前直接失败。
func TestNewRejectsInvalidLogLevel(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name: "graft",
			Env:  "test",
		},
		Log: config.LogConfig{
			Level: "definitely-not-a-level",
		},
	}

	if _, err := New(cfg); err == nil {
		t.Fatal("expected invalid log level error")
	}
}

func TestBuildZapConfigUsesColorOnlyForConsole(t *testing.T) {
	tests := []struct {
		name      string
		appEnv    string
		format    config.LogFormat
		color     config.LogColor
		wantColor bool
		wantJSON  bool
	}{
		{name: "local auto console color", appEnv: "local", format: config.LogFormatAuto, color: config.LogColorAuto, wantColor: true},
		{name: "test auto console color", appEnv: "test", format: config.LogFormatAuto, color: config.LogColorAuto, wantColor: true},
		{name: "console never plain", appEnv: "local", format: config.LogFormatConsole, color: config.LogColorNever},
		{name: "production auto json", appEnv: "production", format: config.LogFormatAuto, color: config.LogColorAlways, wantJSON: true},
		{name: "staging auto json", appEnv: "staging", format: config.LogFormatAuto, color: config.LogColorAuto, wantJSON: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var buffer bytes.Buffer
			zapConfig := buildZapConfig(testCase.appEnv, testCase.format, testCase.color)
			encoder := zapcore.NewConsoleEncoder(zapConfig.EncoderConfig)
			if zapConfig.Encoding == string(config.LogFormatJSON) {
				encoder = zapcore.NewJSONEncoder(zapConfig.EncoderConfig)
			}
			core := zapcore.NewCore(
				encoder,
				zapcore.AddSync(&buffer),
				zapcore.DebugLevel,
			)
			logger := zap.New(core)
			logger.Info("hello")

			output := buffer.String()
			hasColor := strings.Contains(output, "\x1b[")
			if hasColor != testCase.wantColor {
				t.Fatalf("expected color=%v, got output %q", testCase.wantColor, output)
			}
			if testCase.wantJSON {
				var payload map[string]any
				if err := json.Unmarshal(buffer.Bytes(), &payload); err != nil {
					t.Fatalf("expected parseable JSON log, got %q: %v", output, err)
				}
				if payload["msg"] != "hello" {
					t.Fatalf("expected JSON message field, got %#v", payload)
				}
			}
		})
	}
}
