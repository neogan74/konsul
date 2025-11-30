import { useState } from 'react';
import { Menu, User, LogOut, ChevronDown } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate } from 'react-router-dom';
import ConnectionStatus from './ConnectionStatus';

interface NavbarProps {
  onMenuClick: () => void;
}

export default function Navbar({ onMenuClick }: NavbarProps) {
  const { user, isAuthenticated, logout } = useAuth();
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

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
          {isAuthenticated && <ConnectionStatus />}
          <div className="text-sm text-slate-400 hidden md:block">
            Service Discovery & Configuration
          </div>

          {isAuthenticated && user && (
            <div className="relative">
              <button
                onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
                className="flex items-center gap-2 px-3 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors"
              >
                <User size={18} className="text-slate-300" />
                <span className="text-sm text-white">{user.username}</span>
                <ChevronDown size={16} className="text-slate-400" />
              </button>

              {isUserMenuOpen && (
                <>
                  <div
                    className="fixed inset-0 z-10"
                    onClick={() => setIsUserMenuOpen(false)}
                  />
                  <div className="absolute right-0 mt-2 w-64 bg-slate-800 border border-slate-700 rounded-lg shadow-xl z-20">
                    <div className="p-4 border-b border-slate-700">
                      <p className="text-sm font-medium text-white">{user.username}</p>
                      <p className="text-xs text-slate-400 mt-1">ID: {user.user_id}</p>
                      {user.roles && user.roles.length > 0 && (
                        <div className="flex flex-wrap gap-1 mt-2">
                          {user.roles.map((role, idx) => (
                            <span
                              key={idx}
                              className="px-2 py-0.5 bg-blue-900/30 text-blue-400 rounded text-xs"
                            >
                              {role}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                    <button
                      onClick={handleLogout}
                      className="w-full flex items-center gap-2 px-4 py-3 text-red-400 hover:bg-slate-700 transition-colors text-sm"
                    >
                      <LogOut size={16} />
                      Sign Out
                    </button>
                  </div>
                </>
              )}
            </div>
          )}
        </div>
      </div>
    </nav>
  );
}
