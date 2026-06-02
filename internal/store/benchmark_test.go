package store

import (
	"fmt"
	"testing"

	"go.uber.org/zap/zapcore"

	"github.com/neogan74/konsul/internal/logger"
)

func TestMain(m *testing.M) {
	// Suppress INFO logs during benchmarks to keep output clean.
	logger.SetDefault(logger.New(zapcore.ErrorLevel, "json"))
	m.Run()
}

// ── KV Store Benchmarks ──────────────────────────────────────────────────────

func BenchmarkKVSet(b *testing.B) {
	kv := NewKVStore()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		kv.Set("key", "value")
	}
}

func BenchmarkKVGet(b *testing.B) {
	kv := NewKVStore()
	kv.Set("key", "value")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		kv.Get("key")
	}
}

func BenchmarkKVGetMiss(b *testing.B) {
	kv := NewKVStore()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		kv.Get("nonexistent")
	}
}

func BenchmarkKVDelete(b *testing.B) {
	kv := NewKVStore()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		kv.Set("key", "value")
		b.StartTimer()
		kv.Delete("key")
	}
}

func BenchmarkKVSetCAS(b *testing.B) {
	kv := NewKVStore()
	kv.Set("key", "initial")
	entry, _ := kv.GetEntry("key")
	idx := entry.ModifyIndex
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		newIdx, _ := kv.SetCAS("key", "value", idx)
		idx = newIdx
	}
}

func BenchmarkKVSetCASConflict(b *testing.B) {
	kv := NewKVStore()
	kv.Set("key", "initial")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		kv.SetCAS("key", "value", 0) // always conflicts
	}
}

func BenchmarkKVBatchSet10(b *testing.B)   { benchmarkKVBatchSet(b, 10) }
func BenchmarkKVBatchSet100(b *testing.B)  { benchmarkKVBatchSet(b, 100) }
func BenchmarkKVBatchSet1000(b *testing.B) { benchmarkKVBatchSet(b, 1000) }

func benchmarkKVBatchSet(b *testing.B, size int) {
	kv := NewKVStore()
	items := make(map[string]string, size)
	for i := 0; i < size; i++ {
		items[fmt.Sprintf("key/%d", i)] = fmt.Sprintf("value-%d", i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		kv.BatchSet(items)
	}
}

func BenchmarkKVBatchGet10(b *testing.B)   { benchmarkKVBatchGet(b, 10) }
func BenchmarkKVBatchGet100(b *testing.B)  { benchmarkKVBatchGet(b, 100) }
func BenchmarkKVBatchGet1000(b *testing.B) { benchmarkKVBatchGet(b, 1000) }

func benchmarkKVBatchGet(b *testing.B, size int) {
	kv := NewKVStore()
	keys := make([]string, size)
	for i := 0; i < size; i++ {
		k := fmt.Sprintf("key/%d", i)
		keys[i] = k
		kv.Set(k, fmt.Sprintf("value-%d", i))
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		kv.BatchGet(keys)
	}
}

func BenchmarkKVList(b *testing.B) {
	kv := NewKVStore()
	for i := 0; i < 1000; i++ {
		kv.Set(fmt.Sprintf("key/%d", i), "value")
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		kv.List()
	}
}

// Concurrent workloads

func BenchmarkKVConcurrentSet(b *testing.B) {
	kv := NewKVStore()
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			kv.Set(fmt.Sprintf("key/%d", i%100), "value")
			i++
		}
	})
}

func BenchmarkKVConcurrentGet(b *testing.B) {
	kv := NewKVStore()
	for i := 0; i < 100; i++ {
		kv.Set(fmt.Sprintf("key/%d", i), "value")
	}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			kv.Get(fmt.Sprintf("key/%d", i%100))
			i++
		}
	})
}

// 80% reads / 20% writes — realistic workload
func BenchmarkKVMixed(b *testing.B) {
	kv := NewKVStore()
	for i := 0; i < 100; i++ {
		kv.Set(fmt.Sprintf("key/%d", i), "value")
	}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%5 == 0 {
				kv.Set(fmt.Sprintf("key/%d", i%100), "updated")
			} else {
				kv.Get(fmt.Sprintf("key/%d", i%100))
			}
			i++
		}
	})
}

// ── Service Store Benchmarks ─────────────────────────────────────────────────

func BenchmarkServiceRegister(b *testing.B) {
	ss := NewServiceStore()
	svc := Service{Name: "svc", Address: "127.0.0.1", Port: 8080}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ss.Register(svc)
	}
}

func BenchmarkServiceGet(b *testing.B) {
	ss := NewServiceStore()
	ss.Register(Service{Name: "svc", Address: "127.0.0.1", Port: 8080})
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ss.Get("svc")
	}
}

func BenchmarkServiceGetMiss(b *testing.B) {
	ss := NewServiceStore()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ss.Get("nonexistent")
	}
}

func BenchmarkServiceList10(b *testing.B)   { benchmarkServiceList(b, 10) }
func BenchmarkServiceList100(b *testing.B)  { benchmarkServiceList(b, 100) }
func BenchmarkServiceList1000(b *testing.B) { benchmarkServiceList(b, 1000) }

func benchmarkServiceList(b *testing.B, n int) {
	ss := NewServiceStore()
	for i := 0; i < n; i++ {
		ss.Register(Service{Name: fmt.Sprintf("svc-%d", i), Address: "127.0.0.1", Port: 8080 + i})
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ss.List()
	}
}

func BenchmarkServiceDeregister(b *testing.B) {
	ss := NewServiceStore()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ss.Register(Service{Name: "svc", Address: "127.0.0.1", Port: 8080})
		b.StartTimer()
		ss.Deregister("svc")
	}
}

func BenchmarkServiceHeartbeat(b *testing.B) {
	ss := NewServiceStore()
	ss.Register(Service{Name: "svc", Address: "127.0.0.1", Port: 8080})
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ss.Heartbeat("svc")
	}
}

func BenchmarkServiceRegisterCAS(b *testing.B) {
	ss := NewServiceStore()
	ss.Register(Service{Name: "svc", Address: "127.0.0.1", Port: 8080})
	entry, _ := ss.GetEntry("svc")
	idx := entry.ModifyIndex
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		newIdx, _ := ss.RegisterCAS(Service{Name: "svc", Address: "127.0.0.1", Port: 8080}, idx)
		idx = newIdx
	}
}

// Concurrent service registration — simulates many nodes announcing themselves
func BenchmarkServiceConcurrentRegister(b *testing.B) {
	ss := NewServiceStore()
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ss.Register(Service{
				Name:    fmt.Sprintf("svc-%d", i%50),
				Address: "127.0.0.1",
				Port:    8080 + i%50,
			})
			i++
		}
	})
}

// Mixed read/write: 70% Get, 20% Register, 10% Deregister
func BenchmarkServiceMixed(b *testing.B) {
	ss := NewServiceStore()
	for i := 0; i < 50; i++ {
		ss.Register(Service{Name: fmt.Sprintf("svc-%d", i), Address: "127.0.0.1", Port: 8080 + i})
	}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			name := fmt.Sprintf("svc-%d", i%50)
			switch i % 10 {
			case 0, 1: // 20% register
				ss.Register(Service{Name: name, Address: "127.0.0.1", Port: 8080})
			case 2: // 10% deregister+re-register
				ss.Deregister(name)
			default: // 70% read
				ss.Get(name)
			}
			i++
		}
	})
}
