import { useQuery } from '@tanstack/react-query';
import { Server, Database, Activity, Clock } from 'lucide-react';
import StatCard from '../components/StatCard';
import { getServices, listKV, getHealth } from '../lib/api';

export default function Dashboard() {
  const { data: services, isLoading: servicesLoading } = useQuery({
    queryKey: ['services'],
    queryFn: getServices,
    refetchInterval: 5000,
  });

  const { data: kvKeys, isLoading: kvLoading } = useQuery({
    queryKey: ['kv-list'],
    queryFn: listKV,
    refetchInterval: 5000,
  });

  const { data: health, isLoading: healthLoading } = useQuery({
    queryKey: ['health'],
    queryFn: getHealth,
    refetchInterval: 5000,
  });

  const healthyServices = services?.filter(s => s.status === 'passing' || s.status === 'healthy').length || 0;

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const mins = Math.floor((seconds % 3600) / 60);

    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h ${mins}m`;
    return `${mins}m`;
  };

  return (
    <div className="p-6">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white mb-2">Dashboard</h1>
        <p className="text-slate-400">Overview of your Konsul cluster</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <StatCard
          title="Total Services"
          value={services?.length || 0}
          icon={Server}
          loading={servicesLoading}
          trend={{
            value: `${healthyServices} healthy`,
            positive: true,
          }}
        />
        <StatCard
          title="KV Keys"
          value={kvKeys?.length || 0}
          icon={Database}
          loading={kvLoading}
        />
        <StatCard
          title="System Status"
          value={health?.status || 'unknown'}
          icon={Activity}
          loading={healthLoading}
        />
        <StatCard
          title="Uptime"
          value={health?.uptime ? formatUptime(health.uptime) : '0m'}
          icon={Clock}
          loading={healthLoading}
        />
      </div>

      {/* Recent Services */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
          <h2 className="text-xl font-semibold text-white mb-4">Recent Services</h2>
          {servicesLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-16 bg-slate-700 rounded animate-pulse" />
              ))}
            </div>
          ) : services && services.length > 0 ? (
            <div className="space-y-3">
              {services.slice(0, 5).map((service) => (
                <div
                  key={service.id}
                  className="flex items-center justify-between p-3 bg-slate-900 rounded-lg"
                >
                  <div>
                    <p className="text-white font-medium">{service.name}</p>
                    <p className="text-sm text-slate-400">
                      {service.address}:{service.port}
                    </p>
                  </div>
                  <span
                    className={`px-3 py-1 rounded-full text-xs font-medium ${
                      service.status === 'passing' || service.status === 'healthy'
                        ? 'bg-green-900/30 text-green-400'
                        : 'bg-red-900/30 text-red-400'
                    }`}
                  >
                    {service.status || 'unknown'}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-slate-400 text-center py-8">No services registered</p>
          )}
        </div>

        {/* System Info */}
        <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
          <h2 className="text-xl font-semibold text-white mb-4">System Information</h2>
          {healthLoading ? (
            <div className="space-y-3">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="h-12 bg-slate-700 rounded animate-pulse" />
              ))}
            </div>
          ) : health ? (
            <div className="space-y-3">
              <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                <span className="text-slate-400">Status</span>
                <span className="text-white font-medium">{health.status}</span>
              </div>
              <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                <span className="text-slate-400">Version</span>
                <span className="text-white font-medium">{health.version || 'N/A'}</span>
              </div>
              {health.services && (
                <>
                  <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                    <span className="text-slate-400">Healthy Services</span>
                    <span className="text-green-400 font-medium">
                      {health.services.healthy} / {health.services.total}
                    </span>
                  </div>
                </>
              )}
              {health.system && (
                <div className="flex justify-between p-3 bg-slate-900 rounded-lg">
                  <span className="text-slate-400">Goroutines</span>
                  <span className="text-white font-medium">{health.system.goroutines}</span>
                </div>
              )}
            </div>
          ) : (
            <p className="text-slate-400 text-center py-8">Unable to fetch system info</p>
          )}
        </div>
      </div>
    </div>
  );
}
