import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search, Trash2, Edit2, RefreshCw, FolderTree } from 'lucide-react';
import KVEditor from '../components/KVEditor';
import { listKVWithValues, setKV, deleteKV } from '../lib/api';
import type { KVPair } from '../lib/api';
import { useWebSocket } from '../contexts/WebSocketContext';

export default function KVStore() {
  const [searchTerm, setSearchTerm] = useState('');
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [editorMode, setEditorMode] = useState<'create' | 'edit'>('create');
  const [selectedKV, setSelectedKV] = useState<KVPair | null>(null);

  const queryClient = useQueryClient();
  const { subscribeToKV } = useWebSocket();

  const { data: kvPairs, isLoading, refetch } = useQuery({
    queryKey: ['kv-all'],
    queryFn: listKVWithValues,
    // Removed refetchInterval - using WebSocket instead
  });

  // Subscribe to WebSocket updates
  useEffect(() => {
    const unsubscribe = subscribeToKV((event) => {
      console.log('KV event received:', event);
      // Invalidate and refetch KV store on any change
      queryClient.invalidateQueries({ queryKey: ['kv-all'] });
    });

    return () => unsubscribe();
  }, [subscribeToKV, queryClient]);

  const setKVMutation = useMutation({
    mutationFn: ({ key, value }: { key: string; value: string }) => setKV(key, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['kv-all'] });
      setIsEditorOpen(false);
      setSelectedKV(null);
    },
  });

  const deleteKVMutation = useMutation({
    mutationFn: (key: string) => deleteKV(key),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['kv-all'] });
    },
  });

  const handleCreate = () => {
    setEditorMode('create');
    setSelectedKV(null);
    setIsEditorOpen(true);
  };

  const handleEdit = (kv: KVPair) => {
    setEditorMode('edit');
    setSelectedKV(kv);
    setIsEditorOpen(true);
  };

  const handleSave = (key: string, value: string) => {
    setKVMutation.mutate({ key, value });
  };

  const handleDelete = (key: string) => {
    if (confirm(`Are you sure you want to delete key "${key}"?`)) {
      deleteKVMutation.mutate(key);
    }
  };

  const filteredKVPairs = kvPairs?.filter(kv =>
    kv.key.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const decodeValue = (value: string): string => {
    try {
      // Try to decode base64 if it's encoded
      const decoded = atob(value);
      return decoded;
    } catch {
      return value;
    }
  };

  const formatValue = (value: string): string => {
    const decoded = decodeValue(value);
    try {
      const parsed = JSON.parse(decoded);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return decoded;
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">Key-Value Store</h1>
          <p className="text-slate-400">Browse and manage configuration data</p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={() => refetch()}
            className="flex items-center gap-2 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg transition-colors"
          >
            <RefreshCw size={16} />
            Refresh
          </button>
          <button
            onClick={handleCreate}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
          >
            <Plus size={16} />
            Create Key
          </button>
        </div>
      </div>

      {/* Search Bar */}
      <div className="mb-6">
        <div className="relative">
          <Search className="absolute left-4 top-1/2 transform -translate-y-1/2 text-slate-400" size={20} />
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Search keys..."
            className="w-full pl-12 pr-4 py-3 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>

      {/* KV List */}
      {isLoading ? (
        <div className="space-y-3">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="h-32 bg-slate-800 rounded-lg animate-pulse" />
          ))}
        </div>
      ) : filteredKVPairs && filteredKVPairs.length > 0 ? (
        <div className="space-y-3">
          {filteredKVPairs.map((kv) => (
            <div
              key={kv.key}
              className="bg-slate-800 rounded-lg p-6 border border-slate-700 hover:border-slate-600 transition-colors"
            >
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3 flex-1">
                  <div className="p-2 bg-slate-700 rounded-lg">
                    <FolderTree className="text-slate-300" size={20} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <h3 className="text-lg font-semibold text-white break-all">{kv.key}</h3>
                    {kv.modify_index && (
                      <p className="text-sm text-slate-400">
                        Modified Index: {kv.modify_index}
                      </p>
                    )}
                  </div>
                </div>
                <div className="flex gap-2 ml-4">
                  <button
                    onClick={() => handleEdit(kv)}
                    className="p-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
                    title="Edit"
                  >
                    <Edit2 size={16} />
                  </button>
                  <button
                    onClick={() => handleDelete(kv.key)}
                    className="p-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors"
                    title="Delete"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>

              <div className="bg-slate-900 rounded-lg p-4">
                <pre className="text-sm text-slate-300 font-mono overflow-x-auto">
                  {formatValue(kv.value)}
                </pre>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="bg-slate-800 rounded-lg p-12 border border-slate-700 text-center">
          <p className="text-slate-400 text-lg">
            {searchTerm ? 'No keys match your search' : 'No keys stored yet'}
          </p>
          <p className="text-slate-500 mt-2">
            {searchTerm ? 'Try a different search term' : 'Click "Create Key" to add your first key-value pair'}
          </p>
        </div>
      )}

      {/* KV Editor Modal */}
      {isEditorOpen && (
        <KVEditor
          mode={editorMode}
          initialKey={selectedKV?.key || ''}
          initialValue={selectedKV ? formatValue(selectedKV.value) : ''}
          onSave={handleSave}
          onClose={() => {
            setIsEditorOpen(false);
            setSelectedKV(null);
          }}
        />
      )}
    </div>
  );
}
