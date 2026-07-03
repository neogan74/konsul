package store

import "strings"

// NamespacedKV wraps a KVStore and prefixes every key with "ns:<namespace>:".
// It provides the same method surface as KVStore without changing the underlying
// store's interface — handlers construct one per request using the resolved namespace.
type NamespacedKV struct {
	store     *KVStore
	namespace string
	prefix    string // "ns:<namespace>:"
}

// NewNamespacedKV returns a NamespacedKV for the given namespace.
// Panics if namespace is empty to catch misconfiguration early.
func NewNamespacedKV(store *KVStore, namespace string) *NamespacedKV {
	if namespace == "" {
		panic("NamespacedKV: empty namespace")
	}
	return &NamespacedKV{
		store:     store,
		namespace: namespace,
		prefix:    "ns:" + namespace + ":",
	}
}

func (n *NamespacedKV) nsKey(key string) string { return n.prefix + key }

func (n *NamespacedKV) stripPrefix(key string) string {
	return strings.TrimPrefix(key, n.prefix)
}

// --- Read operations ---

func (n *NamespacedKV) Get(key string) (string, bool) {
	return n.store.Get(n.nsKey(key))
}

func (n *NamespacedKV) GetEntry(key string) (KVEntry, bool) {
	return n.store.GetEntry(n.nsKey(key))
}

// List returns all keys in this namespace, with the namespace prefix stripped.
func (n *NamespacedKV) List() []string {
	all := n.store.List()
	out := make([]string, 0, len(all))
	for _, k := range all {
		if strings.HasPrefix(k, n.prefix) {
			out = append(out, n.stripPrefix(k))
		}
	}
	return out
}

// ListEntries returns all entries in this namespace with keys stripped of the prefix.
func (n *NamespacedKV) ListEntries() map[string]KVEntry {
	all := n.store.ListEntries()
	out := make(map[string]KVEntry)
	for k, v := range all {
		if strings.HasPrefix(k, n.prefix) {
			out[n.stripPrefix(k)] = v
		}
	}
	return out
}

func (n *NamespacedKV) BatchGet(keys []string) (found map[string]string, notFound []string) {
	nsKeys := make([]string, len(keys))
	for i, k := range keys {
		nsKeys[i] = n.nsKey(k)
	}
	rawFound, rawNotFound := n.store.BatchGet(nsKeys)
	found = make(map[string]string, len(rawFound))
	for k, v := range rawFound {
		found[n.stripPrefix(k)] = v
	}
	notFound = make([]string, len(rawNotFound))
	for i, k := range rawNotFound {
		notFound[i] = n.stripPrefix(k)
	}
	return found, notFound
}

func (n *NamespacedKV) BatchGetEntries(keys []string) (found map[string]KVEntry, notFound []string) {
	nsKeys := make([]string, len(keys))
	for i, k := range keys {
		nsKeys[i] = n.nsKey(k)
	}
	rawFound, rawNotFound := n.store.BatchGetEntries(nsKeys)
	found = make(map[string]KVEntry, len(rawFound))
	for k, v := range rawFound {
		found[n.stripPrefix(k)] = v
	}
	notFound = make([]string, len(rawNotFound))
	for i, k := range rawNotFound {
		notFound[i] = n.stripPrefix(k)
	}
	return found, notFound
}

// --- Write operations ---

func (n *NamespacedKV) Set(key, value string) {
	n.store.Set(n.nsKey(key), value)
}

func (n *NamespacedKV) SetWithFlags(key, value string, flags uint64) {
	n.store.SetWithFlags(n.nsKey(key), value, flags)
}

func (n *NamespacedKV) SetCAS(key, value string, expectedIndex uint64) (uint64, error) {
	return n.store.SetCAS(n.nsKey(key), value, expectedIndex)
}

func (n *NamespacedKV) Delete(key string) {
	n.store.Delete(n.nsKey(key))
}

func (n *NamespacedKV) DeleteCAS(key string, expectedIndex uint64) error {
	return n.store.DeleteCAS(n.nsKey(key), expectedIndex)
}

func (n *NamespacedKV) BatchSet(items map[string]string) error {
	ns := make(map[string]string, len(items))
	for k, v := range items {
		ns[n.nsKey(k)] = v
	}
	return n.store.BatchSet(ns)
}

func (n *NamespacedKV) BatchSetCAS(items map[string]string, expectedIndices map[string]uint64) (map[string]uint64, error) {
	nsItems := make(map[string]string, len(items))
	for k, v := range items {
		nsItems[n.nsKey(k)] = v
	}
	nsIndices := make(map[string]uint64, len(expectedIndices))
	for k, v := range expectedIndices {
		nsIndices[n.nsKey(k)] = v
	}
	rawResult, err := n.store.BatchSetCAS(nsItems, nsIndices)
	result := make(map[string]uint64, len(rawResult))
	for k, v := range rawResult {
		result[n.stripPrefix(k)] = v
	}
	return result, err
}

func (n *NamespacedKV) BatchDelete(keys []string) error {
	nsKeys := make([]string, len(keys))
	for i, k := range keys {
		nsKeys[i] = n.nsKey(k)
	}
	return n.store.BatchDelete(nsKeys)
}

func (n *NamespacedKV) BatchDeleteCAS(keys []string, expectedIndices map[string]uint64) error {
	nsKeys := make([]string, len(keys))
	for i, k := range keys {
		nsKeys[i] = n.nsKey(k)
	}
	nsIndices := make(map[string]uint64, len(expectedIndices))
	for k, v := range expectedIndices {
		nsIndices[n.nsKey(k)] = v
	}
	return n.store.BatchDeleteCAS(nsKeys, nsIndices)
}

// --- Local (FSM) operations — used by Raft FSM after applying log entries ---

func (n *NamespacedKV) SetLocal(key, value string) {
	n.store.SetLocal(n.nsKey(key), value)
}

func (n *NamespacedKV) SetWithFlagsLocal(key, value string, flags uint64) {
	n.store.SetWithFlagsLocal(n.nsKey(key), value, flags)
}

func (n *NamespacedKV) SetCASLocal(key, value string, expectedIndex uint64) (uint64, error) {
	return n.store.SetCASLocal(n.nsKey(key), value, expectedIndex)
}

func (n *NamespacedKV) DeleteLocal(key string) {
	n.store.DeleteLocal(n.nsKey(key))
}

func (n *NamespacedKV) DeleteCASLocal(key string, expectedIndex uint64) error {
	return n.store.DeleteCASLocal(n.nsKey(key), expectedIndex)
}

func (n *NamespacedKV) BatchSetLocal(items map[string]string) error {
	ns := make(map[string]string, len(items))
	for k, v := range items {
		ns[n.nsKey(k)] = v
	}
	return n.store.BatchSetLocal(ns)
}

func (n *NamespacedKV) BatchSetCASLocal(items map[string]string, expectedIndices map[string]uint64) (map[string]uint64, error) {
	nsItems := make(map[string]string, len(items))
	for k, v := range items {
		nsItems[n.nsKey(k)] = v
	}
	nsIndices := make(map[string]uint64, len(expectedIndices))
	for k, v := range expectedIndices {
		nsIndices[n.nsKey(k)] = v
	}
	rawResult, err := n.store.BatchSetCASLocal(nsItems, nsIndices)
	result := make(map[string]uint64, len(rawResult))
	for k, v := range rawResult {
		result[n.stripPrefix(k)] = v
	}
	return result, err
}

func (n *NamespacedKV) BatchDeleteLocal(keys []string) error {
	nsKeys := make([]string, len(keys))
	for i, k := range keys {
		nsKeys[i] = n.nsKey(k)
	}
	return n.store.BatchDeleteLocal(nsKeys)
}

func (n *NamespacedKV) BatchDeleteCASLocal(keys []string, expectedIndices map[string]uint64) error {
	nsKeys := make([]string, len(keys))
	for i, k := range keys {
		nsKeys[i] = n.nsKey(k)
	}
	nsIndices := make(map[string]uint64, len(expectedIndices))
	for k, v := range expectedIndices {
		nsIndices[n.nsKey(k)] = v
	}
	return n.store.BatchDeleteCASLocal(nsKeys, nsIndices)
}
