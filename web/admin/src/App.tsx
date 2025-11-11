import { useState } from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Navbar from './components/Navbar';
import Sidebar from './components/Sidebar';
import Dashboard from './pages/Dashboard';
import Services from './pages/Services';
import KVStore from './pages/KVStore';
import Health from './pages/Health';

function App() {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  return (
    <Router basename="/admin">
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
            </Routes>
          </main>
        </div>
      </div>
    </Router>
  );
}

export default App;
