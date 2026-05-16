package container

import (
	"sync"
	"sync/atomic"
	"testing"
)

type testService struct {
	name string
}

// TestResolveBuildsSingletonOnceForConcurrentCalls 验证并发调用方会共享同一次
// 构建中的 provider 调用，并最终拿到同一个单例实例。
//
// 测试使用 started/release 双通道显式卡住 provider，确保并发 goroutine
// 真正落在“构建进行中”的窗口里，而不是偶然命中已缓存结果。
func TestResolveBuildsSingletonOnceForConcurrentCalls(t *testing.T) {
	container := New()
	started := make(chan struct{})
	release := make(chan struct{})
	var startedOnce sync.Once
	var providerCalls atomic.Int32

	if err := container.RegisterSingleton((*testService)(nil), func(_ Resolver) (any, error) {
		providerCalls.Add(1)
		startedOnce.Do(func() {
			close(started)
		})
		<-release
		return &testService{name: "shared"}, nil
	}); err != nil {
		t.Fatalf("register singleton: %v", err)
	}

	const goroutineCount = 8
	results := make([]any, goroutineCount)
	errs := make([]error, goroutineCount)
	begin := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutineCount)

	for i := 0; i < goroutineCount; i++ {
		go func(index int) {
			defer wg.Done()
			<-begin
			results[index], errs[index] = container.Resolve((*testService)(nil))
		}(i)
	}

	close(begin)
	<-started
	close(release)
	wg.Wait()

	if got := providerCalls.Load(); got != 1 {
		t.Fatalf("expected provider to be called once, got %d", got)
	}

	first, ok := results[0].(*testService)
	if !ok {
		t.Fatalf("expected first result to be *testService, got %T", results[0])
	}

	for i := 0; i < goroutineCount; i++ {
		if errs[i] != nil {
			t.Fatalf("resolve %d returned error: %v", i, errs[i])
		}

		resolved, ok := results[i].(*testService)
		if !ok {
			t.Fatalf("expected result %d to be *testService, got %T", i, results[i])
		}
		if resolved != first {
			t.Fatalf("expected result %d to reuse the cached singleton", i)
		}
	}
}
