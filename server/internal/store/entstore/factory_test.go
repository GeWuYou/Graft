package entstore

import (
	"strings"
	"testing"
)

// TestNewFactoryRejectsNilClient 验证仓储工厂会在缺失 Ent 客户端时显式返回错误。
func TestNewFactoryRejectsNilClient(t *testing.T) {
	factory, err := NewFactory(nil)
	if err == nil {
		t.Fatal("expected error for nil ent client")
	}
	if factory != nil {
		t.Fatalf("expected nil factory, got %#v", factory)
	}
	if !strings.Contains(err.Error(), "non-nil ent client") {
		t.Fatalf("expected nil client error, got %v", err)
	}
}
