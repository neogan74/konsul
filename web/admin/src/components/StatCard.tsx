import type { LucideIcon } from 'lucide-react';

interface StatCardProps {
  title: string;
  value: string | number;
  icon: LucideIcon;
  trend?: {
    value: string;
    positive: boolean;
  };
  loading?: boolean;
}

export default function StatCard({ title, value, icon: Icon, trend, loading }: StatCardProps) {
  return (
    <div className="bg-slate-800 rounded-lg p-6 border border-slate-700 hover:border-slate-600 transition-colors">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-slate-400 text-sm font-medium">{title}</p>
          {loading ? (
            <div className="h-8 w-24 bg-slate-700 rounded animate-pulse mt-2" />
          ) : (
            <p className="text-3xl font-bold text-white mt-2">{value}</p>
          )}
          {trend && !loading && (
            <p
              className={`text-sm mt-2 ${
                trend.positive ? 'text-green-400' : 'text-red-400'
              }`}
            >
              {trend.value}
            </p>
          )}
        </div>
        <div className="bg-slate-700 p-3 rounded-lg">
          <Icon className="text-slate-300" size={24} />
        </div>
      </div>
    </div>
  );
}
