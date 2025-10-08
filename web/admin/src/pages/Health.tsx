import { useQuery } from '@tanstack/react-query';
import { Activity, Clock, Cpu, HardDrive, Zap } from 'lucide-react';
import StatCard from '../components/StatCard';
import { getHealth } from '../lib/api';

export default function Health() {
  const { data: health, isLoading } = useQuery({
    queryKey: ['health'],
    queryFn: getHealth,
    refetchInterval: 5000,
  });

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const formatUptime = (seconds: number): string => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    const parts = [];
    if (days > 0) parts.push(`${days}d`);
    if (hours > 0) parts.push(`${hours}h`);
    if (mins > 0) parts.push(`${mins}m`);
    if (secs > 0 || parts.length === 0) parts.push(`${secs}s`);

    return parts.join(' ');
  };

  return (
    <div className="p-6">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white mb-2">System Health</h1>
        <p className="text-slate-400">Monitor cluster health and performance metrics</p>
      </div>

      {/* Status Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <StatCard
          title="Status"
          value={health?.status || 'unknown'}
          icon={Activity}
          loading={isLoading}
        />
        <StatCard
          title="Uptime"
          value={health?.uptime ? formatUptime(health.uptime) : '0s'}
          icon={Clock}
          loading={isLoading}
        />
        <StatCard
          title="Services"
          value={health?.services?.total || 0}
          icon={Zap}
          loading={isLoading}
          trend={health?.services ? {
            value: `${health.services.healthy} healthy`,
            positive: health.services.healthy === health.services.total,
          } : undefined}
        />
        <StatCard
          title="KV Keys"
          value={health?.kv?.total_keys || 0}
          icon={HardDrive}
          loading={isLoading}
        />
      </div>

      {/* Detailed Metrics */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Service Statistics */}
        <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
          <div className="flex items-center gap-3 mb-6">
            <div className="p-3 bg-slate-700 rounded-lg">
              <Zap className="text-blue-400" size={24} />
            </div>
            <h2 className="text-xl font-semibold text-white">Service Statistics</h2>
          </div>

          {isLoading ? (
            <div className="space-y-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-16 bg-slate-700 rounded animate-pulse" />
              ))}
            </div>
          ) : health?.services ? (
            <div className="space-y-4">
              <div className="p-4 bg-slate-900 rounded-lg">
                <div className="flex justify-between items-center mb-2">
                  <span className="text-slate-400">Total Services</span>
                  <span className="text-2xl font-bold text-white">{health.services.total}</span>
                </div>
                <div className="w-full bg-slate-700 rounded-full h-2">
                  <div
                    className="bg-blue-500 h-2 rounded-full transition-all duration-300"
                    style={{ width: '100%' }}
                  />
                </div>
              </div>

              <div className="p-4 bg-slate-900 rounded-lg">
                <div className="flex justify-between items-center mb-2">
                  <span className="text-slate-400">Healthy Services</span>
                  <span className="text-2xl font-bold text-green-400">{health.services.healthy}</span>
                </div>
                <div className="w-full bg-slate-700 rounded-full h-2">
                  <div
                    className="bg-green-500 h-2 rounded-full transition-all duration-300"
                    style={{
                      width: `${health.services.total > 0 ? (health.services.healthy / health.services.total) * 100 : 0}%`,
                    }}
                  />
                </div>
              </div>

              <div className="p-4 bg-slate-900 rounded-lg">
                <div className="flex justify-between items-center mb-2">
                  <span className="text-slate-400">Unhealthy Services</span>
                  <span className="text-2xl font-bold text-red-400">{health.services.unhealthy}</span>
                </div>
                <div className="w-full bg-slate-700 rounded-full h-2">
                  <div
                    className="bg-red-500 h-2 rounded-full transition-all duration-300"
                    style={{
                      width: `${health.services.total > 0 ? (health.services.unhealthy / health.services.total) * 100 : 0}%`,
                    }}
                  />
                </div>
              </div>
            </div>
          ) : (
            <p className="text-slate-400 text-center py-8">No service data available</p>
          )}
        </div>

        {/* System Metrics */}
        <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
          <div className="flex items-center gap-3 mb-6">
            <div className="p-3 bg-slate-700 rounded-lg">
              <Cpu className="text-purple-400" size={24} />
            </div>
            <h2 className="text-xl font-semibold text-white">System Metrics</h2>
          </div>

          {isLoading ? (
            <div className="space-y-4">
              {[1, 2, 3, 4, 5].map((i) => (
                <div key={i} className="h-12 bg-slate-700 rounded animate-pulse" />
              ))}
            </div>
          ) : health?.system ? (
            <div className="space-y-3">
              <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                <span className="text-slate-400">Goroutines</span>
                <span className="text-white font-medium">{health.system.goroutines}</span>
              </div>
              <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                <span className="text-slate-400">Memory Allocated</span>
                <span className="text-white font-medium">{formatBytes(health.system.memory.alloc)}</span>
              </div>
              <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                <span className="text-slate-400">Total Allocated</span>
                <span className="text-white font-medium">{formatBytes(health.system.memory.total_alloc)}</span>
              </div>
              <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                <span className="text-slate-400">System Memory</span>
                <span className="text-white font-medium">{formatBytes(health.system.memory.sys)}</span>
              </div>
              <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                <span className="text-slate-400">Heap Allocated</span>
                <span className="text-white font-medium">{formatBytes(health.system.memory.heap_alloc)}</span>
              </div>
              <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                <span className="text-slate-400">Heap System</span>
                <span className="text-white font-medium">{formatBytes(health.system.memory.heap_sys)}</span>
              </div>
            </div>
          ) : (
            <p className="text-slate-400 text-center py-8">No system metrics available</p>
          )}
        </div>

        {/* General Information */}
        <div className="bg-slate-800 rounded-lg p-6 border border-slate-700 lg:col-span-2">
          <div className="flex items-center gap-3 mb-6">
            <div className="p-3 bg-slate-700 rounded-lg">
              <Activity className="text-green-400" size={24} />
            </div>
            <h2 className="text-xl font-semibold text-white">General Information</h2>
          </div>

          {isLoading ? (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-24 bg-slate-700 rounded animate-pulse" />
              ))}
            </div>
          ) : health ? (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="p-4 bg-slate-900 rounded-lg">
                <p className="text-slate-400 text-sm mb-1">Overall Status</p>
                <p className={`text-2xl font-bold ${
                  health.status === 'healthy' ? 'text-green-400' : 'text-yellow-400'
                }`}>
                  {health.status}
                </p>
              </div>
              <div className="p-4 bg-slate-900 rounded-lg">
                <p className="text-slate-400 text-sm mb-1">Version</p>
                <p className="text-2xl font-bold text-white">{health.version || 'N/A'}</p>
              </div>
              <div className="p-4 bg-slate-900 rounded-lg">
                <p className="text-slate-400 text-sm mb-1">Timestamp</p>
                <p className="text-lg font-bold text-white">
                  {new Date(health.timestamp).toLocaleString()}
                </p>
              </div>
            </div>
          ) : (
            <p className="text-slate-400 text-center py-8">Unable to fetch health data</p>
          )}
        </div>
      </div>
    </div>
  );
}
