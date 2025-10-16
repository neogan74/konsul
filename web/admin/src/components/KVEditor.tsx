import { useState } from 'react';
import { X, Save } from 'lucide-react';

interface KVEditorProps {
  initialKey?: string;
  initialValue?: string;
  onSave: (key: string, value: string) => void;
  onClose: () => void;
  mode: 'create' | 'edit';
}

export default function KVEditor({ initialKey = '', initialValue = '', onSave, onClose, mode }: KVEditorProps) {
  const [key, setKey] = useState(initialKey);
  const [value, setValue] = useState(initialValue);
  const [error, setError] = useState('');

  const handleSave = () => {
    if (!key.trim()) {
      setError('Key is required');
      return;
    }

    try {
      // Validate JSON if it looks like JSON
      if (value.trim().startsWith('{') || value.trim().startsWith('[')) {
        JSON.parse(value);
      }
      onSave(key, value);
      onClose();
    } catch (err) {
      setError('Invalid JSON format');
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-slate-800 rounded-lg shadow-xl max-w-2xl w-full border border-slate-700">
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <h2 className="text-xl font-semibold text-white">
            {mode === 'create' ? 'Create New Key' : 'Edit Key'}
          </h2>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition-colors"
          >
            <X size={24} />
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Key
            </label>
            <input
              type="text"
              value={key}
              onChange={(e) => setKey(e.target.value)}
              disabled={mode === 'edit'}
              className="w-full px-4 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
              placeholder="my/key/path"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Value
            </label>
            <textarea
              value={value}
              onChange={(e) => setValue(e.target.value)}
              rows={10}
              className="w-full px-4 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
              placeholder='Enter value (can be JSON)'
            />
          </div>

          {error && (
            <div className="px-4 py-2 bg-red-900/20 border border-red-700 rounded-lg text-red-400 text-sm">
              {error}
            </div>
          )}
        </div>

        <div className="flex items-center justify-end gap-3 p-6 border-t border-slate-700">
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-400 hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors font-medium"
          >
            <Save size={16} />
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
