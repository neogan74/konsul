package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap/zapcore"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

func init() {
	logger.SetDefault(logger.New(zapcore.ErrorLevel, "json"))
}

// ── KV Handler Benchmarks ────────────────────────────────────────────────────

func BenchmarkKVHandler_Get_Hit(b *testing.B) {
	_, app := setupKVHandler()
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
	_, app := setupKVHandler()
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
	_, app := setupKVHandler()
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
	_, app := setupKVHandler()
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
	_, app := setupServiceHandler()
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
	_, app := setupServiceHandler()
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
	_, app := setupServiceHandler()
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
	_, app := setupServiceHandler()
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
	_, app := setupServiceHandler()
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
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/set", handler.BatchKVSet)

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
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/get", handler.BatchKVGet)

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
	app, handler := setupBatchTestApp()
	app.Post("/batch/services/register", handler.BatchServiceRegister)

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
