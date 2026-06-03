package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/middleware"
	"github.com/neogan74/konsul/internal/store"
)

// nopLogger discards every log record — keeps benchmark output clean.
type nopLogger struct{}

func (nopLogger) Debug(_ string, _ ...logger.Field)          {}
func (nopLogger) Info(_ string, _ ...logger.Field)           {}
func (nopLogger) Warn(_ string, _ ...logger.Field)           {}
func (nopLogger) Error(_ string, _ ...logger.Field)          {}
func (nopLogger) WithRequest(_ string) logger.Logger         { return nopLogger{} }
func (nopLogger) WithFields(_ ...logger.Field) logger.Logger { return nopLogger{} }

var quietLogger logger.Logger = nopLogger{}

func init() {
	// Suppress the global default (used by store internals that call logger.Info directly).
	logger.SetDefault(logger.New(zapcore.ErrorLevel, "json"))
	_ = zap.NewNop() // ensure zap is linked
}

// quietMiddleware injects a silent logger into each request context so handler
// log calls don't pollute benchmark output.
func quietMiddleware(c *fiber.Ctx) error {
	c.Locals(middleware.LoggerKey, quietLogger)
	return c.Next()
}

// benchKVApp returns a KVHandler + Fiber app with quiet logging for benchmarks.
func benchKVApp() (*KVHandler, *fiber.App) {
	kvStore := store.NewKVStore()
	handler := NewKVHandler(kvStore, nil)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(quietMiddleware)
	app.Get("/kv/:key", handler.Get)
	app.Put("/kv/:key", handler.Set)
	app.Delete("/kv/:key", handler.Delete)
	return handler, app
}

// benchServiceApp returns a ServiceHandler + Fiber app with quiet logging.
func benchServiceApp() (*ServiceHandler, *fiber.App) {
	svcStore := store.NewServiceStore()
	handler := NewServiceHandler(svcStore, nil)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(quietMiddleware)
	app.Put("/register", handler.Register)
	app.Get("/services/", handler.List)
	app.Get("/services/:name", handler.Get)
	app.Delete("/deregister/:name", handler.Deregister)
	app.Put("/heartbeat/:name", handler.Heartbeat)
	return handler, app
}

// benchBatchApp returns a BatchHandler + Fiber app with quiet logging.
func benchBatchApp() (*fiber.App, *BatchHandler) {
	kvStore := store.NewKVStore()
	svcStore := store.NewServiceStore()
	handler := NewBatchHandler(kvStore, svcStore, nil)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(quietMiddleware)
	app.Post("/batch/kv/get", handler.BatchKVGet)
	app.Post("/batch/kv/set", handler.BatchKVSet)
	app.Post("/batch/services/register", handler.BatchServiceRegister)
	return app, handler
}

// ── KV Handler Benchmarks ────────────────────────────────────────────────────

func BenchmarkKVHandler_Get_Hit(b *testing.B) {
	_, app := benchKVApp()
	// pre-populate
	putReq := httptest.NewRequest(http.MethodPut, "/kv/bench-key",
		bytes.NewBufferString(`{"value":"bench-value"}`))
	putReq.Header.Set("Content-Type", "application/json")
	app.Test(putReq) //nolint:errcheck // setup call, error irrelevant to benchmark

	req := httptest.NewRequest(http.MethodGet, "/kv/bench-key", http.NoBody)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkKVHandler_Get_Miss(b *testing.B) {
	_, app := benchKVApp()
	req := httptest.NewRequest(http.MethodGet, "/kv/nonexistent", http.NoBody)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkKVHandler_Set(b *testing.B) {
	_, app := benchKVApp()
	body := []byte(`{"value":"bench-value"}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPut, "/kv/bench-key", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkKVHandler_Delete(b *testing.B) {
	_, app := benchKVApp()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		put := httptest.NewRequest(http.MethodPut, "/kv/bench-key",
			bytes.NewBufferString(`{"value":"v"}`))
		put.Header.Set("Content-Type", "application/json")
		app.Test(put) //nolint:errcheck // setup call, error irrelevant to benchmark
		b.StartTimer()

		req := httptest.NewRequest(http.MethodDelete, "/kv/bench-key", http.NoBody)
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// ── Service Handler Benchmarks ───────────────────────────────────────────────

func BenchmarkServiceHandler_Register(b *testing.B) {
	_, app := benchServiceApp()
	svc := store.Service{Name: "bench-svc", Address: "127.0.0.1", Port: 8080}
	body, _ := json.Marshal(svc)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkServiceHandler_Get(b *testing.B) {
	_, app := benchServiceApp()
	svc := store.Service{Name: "bench-svc", Address: "127.0.0.1", Port: 8080}
	body, _ := json.Marshal(svc)
	reg := httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
	reg.Header.Set("Content-Type", "application/json")
	app.Test(reg) //nolint:errcheck // setup call, error irrelevant to benchmark

	req := httptest.NewRequest(http.MethodGet, "/services/bench-svc", http.NoBody)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkServiceHandler_List10(b *testing.B)   { benchmarkServiceHandlerList(b, 10) }
func BenchmarkServiceHandler_List100(b *testing.B)  { benchmarkServiceHandlerList(b, 100) }
func BenchmarkServiceHandler_List1000(b *testing.B) { benchmarkServiceHandlerList(b, 1000) }

func benchmarkServiceHandlerList(b *testing.B, n int) {
	_, app := benchServiceApp()
	for i := 0; i < n; i++ {
		svc := store.Service{Name: fmt.Sprintf("svc-%d", i), Address: "127.0.0.1", Port: 8080 + i}
		body, _ := json.Marshal(svc)
		reg := httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
		reg.Header.Set("Content-Type", "application/json")
		app.Test(reg) //nolint:errcheck // setup call, error irrelevant to benchmark
	}

	req := httptest.NewRequest(http.MethodGet, "/services/", http.NoBody)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkServiceHandler_Heartbeat(b *testing.B) {
	_, app := benchServiceApp()
	svc := store.Service{Name: "bench-svc", Address: "127.0.0.1", Port: 8080}
	body, _ := json.Marshal(svc)
	reg := httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
	reg.Header.Set("Content-Type", "application/json")
	app.Test(reg) //nolint:errcheck // setup call, error irrelevant to benchmark

	req := httptest.NewRequest(http.MethodPut, "/heartbeat/bench-svc", http.NoBody)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkServiceHandler_Deregister(b *testing.B) {
	_, app := benchServiceApp()
	body, _ := json.Marshal(store.Service{Name: "bench-svc", Address: "127.0.0.1", Port: 8080})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		reg := httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
		reg.Header.Set("Content-Type", "application/json")
		app.Test(reg) //nolint:errcheck // setup call, error irrelevant to benchmark
		b.StartTimer()

		req := httptest.NewRequest(http.MethodDelete, "/deregister/bench-svc", http.NoBody)
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// ── Batch Handler Benchmarks ─────────────────────────────────────────────────

func BenchmarkBatchKVSet10(b *testing.B)   { benchmarkBatchKVSet(b, 10) }
func BenchmarkBatchKVSet100(b *testing.B)  { benchmarkBatchKVSet(b, 100) }
func BenchmarkBatchKVSet1000(b *testing.B) { benchmarkBatchKVSet(b, 1000) }

func benchmarkBatchKVSet(b *testing.B, size int) {
	app, handler := benchBatchApp()
	_ = handler

	type batchSetRequest struct {
		Items map[string]string `json:"items"`
	}
	items := make(map[string]string, size)
	for i := 0; i < size; i++ {
		items[fmt.Sprintf("key/%d", i)] = fmt.Sprintf("value-%d", i)
	}
	body, _ := json.Marshal(batchSetRequest{Items: items})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/batch/kv/set", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkBatchKVGet10(b *testing.B)   { benchmarkBatchKVGet(b, 10) }
func BenchmarkBatchKVGet100(b *testing.B)  { benchmarkBatchKVGet(b, 100) }
func BenchmarkBatchKVGet1000(b *testing.B) { benchmarkBatchKVGet(b, 1000) }

func benchmarkBatchKVGet(b *testing.B, size int) {
	app, handler := benchBatchApp()

	// pre-populate
	keys := make([]string, size)
	for i := 0; i < size; i++ {
		k := fmt.Sprintf("key/%d", i)
		keys[i] = k
		handler.kvStore.Set(k, fmt.Sprintf("value-%d", i))
	}

	type batchGetRequest struct {
		Keys []string `json:"keys"`
	}
	body, _ := json.Marshal(batchGetRequest{Keys: keys})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/batch/kv/get", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkBatchServiceRegister10(b *testing.B)  { benchmarkBatchServiceRegister(b, 10) }
func BenchmarkBatchServiceRegister100(b *testing.B) { benchmarkBatchServiceRegister(b, 100) }

func benchmarkBatchServiceRegister(b *testing.B, size int) {
	app, handler := benchBatchApp()
	_ = handler

	type batchServiceRequest struct {
		Services []store.Service `json:"services"`
	}
	svcs := make([]store.Service, size)
	for i := 0; i < size; i++ {
		svcs[i] = store.Service{
			Name:    fmt.Sprintf("svc-%d", i),
			Address: "127.0.0.1",
			Port:    8080 + i,
		}
	}
	body, _ := json.Marshal(batchServiceRequest{Services: svcs})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/batch/services/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// ── Concurrent Handler Benchmarks ────────────────────────────────────────────
//
// These measure throughput under parallel load using b.RunParallel.
// Each goroutine issues independent requests; the shared Fiber app and store
// serialize access internally (RWMutex in store, single-listener in app.Test).

func BenchmarkKVHandler_Concurrent_Get(b *testing.B) {
	_, app := benchKVApp()
	// Pre-populate 100 keys so reads always hit.
	for i := 0; i < 100; i++ {
		body := []byte(fmt.Sprintf(`{"value":"v%d"}`, i))
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/kv/key-%d", i), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req) //nolint:errcheck // setup call, error irrelevant to benchmark
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/kv/key-%d", i%100), http.NoBody)
			resp, err := app.Test(req)
			if err != nil {
				b.Error(err)
				return
			}
			resp.Body.Close()
			i++
		}
	})
}

func BenchmarkKVHandler_Concurrent_Set(b *testing.B) {
	_, app := benchKVApp()
	body := []byte(`{"value":"bench-value"}`)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/kv/key-%d", i%100), bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				b.Error(err)
				return
			}
			resp.Body.Close()
			i++
		}
	})
}

// Mixed: 70% GET reads, 20% PUT writes, 10% DELETE — realistic KV workload.
func BenchmarkKVHandler_Concurrent_Mixed(b *testing.B) {
	_, app := benchKVApp()
	setBody := []byte(`{"value":"v"}`)
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/kv/key-%d", i), bytes.NewReader(setBody))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req) //nolint:errcheck // setup call, error irrelevant to benchmark
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("/kv/key-%d", i%100)
			var req *http.Request
			switch i % 10 {
			case 0, 1: // 20% write
				req = httptest.NewRequest(http.MethodPut, key, bytes.NewReader(setBody))
				req.Header.Set("Content-Type", "application/json")
			case 2: // 10% delete
				req = httptest.NewRequest(http.MethodDelete, key, http.NoBody)
			default: // 70% read
				req = httptest.NewRequest(http.MethodGet, key, http.NoBody)
			}
			resp, err := app.Test(req)
			if err != nil {
				b.Error(err)
				return
			}
			resp.Body.Close()
			i++
		}
	})
}

func BenchmarkServiceHandler_Concurrent_Get(b *testing.B) {
	_, app := benchServiceApp()
	for i := 0; i < 50; i++ {
		svc := store.Service{Name: fmt.Sprintf("svc-%d", i), Address: "127.0.0.1", Port: 8080 + i}
		body, _ := json.Marshal(svc)
		req := httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req) //nolint:errcheck // setup call, error irrelevant to benchmark
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet,
				fmt.Sprintf("/services/svc-%d", i%50), http.NoBody)
			resp, err := app.Test(req)
			if err != nil {
				b.Error(err)
				return
			}
			resp.Body.Close()
			i++
		}
	})
}

func BenchmarkServiceHandler_Concurrent_Register(b *testing.B) {
	_, app := benchServiceApp()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			svc := store.Service{
				Name:    fmt.Sprintf("svc-%d", i%50),
				Address: "127.0.0.1",
				Port:    8080 + i%50,
			}
			body, _ := json.Marshal(svc)
			req := httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				b.Error(err)
				return
			}
			resp.Body.Close()
			i++
		}
	})
}

// Mixed service workload: 60% Get, 30% Register, 10% Heartbeat.
func BenchmarkServiceHandler_Concurrent_Mixed(b *testing.B) {
	_, app := benchServiceApp()
	for i := 0; i < 50; i++ {
		svc := store.Service{Name: fmt.Sprintf("svc-%d", i), Address: "127.0.0.1", Port: 8080 + i}
		body, _ := json.Marshal(svc)
		req := httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req) //nolint:errcheck // setup call, error irrelevant to benchmark
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			name := fmt.Sprintf("svc-%d", i%50)
			var req *http.Request
			switch i % 10 {
			case 0, 1, 2: // 30% register
				svc := store.Service{Name: name, Address: "127.0.0.1", Port: 8080}
				body, _ := json.Marshal(svc)
				req = httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			case 3: // 10% heartbeat
				req = httptest.NewRequest(http.MethodPut,
					fmt.Sprintf("/heartbeat/%s", name), http.NoBody)
			default: // 60% get
				req = httptest.NewRequest(http.MethodGet,
					fmt.Sprintf("/services/%s", name), http.NoBody)
			}
			resp, err := app.Test(req)
			if err != nil {
				b.Error(err)
				return
			}
			resp.Body.Close()
			i++
		}
	})
}
