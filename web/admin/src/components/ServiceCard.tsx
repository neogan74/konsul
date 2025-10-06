import { Server, Trash2, Heart } from 'lucide-react';
import type { Service } from '../lib/api';

interface ServiceCardProps {
  service: Service;
  onDeregister: (id: string) => void;
  onHeartbeat: (id: string) => void;
}

export default function ServiceCard({ service, onDeregister, onHeartbeat }: ServiceCardProps) {
  const isHealthy = service.status === 'passing' || service.status === 'healthy';

  return (
    <div className="bg-slate-800 rounded-lg p-6 border border-slate-700 hover:border-slate-600 transition-colors">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className={`p-2 rounded-lg ${isHealthy ? 'bg-green-900/30' : 'bg-red-900/30'}`}>
            <Server className={isHealthy ? 'text-green-400' : 'text-red-400'} size={20} />
          </div>
          <div>
            <h3 className="text-lg font-semibold text-white">{service.name}</h3>
            <p className="text-sm text-slate-400">{service.id}</p>
          </div>
        </div>
        <span
          className={`px-3 py-1 rounded-full text-xs font-medium ${
            isHealthy
              ? 'bg-green-900/30 text-green-400'
              : 'bg-red-900/30 text-red-400'
          }`}
        >
          {service.status || 'unknown'}
        </span>
      </div>

      <div className="space-y-2 mb-4">
        <div className="flex items-center gap-2 text-sm">
          <span className="text-slate-400">Address:</span>
          <span className="text-white">{service.address}:{service.port}</span>
        </div>
        {service.tags && service.tags.length > 0 && (
          <div className="flex items-center gap-2 text-sm">
            <span className="text-slate-400">Tags:</span>
            <div className="flex flex-wrap gap-1">
              {service.tags.map((tag, idx) => (
                <span
                  key={idx}
                  className="px-2 py-0.5 bg-slate-700 text-slate-300 rounded text-xs"
                >
                  {tag}
                </span>
              ))}
            </div>
          </div>
        )}
        {service.last_heartbeat && (
          <div className="flex items-center gap-2 text-sm">
            <span className="text-slate-400">Last Heartbeat:</span>
            <span className="text-white">{new Date(service.last_heartbeat).toLocaleString()}</span>
          </div>
        )}
      </div>

      <div className="flex gap-2">
        <button
          onClick={() => onHeartbeat(service.id)}
          className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors text-sm font-medium"
        >
          <Heart size={16} />
          Heartbeat
        </button>
        <button
          onClick={() => onDeregister(service.id)}
          className="flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors text-sm font-medium"
        >
          <Trash2 size={16} />
          Remove
        </button>
      </div>
    </div>
  );
}
