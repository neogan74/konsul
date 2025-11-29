import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Lock, User, Shield, Key } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';

export default function Login() {
  const [username, setUsername] = useState('');
  const [userId, setUserId] = useState('');
  const [roles, setRoles] = useState('admin');
  const [policies, setPolicies] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      const roleList = roles.split(',').map(r => r.trim()).filter(Boolean);
      const policyList = policies.split(',').map(p => p.trim()).filter(Boolean);

      await login(username, userId || username, roleList, policyList);
      navigate('/');
    } catch (err) {
      setError('Login failed. Please check your credentials and try again.');
      console.error('Login error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-900 flex items-center justify-center p-4">
      <div className="max-w-md w-full">
        {/* Logo/Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-blue-600 rounded-2xl mb-4">
            <Lock className="text-white" size={32} />
          </div>
          <h1 className="text-3xl font-bold text-white mb-2">Konsul Admin</h1>
          <p className="text-slate-400">Sign in to manage your cluster</p>
        </div>

        {/* Login Form */}
        <div className="bg-slate-800 rounded-lg border border-slate-700 p-8 shadow-xl">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Username */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                Username
              </label>
              <div className="relative">
                <User className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400" size={20} />
                <input
                  type="text"
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full pl-10 pr-4 py-3 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Enter username"
                />
              </div>
            </div>

            {/* User ID (optional) */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                User ID (optional)
              </label>
              <div className="relative">
                <Key className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400" size={20} />
                <input
                  type="text"
                  value={userId}
                  onChange={(e) => setUserId(e.target.value)}
                  className="w-full pl-10 pr-4 py-3 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Defaults to username"
                />
              </div>
            </div>

            {/* Roles */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                Roles (comma-separated)
              </label>
              <div className="relative">
                <Shield className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400" size={20} />
                <input
                  type="text"
                  required
                  value={roles}
                  onChange={(e) => setRoles(e.target.value)}
                  className="w-full pl-10 pr-4 py-3 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="admin, developer"
                />
              </div>
            </div>

            {/* Policies (optional) */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                Policies (comma-separated, optional)
              </label>
              <input
                type="text"
                value={policies}
                onChange={(e) => setPolicies(e.target.value)}
                className="w-full px-4 py-3 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="developer, readonly"
              />
            </div>

            {/* Error Message */}
            {error && (
              <div className="px-4 py-3 bg-red-900/20 border border-red-700 rounded-lg text-red-400 text-sm">
                {error}
              </div>
            )}

            {/* Submit Button */}
            <button
              type="submit"
              disabled={isLoading}
              className="w-full py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? 'Signing in...' : 'Sign In'}
            </button>
          </form>

          {/* Info Note */}
          <div className="mt-6 pt-6 border-t border-slate-700">
            <p className="text-xs text-slate-400 text-center">
              This login uses Konsul's JWT authentication system.
              Ensure authentication is enabled on the server.
            </p>
          </div>
        </div>

        {/* Development Note */}
        <div className="mt-4 text-center">
          <p className="text-xs text-slate-500">
            Development mode: Authentication may be disabled on the server
          </p>
        </div>
      </div>
    </div>
  );
}