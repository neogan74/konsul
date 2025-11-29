import { useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import Navbar from './components/Navbar';
import Sidebar from './components/Sidebar';
import ProtectedRoute from './components/ProtectedRoute';
import Dashboard from './pages/Dashboard';
import Services from './pages/Services';
import KVStore from './pages/KVStore';
import Health from './pages/Health';
import Login from './pages/Login';
import APIKeys from './pages/APIKeys';
import { useAuth } from './contexts/AuthContext';

function App() {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-slate-400">Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <Router basename="/admin">
      <Routes>
        {/* Public route */}
        <Route
          path="/login"
          element={isAuthenticated ? <Navigate to="/" replace /> : <Login />}
        />

        {/* Protected routes */}
        <Route
          path="/*"
          element={
            <ProtectedRoute>
              <div className="min-h-screen bg-slate-900">
                <Navbar onMenuClick={() => setIsSidebarOpen(true)} />

                <div className="flex">
                  <Sidebar
                    isOpen={isSidebarOpen}
                    onClose={() => setIsSidebarOpen(false)}
                  />

                  <main className="flex-1 min-w-0">
                    <Routes>
                      <Route path="/" element={<Dashboard />} />
                      <Route path="/services" element={<Services />} />
                      <Route path="/kv" element={<KVStore />} />
                      <Route path="/health" element={<Health />} />
                      <Route path="/apikeys" element={<APIKeys />} />
                    </Routes>
                  </main>
                </div>
              </div>
            </ProtectedRoute>
          }
        />
      </Routes>
    </Router>
  );
}

export default App;
