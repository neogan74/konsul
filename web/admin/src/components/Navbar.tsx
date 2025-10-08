import { Menu } from 'lucide-react';

interface NavbarProps {
  onMenuClick: () => void;
}

export default function Navbar({ onMenuClick }: NavbarProps) {
  return (
    <nav className="bg-slate-800 border-b border-slate-700 px-6 py-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={onMenuClick}
            className="lg:hidden text-slate-400 hover:text-white transition-colors"
          >
            <Menu size={24} />
          </button>
          <h1 className="text-xl font-bold text-white">Konsul Admin</h1>
        </div>
        <div className="flex items-center gap-4">
          <div className="text-sm text-slate-400">
            Service Discovery & Configuration
          </div>
        </div>
      </div>
    </nav>
  );
}
