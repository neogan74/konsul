import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, X, Key, Copy, Check, Shield, Clock } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';

interface APIKey {
  id: string;
  name: string;
  key?: string;
  permissions: string[];
  metadata?: Record<string, string>;
  created_at: string;
  expires_at?: string;
  enabled: boolean;
}

export default function APIKeys() {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [newKeyData, setNewKeyData] = useState<{
    key?: string;
    api_key?: APIKey;
  } | null>(null);
  const [copiedKey, setCopiedKey] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    permissions: 'read,write',
    expiresIn: '31536000', // 1 year in seconds
  });

  const { token } = useAuth();
  const queryClient = useQueryClient();

  const { data: apiKeys, isLoading } = useQuery({
    queryKey: ['apikeys'],
    queryFn: async () => {
      const response = await fetch('/auth/apikeys', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      if (!response.ok) throw new Error('Failed to fetch API keys');
      return response.json() as Promise<APIKey[]>;
    },
    enabled: !!token,
  });

  const createMutation = useMutation({
    mutationFn: async (data: typeof formData) => {
      const response = await fetch('/auth/apikeys', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          name: data.name,
          permissions: data.permissions.split(',').map(p => p.trim()),
          expires_in: parseInt(data.expiresIn),
        }),
      });
      if (!response.ok) throw new Error('Failed to create API key');
      return response.json();
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['apikeys'] });
      setNewKeyData(data);
      setFormData({ name: '', permissions: 'read,write', expiresIn: '31536000' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (keyId: string) => {
      const response = await fetch(`/auth/apikeys/${keyId}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      if (!response.ok) throw new Error('Failed to delete API key');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apikeys'] });
    },
  });

  const revokeMutation = useMutation({
    mutationFn: async (keyId: string) => {
      const response = await fetch(`/auth/apikeys/${keyId}/revoke`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      if (!response.ok) throw new Error('Failed to revoke API key');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apikeys'] });
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createMutation.mutate(formData);
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(true);
    setTimeout(() => setCopiedKey(false), 2000);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">API Keys</h1>
          <p className="text-slate-400">Manage authentication keys for programmatic access</p>
        </div>
        <button
          onClick={() => setIsModalOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
        >
          <Plus size={16} />
          Create API Key
        </button>
      </div>

      {/* API Keys List */}
      {isLoading ? (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-32 bg-slate-800 rounded-lg animate-pulse" />
          ))}
        </div>
      ) : apiKeys && apiKeys.length > 0 ? (
        <div className="space-y-4">
          {apiKeys.map((key) => (
            <div
              key={key.id}
              className="bg-slate-800 rounded-lg p-6 border border-slate-700 hover:border-slate-600 transition-colors"
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-4 flex-1">
                  <div className={`p-3 rounded-lg ${key.enabled ? 'bg-green-900/30' : 'bg-red-900/30'}`}>
                    <Key className={key.enabled ? 'text-green-400' : 'text-red-400'} size={24} />
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <h3 className="text-lg font-semibold text-white">{key.name}</h3>
                      <span
                        className={`px-2 py-1 rounded-full text-xs font-medium ${
                          key.enabled
                            ? 'bg-green-900/30 text-green-400'
                            : 'bg-red-900/30 text-red-400'
                        }`}
                      >
                        {key.enabled ? 'Active' : 'Revoked'}
                      </span>
                    </div>
                    <div className="space-y-1 text-sm">
                      <div className="flex items-center gap-2 text-slate-400">
                        <Shield size={14} />
                        <span>Permissions: {key.permissions.join(', ')}</span>
                      </div>
                      <div className="flex items-center gap-2 text-slate-400">
                        <Clock size={14} />
                        <span>Created: {formatDate(key.created_at)}</span>
                      </div>
                      {key.expires_at && (
                        <div className="flex items-center gap-2 text-slate-400">
                          <Clock size={14} />
                          <span>Expires: {formatDate(key.expires_at)}</span>
                        </div>
                      )}
                      <p className="text-xs text-slate-500 mt-2">ID: {key.id}</p>
                    </div>
                  </div>
                </div>
                <div className="flex gap-2 ml-4">
                  {key.enabled && (
                    <button
                      onClick={() => revokeMutation.mutate(key.id)}
                      className="px-3 py-2 bg-yellow-600 hover:bg-yellow-700 text-white rounded-lg transition-colors text-sm"
                    >
                      Revoke
                    </button>
                  )}
                  <button
                    onClick={() => {
                      if (confirm(`Are you sure you want to delete "${key.name}"?`)) {
                        deleteMutation.mutate(key.id);
                      }
                    }}
                    className="px-3 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors text-sm"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="bg-slate-800 rounded-lg p-12 border border-slate-700 text-center">
          <Key className="mx-auto mb-4 text-slate-600" size={48} />
          <p className="text-slate-400 text-lg">No API keys created yet</p>
          <p className="text-slate-500 mt-2">Click "Create API Key" to generate your first key</p>
        </div>
      )}

      {/* Create API Key Modal */}
      {isModalOpen && !newKeyData && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-slate-800 rounded-lg shadow-xl max-w-2xl w-full border border-slate-700">
            <div className="flex items-center justify-between p-6 border-b border-slate-700">
              <h2 className="text-xl font-semibold text-white">Create New API Key</h2>
              <button
                onClick={() => setIsModalOpen(false)}
                className="text-slate-400 hover:text-white transition-colors"
              >
                <X size={24} />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Key Name *
                </label>
                <input
                  type="text"
                  required
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-4 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="production-api-key"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Permissions (comma-separated) *
                </label>
                <input
                  type="text"
                  required
                  value={formData.permissions}
                  onChange={(e) => setFormData({ ...formData, permissions: e.target.value })}
                  className="w-full px-4 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="read, write"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Expires In (seconds)
                </label>
                <select
                  value={formData.expiresIn}
                  onChange={(e) => setFormData({ ...formData, expiresIn: e.target.value })}
                  className="w-full px-4 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="3600">1 hour</option>
                  <option value="86400">1 day</option>
                  <option value="604800">1 week</option>
                  <option value="2592000">30 days</option>
                  <option value="31536000">1 year</option>
                </select>
              </div>

              <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-700">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 text-slate-400 hover:text-white transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending}
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors font-medium disabled:opacity-50"
                >
                  {createMutation.isPending ? 'Creating...' : 'Create API Key'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* New API Key Display Modal */}
      {newKeyData && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-slate-800 rounded-lg shadow-xl max-w-2xl w-full border border-slate-700">
            <div className="p-6 border-b border-slate-700">
              <h2 className="text-xl font-semibold text-white">API Key Created Successfully!</h2>
            </div>

            <div className="p-6 space-y-4">
              <div className="bg-yellow-900/20 border border-yellow-700 rounded-lg p-4">
                <p className="text-yellow-400 text-sm font-medium mb-2">
                  Important: Save this key now!
                </p>
                <p className="text-yellow-300 text-sm">
                  This is the only time you'll see the full key. Store it securely.
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">API Key</label>
                <div className="flex gap-2">
                  <code className="flex-1 px-4 py-3 bg-slate-900 border border-slate-700 rounded-lg text-green-400 font-mono text-sm break-all">
                    {newKeyData.key}
                  </code>
                  <button
                    onClick={() => newKeyData.key && copyToClipboard(newKeyData.key)}
                    className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg transition-colors"
                  >
                    {copiedKey ? <Check size={20} /> : <Copy size={20} />}
                  </button>
                </div>
              </div>

              {newKeyData.api_key && (
                <div className="space-y-2">
                  <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                    <span className="text-slate-400">Name</span>
                    <span className="text-white font-medium">{newKeyData.api_key.name}</span>
                  </div>
                  <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                    <span className="text-slate-400">Permissions</span>
                    <span className="text-white font-medium">
                      {newKeyData.api_key.permissions.join(', ')}
                    </span>
                  </div>
                  <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                    <span className="text-slate-400">Created</span>
                    <span className="text-white font-medium">
                      {formatDate(newKeyData.api_key.created_at)}
                    </span>
                  </div>
                </div>
              )}
            </div>

            <div className="flex items-center justify-end gap-3 p-6 border-t border-slate-700">
              <button
                onClick={() => {
                  setNewKeyData(null);
                  setIsModalOpen(false);
                }}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors font-medium"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}