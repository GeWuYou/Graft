package cachex

import "golang.org/x/sync/singleflight"

// Group 合并同一缓存键的并发未命中加载，避免重复访问外部数据源。
type Group struct {
	group singleflight.Group
}

// NewGroup 创建一个按缓存键去重并发加载的分组。
func NewGroup() *Group {
	return &Group{}
}

// Do 为指定键执行一次 fn，并向并发调用方共享结果；Group 为空时退化为直接执行。
func (g *Group) Do(key string, fn func() (Item, error)) (Item, error, bool) {
	if g == nil {
		item, err := fn()
		return item, err, false
	}

	value, err, shared := g.group.Do(key, func() (any, error) {
		return fn()
	})
	if err != nil {
		return Item{}, err, shared
	}

	item, ok := value.(Item)
	if !ok {
		return Item{}, nil, shared
	}

	return item.Clone(), nil, shared
}
