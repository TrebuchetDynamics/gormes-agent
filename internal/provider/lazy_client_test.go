package provider

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// countingFactory wraps a real factory and counts how many times it is called.
// Only the factory call is counted, not individual method calls on the client.
type countingFactory struct {
	inner   func() (llm.Client, error)
	calls   atomic.Int32
	clients []llm.Client // tracks constructed clients for identity checks
}

func (f *countingFactory) New() (llm.Client, error) {
	f.calls.Add(1)
	c, err := f.inner()
	if err != nil {
		return nil, err
	}
	f.clients = append(f.clients, c)
	return c, nil
}

func fakeClient(name string) llm.Client {
	return &stubClient{name: name}
}

type stubClient struct {
	name string
}

func (s *stubClient) OpenStream(_ context.Context, _ llm.ChatRequest) (llm.Stream, error) {
	return nil, nil
}
func (s *stubClient) OpenRunEvents(_ context.Context, _ string) (llm.RunEventStream, error) {
	return nil, nil
}
func (s *stubClient) Health(_ context.Context) error { return nil }

// TestProviderClientLazyInit_DoesNotConstructUnselectedProvider proves that
// only the selected provider is constructed; unselected providers never
// invoke their factory.
func TestProviderClientLazyInit_DoesNotConstructUnselectedProvider(t *testing.T) {
	anthropicCalls := &countingFactory{inner: func() (llm.Client, error) { return fakeClient("anthropic"), nil }}
	openaiCalls := &countingFactory{inner: func() (llm.Client, error) { return fakeClient("openai"), nil }}
	bedrockCalls := &countingFactory{inner: func() (llm.Client, error) { return fakeClient("bedrock"), nil }}
	firecrawlCalls := &countingFactory{inner: func() (llm.Client, error) { return fakeClient("firecrawl"), nil }}

	pool := NewClientPool()
	pool.Register("anthropic", anthropicCalls.New)
	pool.Register("openai", openaiCalls.New)
	pool.Register("bedrock", bedrockCalls.New)
	pool.Register("firecrawl", firecrawlCalls.New)

	// Select anthropic — only anthropic should be constructed.
	client, err := pool.Get("anthropic")
	if err != nil {
		t.Fatalf("Get(anthropic): %v", err)
	}
	_ = client

	if anthropicCalls.calls.Load() != 1 {
		t.Errorf("anthropic factory calls = %d, want 1", anthropicCalls.calls.Load())
	}
	if openaiCalls.calls.Load() != 0 {
		t.Errorf("openai factory calls = %d, want 0 (unselected)", openaiCalls.calls.Load())
	}
	if bedrockCalls.calls.Load() != 0 {
		t.Errorf("bedrock factory calls = %d, want 0 (unselected)", bedrockCalls.calls.Load())
	}
	if firecrawlCalls.calls.Load() != 0 {
		t.Errorf("firecrawl factory calls = %d, want 0 (unselected)", firecrawlCalls.calls.Load())
	}
}

// TestProviderClientLazyInit_ConstructedOnce proves the selected provider
// client is constructed exactly once and the same instance is returned on
// repeated calls.
func TestProviderClientLazyInit_ConstructedOnce(t *testing.T) {
	var constructed atomic.Int32
	factory := &countingFactory{inner: func() (llm.Client, error) {
		constructed.Add(1)
		return fakeClient("test-provider"), nil
	}}

	pool := NewClientPool()
	pool.Register("test", factory.New)

	// First call — should construct.
	c1, err := pool.Get("test")
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if factory.calls.Load() != 1 {
		t.Errorf("factory calls after first Get = %d, want 1", factory.calls.Load())
	}

	// Second call — should reuse.
	c2, err := pool.Get("test")
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if factory.calls.Load() != 1 {
		t.Errorf("factory calls after second Get = %d, want 1 (reused)", factory.calls.Load())
	}

	// Same instance.
	if c1 != c2 {
		t.Error("c1 != c2, lazy client must return the same instance")
	}
}

// TestProviderClientLazyInit_ResetForTesting proves that Reset clears
// constructed clients so subsequent Get calls re-invoke the factory.
func TestProviderClientLazyInit_ResetForTesting(t *testing.T) {
	factory := &countingFactory{inner: func() (llm.Client, error) { return fakeClient("test"), nil }}

	pool := NewClientPool()
	pool.Register("test", factory.New)

	// First Get.
	c1, err := pool.Get("test")
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if factory.calls.Load() != 1 {
		t.Errorf("factory calls after first Get = %d, want 1", factory.calls.Load())
	}

	// Reset.
	pool.Reset()

	// Second Get — should re-construct.
	c2, err := pool.Get("test")
	if err != nil {
		t.Fatalf("second Get after reset: %v", err)
	}
	if factory.calls.Load() != 2 {
		t.Errorf("factory calls after reset + second Get = %d, want 2", factory.calls.Load())
	}

	// Different instances after reset.
	if c1 == c2 {
		t.Error("c1 == c2 after reset, want different instances")
	}
}

// TestProviderClientLazyInit_UnknownProvider returns an error.
func TestProviderClientLazyInit_UnknownProvider(t *testing.T) {
	pool := NewClientPool()
	_, err := pool.Get("nonexistent")
	if err == nil {
		t.Error("Get(nonexistent) = nil error, want error for unknown provider")
	}
}

// TestProviderClientLazyInit_FactoryError propagates factory errors.
func TestProviderClientLazyInit_FactoryError(t *testing.T) {
	pool := NewClientPool()
	pool.Register("broken", func() (llm.Client, error) {
		return nil, llm.ErrProviderUnavailable
	})

	_, err := pool.Get("broken")
	if err == nil {
		t.Error("Get(broken) = nil error, want factory error")
	}
}

func TestProviderClientPoolZeroValueRegisterInitializesMaps(t *testing.T) {
	var pool ClientPool
	pool.Register("zero", func() (llm.Client, error) {
		return fakeClient("zero"), nil
	})

	client, err := pool.Get("zero")
	if err != nil {
		t.Fatalf("Get(zero): %v", err)
	}
	if client == nil {
		t.Fatal("Get(zero) = nil client")
	}
}

func TestProviderClientPoolNilFactoryReturnsError(t *testing.T) {
	pool := NewClientPool()
	pool.Register("nil-factory", nil)

	_, err := pool.Get("nil-factory")
	if err == nil {
		t.Fatal("Get(nil-factory) = nil error, want factory error")
	}
	if !strings.Contains(err.Error(), "factory") {
		t.Fatalf("Get(nil-factory) error = %q, want factory context", err.Error())
	}
}

// TestProviderClientLazyInit_ConcurrentAccess proves the pool is safe for
// concurrent access.
func TestProviderClientLazyInit_ConcurrentAccess(t *testing.T) {
	factory := &countingFactory{inner: func() (llm.Client, error) { return fakeClient("concurrent"), nil }}

	pool := NewClientPool()
	pool.Register("concurrent", factory.New)

	const goroutines = 10
	done := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		go func() {
			c, err := pool.Get("concurrent")
			if err != nil {
				t.Errorf("Get(concurrent): %v", err)
			}
			if c == nil {
				t.Error("Get(concurrent) = nil client")
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	if factory.calls.Load() != 1 {
		t.Errorf("factory calls under concurrency = %d, want 1", factory.calls.Load())
	}
}
