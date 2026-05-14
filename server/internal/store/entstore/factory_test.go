package entstore

import "testing"

// TestNewFactoryRejectsNilClient 验证仓储工厂会在缺失 Ent 客户端时快速失败。
func TestNewFactoryRejectsNilClient(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for nil ent client")
		}
	}()

	_ = NewFactory(nil)
}
