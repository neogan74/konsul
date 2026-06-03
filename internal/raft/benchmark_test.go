package raft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	goraft "github.com/hashicorp/raft"
	"go.uber.org/zap/zapcore"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

func init() {
	// Suppress INFO logs from store operations during benchmarks.
	logger.SetDefault(logger.New(zapcore.ErrorLevel, "json"))
}

// newTestFSM creates an FSM backed by real in-memory stores.
func newTestFSM() *KonsulFSM {
	return NewFSM(FSMConfig{
		KVStore:      store.NewKVStore(),
		ServiceStore: store.NewServiceStore(),
	})
}

// mustMarshalCmd builds and serializes a Command, panicking on any error.
func mustMarshalCmd(cmdType CommandType, payload interface{}) []byte {
	cmd, err := NewCommand(cmdType, payload)
	if err != nil {
		panic(err)
	}
	data, err := cmd.Marshal()
	if err != nil {
		panic(err)
	}
	return data
}

// logEntry wraps raw bytes into a raft.Log suitable for FSM.Apply.
func logEntry(data []byte) *goraft.Log {
	return &goraft.Log{Type: goraft.LogCommand, Data: data}
}

// ── Command serialization ────────────────────────────────────────────────────

func BenchmarkCommandMarshal_KVSet(b *testing.B) {
	payload := KVSetPayload{Key: "config/app/timeout", Value: "30s"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd, _ := NewCommand(CmdKVSet, payload)
		_, _ = cmd.Marshal()
	}
}

func BenchmarkCommandUnmarshal_KVSet(b *testing.B) {
	data := mustMarshalCmd(CmdKVSet, KVSetPayload{Key: "k", Value: "v"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = UnmarshalCommand(data)
	}
}

func BenchmarkCommandRoundtrip_KVSet(b *testing.B) {
	payload := KVSetPayload{Key: "config/app/timeout", Value: "30s"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd, _ := NewCommand(CmdKVSet, payload)
		data, _ := cmd.Marshal()
		_, _ = UnmarshalCommand(data)
	}
}

func BenchmarkCommandRoundtrip_BatchSet10(b *testing.B)   { benchmarkBatchCmdRoundtrip(b, 10) }
func BenchmarkCommandRoundtrip_BatchSet100(b *testing.B)  { benchmarkBatchCmdRoundtrip(b, 100) }
func BenchmarkCommandRoundtrip_BatchSet1000(b *testing.B) { benchmarkBatchCmdRoundtrip(b, 1000) }

func benchmarkBatchCmdRoundtrip(b *testing.B, size int) {
	items := make(map[string]string, size)
	for i := 0; i < size; i++ {
		items[fmt.Sprintf("key/%d", i)] = fmt.Sprintf("value-%d", i)
	}
	payload := KVBatchSetPayload{Items: items}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd, _ := NewCommand(CmdKVBatchSet, payload)
		data, _ := cmd.Marshal()
		_, _ = UnmarshalCommand(data)
	}
}

// ── FSM Apply — KV ──────────────────────────────────────────────────────────

func BenchmarkFSMApply_KVSet(b *testing.B) {
	fsm := newTestFSM()
	entry := logEntry(mustMarshalCmd(CmdKVSet, KVSetPayload{Key: "k", Value: "v"}))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fsm.Apply(entry)
	}
}

func BenchmarkFSMApply_KVSetCAS_Hit(b *testing.B) {
	fsm := newTestFSM()
	// Prime the key and track its current index.
	fsm.Apply(logEntry(mustMarshalCmd(CmdKVSet, KVSetPayload{Key: "k", Value: "v0"})))
	snap, _ := fsm.kvStore.GetEntrySnapshot("k")
	idx := snap.ModifyIndex
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := fsm.Apply(logEntry(mustMarshalCmd(CmdKVSetCAS,
			KVSetCASPayload{Key: "k", Value: "vN", ExpectedIndex: idx})))
		if res, ok := r.(*CASResult); ok && res.Err == nil {
			idx = res.NewIndex
		}
	}
}

func BenchmarkFSMApply_KVSetCAS_Miss(b *testing.B) {
	fsm := newTestFSM()
	fsm.Apply(logEntry(mustMarshalCmd(CmdKVSet, KVSetPayload{Key: "k", Value: "v"})))
	entry := logEntry(mustMarshalCmd(CmdKVSetCAS,
		KVSetCASPayload{Key: "k", Value: "new", ExpectedIndex: 0}))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fsm.Apply(entry)
	}
}

func BenchmarkFSMApply_KVDelete(b *testing.B) {
	fsm := newTestFSM()
	entry := logEntry(mustMarshalCmd(CmdKVDelete, KVDeletePayload{Key: "k"}))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fsm.Apply(logEntry(mustMarshalCmd(CmdKVSet, KVSetPayload{Key: "k", Value: "v"})))
		fsm.Apply(entry)
	}
}

func BenchmarkFSMApply_KVBatchSet10(b *testing.B)   { benchmarkFSMBatchSet(b, 10) }
func BenchmarkFSMApply_KVBatchSet100(b *testing.B)  { benchmarkFSMBatchSet(b, 100) }
func BenchmarkFSMApply_KVBatchSet1000(b *testing.B) { benchmarkFSMBatchSet(b, 1000) }

func benchmarkFSMBatchSet(b *testing.B, size int) {
	fsm := newTestFSM()
	items := make(map[string]string, size)
	for i := 0; i < size; i++ {
		items[fmt.Sprintf("key/%d", i)] = fmt.Sprintf("value-%d", i)
	}
	entry := logEntry(mustMarshalCmd(CmdKVBatchSet, KVBatchSetPayload{Items: items}))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fsm.Apply(entry)
	}
}

// ── FSM Apply — Service ──────────────────────────────────────────────────────

func BenchmarkFSMApply_ServiceRegister(b *testing.B) {
	fsm := newTestFSM()
	entry := logEntry(mustMarshalCmd(CmdServiceRegister, ServiceRegisterPayload{
		Service: store.Service{Name: "web", Address: "10.0.0.1", Port: 8080},
	}))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fsm.Apply(entry)
	}
}

func BenchmarkFSMApply_ServiceHeartbeat(b *testing.B) {
	fsm := newTestFSM()
	fsm.Apply(logEntry(mustMarshalCmd(CmdServiceRegister, ServiceRegisterPayload{
		Service: store.Service{Name: "web", Address: "10.0.0.1", Port: 8080},
	})))
	entry := logEntry(mustMarshalCmd(CmdServiceHeartbeat, ServiceHeartbeatPayload{Name: "web"}))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fsm.Apply(entry)
	}
}

func BenchmarkFSMApply_ServiceDeregister(b *testing.B) {
	fsm := newTestFSM()
	reg := logEntry(mustMarshalCmd(CmdServiceRegister, ServiceRegisterPayload{
		Service: store.Service{Name: "web", Address: "10.0.0.1", Port: 8080},
	}))
	dereg := logEntry(mustMarshalCmd(CmdServiceDeregister, ServiceDeregisterPayload{Name: "web"}))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fsm.Apply(reg)
		fsm.Apply(dereg)
	}
}

// ── FSM Snapshot / Restore ───────────────────────────────────────────────────

func BenchmarkFSMSnapshot_100(b *testing.B)  { benchmarkFSMSnapshot(b, 100) }
func BenchmarkFSMSnapshot_1000(b *testing.B) { benchmarkFSMSnapshot(b, 1000) }

func benchmarkFSMSnapshot(b *testing.B, kvEntries int) {
	fsm := newTestFSM()
	for i := 0; i < kvEntries; i++ {
		fsm.Apply(logEntry(mustMarshalCmd(CmdKVSet,
			KVSetPayload{Key: fmt.Sprintf("key/%d", i), Value: fmt.Sprintf("value-%d", i)})))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snap, err := fsm.Snapshot()
		if err != nil {
			b.Fatal(err)
		}
		snap.Release()
	}
}

func BenchmarkFSMSnapshotPersist_100(b *testing.B)  { benchmarkFSMSnapshotPersist(b, 100) }
func BenchmarkFSMSnapshotPersist_1000(b *testing.B) { benchmarkFSMSnapshotPersist(b, 1000) }

func benchmarkFSMSnapshotPersist(b *testing.B, kvEntries int) {
	fsm := newTestFSM()
	for i := 0; i < kvEntries; i++ {
		fsm.Apply(logEntry(mustMarshalCmd(CmdKVSet,
			KVSetPayload{Key: fmt.Sprintf("key/%d", i), Value: fmt.Sprintf("value-%d", i)})))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snap, err := fsm.Snapshot()
		if err != nil {
			b.Fatal(err)
		}
		sink := &memSink{buf: &bytes.Buffer{}}
		if err := snap.Persist(sink); err != nil {
			b.Fatal(err)
		}
		snap.Release()
	}
}

func BenchmarkFSMRestore_100(b *testing.B)  { benchmarkFSMRestore(b, 100) }
func BenchmarkFSMRestore_1000(b *testing.B) { benchmarkFSMRestore(b, 1000) }

func benchmarkFSMRestore(b *testing.B, kvEntries int) {
	// Build a source FSM and serialize its snapshot once.
	src := newTestFSM()
	for i := 0; i < kvEntries; i++ {
		src.Apply(logEntry(mustMarshalCmd(CmdKVSet,
			KVSetPayload{Key: fmt.Sprintf("key/%d", i), Value: fmt.Sprintf("value-%d", i)})))
	}
	snap, err := src.Snapshot()
	if err != nil {
		b.Fatal(err)
	}
	sink := &memSink{buf: &bytes.Buffer{}}
	if err := snap.Persist(sink); err != nil {
		b.Fatal(err)
	}
	snapBytes := sink.buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fsm := newTestFSM()
		if err := fsm.Restore(io.NopCloser(bytes.NewReader(snapBytes))); err != nil {
			b.Fatal(err)
		}
	}
}

// ── Concurrent FSM Apply ─────────────────────────────────────────────────────

func BenchmarkFSMApply_Concurrent_KVSet(b *testing.B) {
	fsm := newTestFSM()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			fsm.Apply(logEntry(mustMarshalCmd(CmdKVSet,
				KVSetPayload{Key: fmt.Sprintf("k%d", i%100), Value: "v"})))
			i++
		}
	})
}

func BenchmarkFSMApply_Concurrent_ServiceHeartbeat(b *testing.B) {
	fsm := newTestFSM()
	// Pre-register 10 services.
	for i := 0; i < 10; i++ {
		fsm.Apply(logEntry(mustMarshalCmd(CmdServiceRegister, ServiceRegisterPayload{
			Service: store.Service{Name: fmt.Sprintf("svc-%d", i), Address: "127.0.0.1", Port: 8080 + i},
		})))
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			fsm.Apply(logEntry(mustMarshalCmd(CmdServiceHeartbeat,
				ServiceHeartbeatPayload{Name: fmt.Sprintf("svc-%d", i%10)})))
			i++
		}
	})
}

// ── memSink — in-memory raft.SnapshotSink ────────────────────────────────────

type memSink struct {
	buf *bytes.Buffer
	id  string
}

func (s *memSink) Write(p []byte) (int, error) { return s.buf.Write(p) }
func (s *memSink) Close() error                { return nil }
func (s *memSink) ID() string                  { return s.id }
func (s *memSink) Cancel() error               { return nil }

// ── Payload size measurement (runs as a regular test) ────────────────────────

func TestCommandPayloadSizes(t *testing.T) {
	cases := []struct {
		name    string
		cmdType CommandType
		payload interface{}
	}{
		{"KVSet", CmdKVSet, KVSetPayload{Key: "config/app/timeout", Value: "30s"}},
		{"KVSetCAS", CmdKVSetCAS, KVSetCASPayload{Key: "k", Value: "v", ExpectedIndex: 42}},
		{"KVBatchSet10", CmdKVBatchSet, func() KVBatchSetPayload {
			m := make(map[string]string, 10)
			for i := 0; i < 10; i++ {
				m[fmt.Sprintf("key/%d", i)] = fmt.Sprintf("value-%d", i)
			}
			return KVBatchSetPayload{Items: m}
		}()},
		{"ServiceRegister", CmdServiceRegister, ServiceRegisterPayload{
			Service: store.Service{Name: "web", Address: "10.0.0.1", Port: 8080, Tags: []string{"primary"}},
		}},
		{"ServiceHeartbeat", CmdServiceHeartbeat, ServiceHeartbeatPayload{Name: "web"}},
	}

	for _, tc := range cases {
		cmd, err := NewCommand(tc.cmdType, tc.payload)
		if err != nil {
			t.Fatalf("%s: NewCommand: %v", tc.name, err)
		}
		raw, err := cmd.Marshal()
		if err != nil {
			t.Fatalf("%s: Marshal: %v", tc.name, err)
		}
		payload, _ := json.Marshal(tc.payload)
		t.Logf("%-20s  total=%4d B  payload=%4d B", tc.name, len(raw), len(payload))
	}
}

func TestSnapshotSize(t *testing.T) {
	for _, n := range []int{100, 1000} {
		fsm := newTestFSM()
		for i := 0; i < n; i++ {
			fsm.Apply(logEntry(mustMarshalCmd(CmdKVSet,
				KVSetPayload{
					Key:   fmt.Sprintf("key/%d", i),
					Value: fmt.Sprintf("value-%d-%s", i, time.Now().String()),
				})))
		}
		snap, _ := fsm.Snapshot()
		sink := &memSink{buf: &bytes.Buffer{}}
		snap.Persist(sink)
		t.Logf("Snapshot %5d KV entries = %7d B (%.1f KB)", n, sink.buf.Len(), float64(sink.buf.Len())/1024)
	}
}
