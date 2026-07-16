package cachex

import (
	"context"
	"errors"
)

// ErrLoaderRequired 表示读穿透缓存调用没有提供未命中加载器。
var ErrLoaderRequired = errors.New("cache loader is required")

// Loader 在缓存未命中时从外部数据源构建一项缓存内容。
type Loader func(context.Context) (Item, error)
