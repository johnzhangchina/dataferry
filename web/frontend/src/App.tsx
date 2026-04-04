import { useState, useEffect } from 'react';
import { Routes, Route, Link, useNavigate } from 'react-router-dom';
import FlowList from './pages/FlowList';
import FlowEdit from './pages/FlowEdit';
import FlowLogs from './pages/FlowLogs';
import Login from './pages/Login';
import Logo from './Logo';

function App() {
  const [authState, setAuthState] = useState<'loading' | 'login' | 'ok'>('loading');

  const checkAuth = () => {
    fetch('/api/auth')
      .then(r => r.json())
      .then(data => {
        if (!data.required || data.authenticated) {
          setAuthState('ok');
        } else {
          setAuthState('login');
        }
      })
      .catch(() => setAuthState('ok'));
  };

  useEffect(() => { checkAuth(); }, []);

  if (authState === 'loading') {
    return <div className="app"><div className="empty"><p>加载中...</p></div></div>;
  }

  if (authState === 'login') {
    return (
      <div className="app">
        <Login onLogin={() => setAuthState('ok')} />
      </div>
    );
  }

  return (
    <div className="app">
      <AppHeader onLogout={() => setAuthState('login')} />
      <Routes>
        <Route path="/" element={<FlowList />} />
        <Route path="/flows/:id" element={<FlowEdit />} />
        <Route path="/flows/:id/logs" element={<FlowLogs />} />
      </Routes>
    </div>
  );
}

function AppHeader({ onLogout }: { onLogout: () => void }) {
  const navigate = useNavigate();

  const handleLogout = async () => {
    await fetch('/api/logout', { method: 'POST' });
    onLogout();
    navigate('/');
  };

  // Check if auth is required (to show logout button)
  const [authRequired, setAuthRequired] = useState(false);
  useEffect(() => {
    fetch('/api/auth').then(r => r.json()).then(d => setAuthRequired(d.required));
  }, []);

  return (
    <header className="header">
      <Link to="/" className="header-brand">
        <Logo size={28} />
        <span className="header-name">DataFerry</span>
      </Link>
      <span className="header-divider" />
      <span className="header-tagline">API 胶水平台</span>
      {authRequired && (
        <button
          className="btn btn-ghost btn-sm"
          style={{ marginLeft: 'auto' }}
          onClick={handleLogout}
        >
          退出
        </button>
      )}
    </header>
  );
}

export default App;
