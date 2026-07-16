package cachex

import (
	"errors"
	"time"

	"graft/server/internal/cachex/backend"
)

// Item 表示一项缓存载荷及其过期语义。
type Item struct {
	Value     []byte
	TTL       time.Duration
	ExpiresAt time.Time
}

// NewItem 创建带有载荷和 TTL 的缓存项，并复制载荷以隔离调用方后续修改。
func NewItem(value []byte, ttl time.Duration) Item {
	return Item{
		Value: cloneBytes(value),
		TTL:   ttl,
	}
}

// Clone 返回缓存项载荷和元数据的防御性副本。
func (i Item) Clone() Item {
	return Item{
		Value:     cloneBytes(i.Value),
		TTL:       i.TTL,
		ExpiresAt: i.ExpiresAt,
	}
}

// Validate 检查缓存项的过期元数据是否自洽。
func (i Item) Validate() error {
	if i.TTL < 0 {
		return errors.New("cache item ttl must be non-negative")
	}
	if len(i.Value) == 0 {
		return errors.New("cache item value is required")
	}

	return nil
}

// itemFromEntry 将后端项转换为缓存项，并防御性复制载荷。
func itemFromEntry(entry backend.Entry) Item {
	return Item{
		Value:     cloneBytes(entry.Value),
		ExpiresAt: entry.ExpiresAt,
	}
}

// entryFromItem 将缓存项转换为后端 Entry，复制载荷并保留绝对过期时间。
func entryFromItem(item Item) backend.Entry {
	return backend.Entry{
		Value:     cloneBytes(item.Value),
		ExpiresAt: item.ExpiresAt,
	}
}

// cloneBytes 返回字节切片的防御性副本；空切片返回 nil。
func cloneBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}

	cloned := make([]byte, len(value))
	copy(cloned, value)
	return cloned
}
